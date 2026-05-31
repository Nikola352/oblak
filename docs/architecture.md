# Architecture

## System Overview

Oblak consists of three long-running services that communicate through RabbitMQ, plus a CLI for end users and an admin CLI for operators. All services share PostgreSQL (state) and MinIO (file storage).

```
┌─────────────────────────────────────────────────────────┐
│                     User / Operator                     │
│          CLI (upload/list/invoke)  Admin CLI            │
└───────────────────────┬─────────────────────────────────┘
                        │ HTTP (HMAC-SHA256 auth)
                        ▼
                ┌─────────────────┐
                │     Server      │  :8080
                │   (REST API)    │
                └────────┬────────┘
                         │ RabbitMQ (analyzer exchange)
              ┌──────────▼──────────┐
              │      Analyzer       │
              │ (security pipeline) │
              └──────────┬──────────┘
                         │ RabbitMQ (vm exchange)
              ┌──────────▼──────────┐
              │     VM Service      │
              │  (build + execute)  │
              └──────────┬──────────┘
                         │ vsock
              ┌──────────▼────────────┐
              │    Agent (in-VM)      │
              │ (Firecracker microVM) │
              └───────────────────────┘

Shared infrastructure:
  PostgreSQL  — function and invocation state
  MinIO       — archive, drive, and log storage
  RabbitMQ    — inter-service messaging
```

---

## Server

**Entry point:** `cmd/server` | **Port:** 8080 | **Framework:** Gin

The server is the only component exposed externally. It handles all user-facing operations.

### Responsibilities

- Accept function archive uploads (`.tar.gz`, `.tar.xz`, `.zip`) via multipart form
- Store uploaded archives in the MinIO quarantine bucket
- Record new functions in PostgreSQL with status `QUARANTINED`
- Publish a `FunctionMessage` to the RabbitMQ `analyzer` exchange
- Serve the function list, invocation history, and invocation logs for authenticated users
- Proxy invocation log content from MinIO to the client

### Key packages

| Package | Role |
|---------|------|
| `internal/server/handler/` | HTTP handlers: upload, list, invocations, invocation detail |
| `internal/server/middleware/` | HMAC-SHA256 authentication middleware |
| `internal/server/database/` | PostgreSQL queries |
| `internal/server/filestore/` | MinIO client (upload to quarantine, download logs) |
| `internal/server/events/` | In-process pub/sub for QuarantineEvent and ExtractionEvent |
| `internal/server/analyzerpublisher/` | RabbitMQ publisher to the analyzer exchange |
| `internal/server/authkey/` | Encrypted auth key storage and retrieval |
| `internal/server/invocations/` | Invocation log management |

### Authentication

See [Security — API Authentication](security.md#api-authentication).

---

## Analyzer

**Entry point:** `cmd/analyzer` | **Build target:** linux/amd64

The analyzer is the security gatekeeper. It consumes messages from the `analyzer` RabbitMQ queue and runs a pipeline of checks against the uploaded archive. It never executes user code with host privileges.

### Pipeline

For each message received, the orchestrator (`internal/analyzer/orchestrator/`) runs these stages in order:

**1. Extraction**
Downloads the archive from the quarantine bucket, extracts it to `/tmp/quarantine/<function-id>/extracted/`. Both the archive and the extracted directory are deleted when the pipeline finishes.

**2. Dependency audit** (`internal/analyzer/audit/`)
If `requirements.txt` is present, runs `pip-audit -r requirements.txt`. Any known CVE found in a declared dependency causes immediate rejection. Functions with no `requirements.txt` pass this stage unconditionally.

**3. Antivirus scan** (`internal/analyzer/av/`)
Runs a ClamAV scan on the extracted directory over TCP. A positive malware signature match causes immediate rejection.

**4. Static analysis** (`internal/analyzer/sast/`)
Runs Semgrep with Python security rules against the extracted code. Each finding is individually reviewed by the LLM judge (see below). If the judge marks any single finding as malicious, the function is rejected.

**5. Dynamic analysis / detonation** (`internal/analyzer/dast/`)
Runs the user code inside a Docker container using the `runsc` (gVisor) runtime. gVisor intercepts all syscalls and writes a structured strace-style log to `/tmp/gvisor-logs/`. The container uses the `python:3.11-alpine` image with memory limited to 256 MB. Stdout, stderr, and the gVisor syscall log are captured and written as a JSON report to `JSON_REPORT_OUTPUT_PATH/<function-id>`.

**6. LLM behavioral judge** (`internal/analyzer/llm/`)
The JSON report from the DAST stage is sent to the Ollama API (model: `qwen2.5:3b`). The model classifies the function's runtime behavior as `SAFE` or `MALICIOUS`, with a confidence score and summary. A `MALICIOUS` verdict causes rejection.

### Verdict outcomes

| Verdict | DB Status | File location |
|---------|-----------|---------------|
| SAFE | `VERIFIED` | Moved from quarantine bucket to functions bucket |
| MALICIOUS | `DETECTED` | Remains in quarantine bucket |
| FAILURE (pipeline error) | `FAILED` | Remains in quarantine bucket |

On a SAFE verdict, the analyzer publishes a `FunctionMessage` to the `vm` RabbitMQ exchange to trigger environment preparation.

---

## VM Service

**Entry point:** `cmd/vm`

The VM service manages the full lifecycle of Firecracker microVMs. It processes two queues from RabbitMQ: `build` (prepare a Python environment) and `execute` (run a function).

### Build phase

When a function is verified, the analyzer publishes to the `vm` exchange. The VM service:

1. Downloads the function archive from the functions bucket
2. Extracts `requirements.txt`
3. Creates an ext4 drive image and installs the Python dependencies into it using `pip install`
4. Uploads the prepared drive image to the `oblak-drives` MinIO bucket
5. Updates the function status to `READY`

Concurrency is limited by `MAX_CONCURRENT_BUILDS` (default 3).

### Execute phase

Invocation requests (triggered via the server API) are enqueued to the `execute` queue. The VM service:

1. Downloads the app squashfs and deps ext4 drive from MinIO
2. Creates a fresh ext4 drive for `/tmp` (scratch space)
3. Boots a Firecracker microVM via the jailer with three drives attached
4. Communicates with the in-VM agent over vsock (port 6000)
5. Sends an `ExecJob` command and streams the output back
6. Stores the full log in the `oblak-logs` MinIO bucket (30-day retention)
7. Updates the invocation status (`DONE` or `FAILED`)

Concurrency is limited by `MAX_CONCURRENT_EXECUTES` and `MAX_CONCURRENT_VMS` (default 20 each).

Failed tasks are routed to a dead-letter queue (`build_dlq`) for inspection.

### Key packages

| Package | Role |
|---------|------|
| `internal/vm/queue/` | RabbitMQ consumer and message routing (Watermill) |
| `internal/vm/service/` | EnvironmentPrepareService, ExecutionService |
| `internal/vm/vm/` | Firecracker VM lifecycle, environment/execution runners |
| `internal/vm/config/` | Queue names, AMQP URI, concurrency settings |

---

## Agent

**Entry point:** `cmd/agent` | **Build target:** linux/amd64 | **Embedded in:** `rootfs.squashfs`

The agent runs as PID 1 (or near it) inside each Firecracker VM. It listens on vsock port 6000 for commands from the host VM service.

### Build job (`BuildJob`)

1. Mounts the deps drive (`/dev/vdb`) as ext4 read-write at `/deps`
2. Runs `pip install -r /tmp/requirements.txt --target /deps` to populate the drive
3. Unmounts and syncs to flush the filesystem before signaling done

### Exec job (`ExecJob`)

1. Mounts the app drive (`/dev/vdb`) as squashfs read-only at `/app`
2. Mounts the deps drive (`/dev/vdc`) as ext4 read-only at `/deps`
3. Mounts the tmp drive (`/dev/vdd`) as ext4 read-write at `/tmp`
4. Runs `python3 -c "import handler; handler.handle('hello!')"` with `PYTHONPATH=/deps`
5. Streams stdout and stderr line-by-line back to the host via vsock
6. On completion: kills any remaining user processes (entire process group), unmounts all drives, calls `sync`, sends `Done(exit_code)` to host

### Protocol (`internal/agentproto/`)

JSON messages over vsock:

| Direction | Message type | Content |
|-----------|-------------|---------|
| Host → Agent | `build` / `exec` | Job type |
| Agent → Host | `output` | `{stream: "stdout"\|"stderr"\|"system", data: "..."}` |
| Agent → Host | `done` | `{exit_code: N}` |
| Agent → Host | `error` | `{message: "..."}` |

---

## CLI

**Entry point:** `cmd/cli` | **Framework:** Cobra | **Config:** `~/.oblak/config.yaml`

| Command | Description |
|---------|-------------|
| `configure` | Set the server endpoint and auth credentials |
| `upload <archive>` | Upload a function archive to the server |
| `list` | List all functions for the authenticated user |
| `details <invocation-id>` | Fetch and print invocation logs |

The CLI signs every request using the same HMAC-SHA256 scheme as any other client.

---

## Admin CLI

**Entry point:** `cmd/admin`

Operator-only tool. Connects directly to the database using `DATABASE_URL` from the environment.

| Command | Description |
|---------|-------------|
| `create-user --username <name> --email <email>` | Create a new user record |
| `create-key --username <name>` | Generate and print an API key pair for the user |

Auth key secret values are encrypted with the `KEY_ENCRYPTION_KEY` before being stored.

---

## Data Model

```sql
users (
    user_id   UUID PRIMARY KEY,
    username  TEXT UNIQUE,
    email     TEXT
)

auth_keys (
    auth_id    TEXT PRIMARY KEY,   -- credential ID, sent in Authorization header
    user_id    UUID → users,
    secret_key TEXT                -- encrypted with KEY_ENCRYPTION_KEY
)

functions (
    function_id  UUID PRIMARY KEY,
    user_id      UUID → users,
    status       TEXT,             -- see lifecycle below
    archive_path TEXT,             -- MinIO object path (quarantine or functions bucket)
    drive_path   TEXT              -- MinIO object path (drives bucket, set after build)
)

invocations (
    invocation_id   UUID PRIMARY KEY,
    function_id     UUID → functions,
    status          TEXT,          -- PENDING | EXECUTING | DONE | FAILED
    invocation_time TIMESTAMPTZ,
    end_time        TIMESTAMPTZ,
    log_path        TEXT           -- MinIO object path (logs bucket)
)
```

**Function status transitions:**

```
QUARANTINED → SCANNING → DETECTED (malicious)
                       → VERIFIED → PREPARING_ENVIRONMENT → READY → (invocable)
                       → FAILED   (pipeline error)
```

---

## Storage (MinIO)

| Bucket | Content | Written by | Read by |
|--------|---------|------------|---------|
| `oblak-quarantine` | Uploaded archives (unverified) | Server | Analyzer |
| `oblak-functions` | Verified archives | Analyzer | VM service |
| `oblak-drives` | Prepared Python environment drives (ext4) | VM service | VM service |
| `oblak-logs` | Invocation stdout/stderr logs | VM service | Server |

Logs have a 30-day retention policy.

---

## Message Queues (RabbitMQ)

### Analyzer exchange

| Exchange | Type | Publisher | Consumer |
|----------|------|-----------|----------|
| `analyzer` | fanout | Server (on upload) | Analyzer |

Payload: `{ "function_id": "<uuid>", "path": "<minio-path>", "bucket": "oblak-quarantine" }`

### VM exchange

| Exchange | Type | Publisher | Consumer |
|----------|------|-----------|----------|
| `vm` | fanout | Analyzer (on SAFE verdict) | VM service |

Payload: same `FunctionMessage` structure.

### VM internal queues

| Queue | Bound to | Purpose |
|-------|----------|---------|
| `build` | `vm` exchange | Trigger environment preparation |
| `execute` | (direct) | Trigger function execution |
| `build_dlq` | — | Dead-letter queue for failed builds |

The VM service uses [Watermill](https://watermill.io/) over AMQP for message handling.

---

## VM Infrastructure

### Firecracker and jailer

Each VM is launched via `jailer`, a setuid-root binary that:
1. Creates a chroot under `/srv/jailer/<vm-id>/`
2. Hard-links the kernel and rootfs images into the chroot
3. Sets up cgroups and drops privileges to `firecracker` (uid/gid 900)
4. Exec's `firecracker` inside the jail

The VM binary holds `cap_net_admin`, `cap_sys_admin`, `cap_dac_override`, and `cap_kill` (required to configure CNI networking and manage the jailer chroot).

### Drive layout per VM

```
/dev/vda  — rootfs.squashfs  (read-only, shared via hard-link)
/dev/vdb  — app.squashfs     (read-only, user's function code)
/dev/vdc  — deps.ext4        (read-only, pre-installed dependencies)
/dev/vdd  — tmp.ext4         (read-write, scratch space)
```

### Networking

VMs get an IP in `192.168.127.0/24` via CNI (ptp + tc-redirect-tap). iptables rules on the host:
- Block all VM → RFC 1918 traffic (VMs cannot reach your LAN or other VMs)
- Allow VM → internet (for pip installs during the build phase)
- Allow established/related return traffic
