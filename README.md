# Oblak

Oblak is a secure serverless function execution platform, similar in concept to what major cloud providers offer, but with a primary focus on security. Users upload Python function packages that pass through a multi-stage security analysis pipeline before being executed in isolated Firecracker microVMs.

## How It Works

Every function a user uploads is treated as untrusted code. Before it can ever be executed, it must pass through a layered security analysis pipeline. Only functions that clear all checks are promoted to an executable state. Execution then happens inside a Firecracker microVM with the jailer's privilege isolation, with no network access to your internal LAN.

### Function Lifecycle

```
User uploads .tar.gz / .zip
         │
         ▼
   Server (REST API)
   ├─ Stores archive in quarantine bucket (MinIO)
   ├─ Records function in DB (status: QUARANTINED)
   └─ Publishes message to analyzer queue (RabbitMQ)
         │
         ▼
   Analyzer (security pipeline)
   ├─ 1. Dependency audit    (pip-audit: known CVEs in requirements.txt)
   ├─ 2. Antivirus scan      (ClamAV: malware signatures)
   ├─ 3. Static analysis     (Semgrep: Python security rules)
   ├─    └─ LLM review       (Qwen 2.5 3B: judge each SAST finding)
   ├─ 4. Dynamic analysis    (gVisor sandbox: capture runtime syscall behavior)
   └─ 5. LLM behavioral judge (Qwen 2.5 3B: verdict on sandbox trace)
         │
         ├─ MALICIOUS → status: DETECTED, stays quarantined
         │
         └─ SAFE → status: VERIFIED, moved to functions bucket
                   └─ Message published to VM build queue
         │
         ▼
   VM Service (build phase)
   ├─ Installs Python dependencies into an ext4 drive image
   └─ Uploads prepared drive to MinIO (status: READY)
         │
         ▼
   VM Service (execution phase, triggered by invocation request)
   ├─ Boots Firecracker microVM via jailer
   ├─ Attaches drives (app: squashfs ro, deps: ext4 ro, tmp: ext4 rw)
   ├─ Communicates with in-VM agent over vsock
   └─ Streams stdout/stderr back; stores log in MinIO
```

## Components

| Component | Description |
|-----------|-------------|
| **Server** | REST API (Gin). Accepts uploads, authenticates requests, serves invocation history. |
| **Analyzer** | Security pipeline. Consumes from RabbitMQ, orchestrates all analysis stages. |
| **VM Service** | Manages Firecracker microVMs. Handles build and execute queues. |
| **Agent** | Runs inside each VM. Mounts drives, runs user code, streams output via vsock. |
| **CLI** | User-facing command-line tool (`configure`, `upload`, `list`, `details`). |
| **Admin CLI** | Operator tool for creating users and generating API keys. |

## Usage

### CLI

The CLI binary (`bin/cli`) is referred to as `oblak` below.

Configure it once with the server address and your credentials:

```bash
oblak configure
# Auth ID:  <your-auth-id>
# Secret:   <your-secret-key>
# Endpoint: http://localhost:8080
```

Upload a function:

```bash
oblak upload my-function/
```

Check the status of your functions:

```bash
oblak list
```

View the log for a specific invocation:

```bash
oblak details <invocation-id>
```

### Invoking a function

Once a function reaches `READY` status it can be invoked via HTTP. Unlike uploads, **invocation requires no authentication** — the function ID is the only credential needed, similar to how AWS Lambda function URLs work.

```
POST /execute/{functionId}
```

```bash
curl -X POST http://localhost:8080/execute/<function-id>
```

The response includes the invocation status. Stdout and stderr from the function are stored as a log and can be retrieved with `cli details` or by calling `GET /invocations/{invocationId}`.


## Function Format

Functions are directories containing a `handler.py` file that exports a `handle()` function. The CLI packages the directory automatically on upload.

```python
# handler.py
def handle():
    print("Hello from Oblak!")
```

If your function has dependencies, include a `requirements.txt` alongside `handler.py`. Dependencies are installed during the build phase and made available at runtime via the `/deps` mount.

```
my-function/
├── handler.py
└── requirements.txt   # optional
```

See `codeexamples/` for working examples.

## Documentation

- [Setup Guide](docs/setup.md) — prerequisites, installation steps, configuration, make commands
- [Architecture](docs/architecture.md) — component internals, storage, message queues, data model
- [Security Pipeline](docs/security.md) — analysis stages, sandboxing, authentication scheme
