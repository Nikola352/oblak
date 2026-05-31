# Setup Guide

This guide walks through setting up Oblak from scratch on a Linux/amd64 host. KVM hardware virtualization is required for Firecracker.

## Prerequisites

Install the following before continuing. Some must be at specific paths.

### Required system packages

```bash
sudo apt install squashfs-tools fakeroot unzip curl wget git python3 python3-venv
```

### Go

Go 1.25 or newer. Install from [go.dev/dl](https://go.dev/dl) and ensure `go` is on your `PATH`.

### Firecracker and jailer

Both binaries must be at `/usr/local/bin/`:

```bash
# Download the latest release from GitHub
ARCH="$(uname -m)"
RELEASE=$(curl -fsSL https://api.github.com/repos/firecracker-microvm/firecracker/releases/latest | grep tag_name | cut -d'"' -f4)
TMP=$(mktemp -d)
curl -fsSL "https://github.com/firecracker-microvm/firecracker/releases/download/${RELEASE}/firecracker-${RELEASE}-${ARCH}.tgz" \
    | tar -xz -C "$TMP"
sudo cp "$TMP/release-${RELEASE}-${ARCH}/firecracker-${RELEASE}-${ARCH}" /usr/local/bin/firecracker
sudo cp "$TMP/release-${RELEASE}-${ARCH}/jailer-${RELEASE}-${ARCH}"     /usr/local/bin/jailer
sudo chmod +x /usr/local/bin/firecracker /usr/local/bin/jailer
```

> The `host-setup.sh` script expects `jailer` at `/usr/local/bin/jailer`. The path is hardcoded.

### Docker

Required for the analyzer's dynamic analysis (DAST) stage. Install Docker Engine and ensure your user can run `docker` commands (i.e. is in the `docker` group).

---

## Setup Steps

Follow these steps **in order**.

### 1. Clone the repository

```bash
git clone <repo-url> oblak
cd oblak
```

### 2. Download VM kernel and base rootfs

Run the asset download script from inside `deployment/`:

```bash
cd deployment
sudo bash download-assets.sh
cd ..
```

This downloads `vmlinux-6.1.155` and the Ubuntu 24.04 squashfs base image into `deployment/firecracker/`. These are the base assets that `build-rootfs.sh` will inject the agent binary into.

### 3. Host setup

Creates the required users, groups, and directories on the host. Run once; safe to re-run.

**Development** (adds your own user to the `oblak` group):
```bash
sudo bash deployment/host-setup.sh $USER
```

**Production** (creates a dedicated `oblak` service user):
```bash
sudo bash deployment/host-setup.sh
```

After running in development mode, either log out and back in or run `newgrp oblak` in your current shell so the group membership takes effect.

What it sets up:
- `firecracker` system user/group (uid/gid 900) — the identity that jailed VMs run as
- `oblak` group — the group that owns `/srv/jailer`
- `/srv/jailer/` — chroot base directory for per-VM jails (root:oblak, 775)
- `/srv/jailer/drives/` — staging area for drive images with setgid bit (firecracker group, 2770)
- `/srv/firecracker/` — stores the kernel and rootfs image (oblak:oblak, 755)
- Sets the `jailer` binary setuid root

### 4. Network setup

Configures CNI plugins and iptables rules for VM networking. Must be run as root; re-run after reboots (iptables rules are not persisted).

```bash
sudo bash deployment/network-setup.sh
```

What it sets up:
- Downloads CNI plugins (v1.9.1) to `/opt/cni/bin/`
- Builds and installs `tc-redirect-tap` to `/opt/cni/bin/`
- Sets `cap_net_admin,cap_sys_admin` on `tc-redirect-tap` and `ptp`
- Writes `/etc/cni/conf.d/oblak.conflist` (ptp + tc-redirect-tap, subnet `192.168.127.0/24`)
- Creates `/var/run/netns` and `/var/lib/cni` (owned by your user)
- Enables IP forwarding and persists it via `/etc/sysctl.d/`
- Adds iptables FORWARD rules: blocks VM traffic to RFC 1918 ranges, allows internet egress, allows established return traffic

> Network rules reset on reboot. Run `sudo bash deployment/network-setup.sh` again after each reboot (or use `iptables-persistent`).

### 5. Set up analyzer dependencies

Run these three scripts in order. All must be run from the repo root.

**5a. Create the Python virtual environment**

```bash
bash deployment/prepare_venv.sh
```

Creates `.venv/` and installs `semgrep` and `pip-audit` into it. The default `.env` points `AUDITOR_PATH` and `SAST_PATH` at the binaries inside this venv.

**5b. Pull the Alpine Docker image used by the DAST sandbox**

```bash
bash deployment/pull_alpine_environment.sh
```

Pulls `python:3.11-alpine` and does a test run inside a gVisor container to confirm the sandbox image is ready.

**5c. Install gVisor and configure the Docker runtime**

```bash
bash deployment/prepare_gvisor_runtime.sh
```

- Downloads `runsc` and installs it at `/usr/local/bin/runsc`
- Creates `/tmp/gvisor-logs/` for strace output
- Writes the `runsc` runtime entry into `/etc/docker/daemon.json`
- Restarts Docker

### 6. Start infrastructure services

```bash
docker compose -f deployment/docker-compose.yml up -d
```

This starts:
| Service | Port(s) | Credentials |
|---------|---------|-------------|
| PostgreSQL 17 | 5433 | postgres / postgres |
| MinIO | 9000 (API), 9001 (UI) | minioadmin / minioadmin |
| RabbitMQ 4.0 | 5672, 15672 (UI) | admin / admin |
| ClamAV | 3310 | — |
| Ollama (Qwen 2.5 3B) | 11434 | — |

> Ollama pulls the `qwen2.5:3b` model on first start. This may take a few minutes.

ClamAV also downloads its virus database on first start. Wait until it's ready before running the analyzer:

```bash
docker logs -f $(docker ps -qf name=clamav)
# Wait for: "Listening daemon: PID: ..."
```

### 7. Configure environment variables

Copy the example env file and set the paths to match your environment:

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:

```dotenv
# Analyzer tool paths — adjust to match your venv location
AUDITOR_PATH=/path/to/oblak/.venv/bin/pip-audit
SAST_PATH=/path/to/oblak/.venv/bin/semgrep

# A 64-character hex string used to encrypt stored auth keys
KEY_ENCRYPTION_KEY=<64 hex chars>
```

All other defaults work with the `docker-compose.yml` configuration as-is.

Full variable reference: see [Environment Variables](#environment-variables) below.

### 8. Run database migrations

```bash
psql postgres://postgres:postgres@localhost:5433/oblak -f migrations/001_initial.sql
```

(Run each file in `migrations/` in order if there are multiple.)

### 9. Build the services

```bash
make build
```

This compiles `bin/server`, `bin/vm` (with required Linux capabilities), and `bin/analyzer`.

### 10. Build the VM rootfs

```bash
make deployment/firecracker/rootfs.squashfs
```

This calls `deployment/build-rootfs.sh`, which:
1. Cross-compiles the agent binary for `linux/amd64`
2. Unpacks the Ubuntu 24.04 base squashfs
3. Injects the agent binary at `/usr/local/bin/agent-runner`
4. Bootstraps pip inside the image
5. Repacks as `deployment/firecracker/rootfs.squashfs`

The rootfs is rebuilt automatically by `make` only when agent source files change.

### 11. Install VM images

```bash
sudo make install
```

Copies `vmlinux-6.1.155` and `rootfs.squashfs` into `/srv/firecracker/`.

### 12. Create a user and API key

Use the admin CLI to create a user and issue an API key:

```bash
./bin/admin create-user --username alice --email alice@example.com
./bin/admin create-key --username alice
```

The `create-key` command prints the auth ID and secret key. Keep these — they are not recoverable. You'll need them to configure the CLI.

### 13. Start the services

Run all three services in a single terminal with prefixed output:

```bash
make run
```

Or run each in a separate terminal:

```bash
make run-server
make run-vm
make run-analyzer
```

### 14. Configure the CLI

```bash
./bin/cli configure
```

Enter the server endpoint (default `http://localhost:8080`) and the auth ID and secret key from step 12. Config is saved to `~/.oblak/config.yaml`.

---

## Make Targets Reference

| Target | Description |
|--------|-------------|
| `make build` | Compile all three service binaries into `bin/` |
| `make build-server` | Compile only the server |
| `make build-vm` | Compile the VM service and set required Linux capabilities |
| `make build-analyzer` | Compile only the analyzer |
| `make deployment/firecracker/rootfs.squashfs` | Build the VM rootfs image (agent injected) |
| `make install` | Copy kernel and rootfs to `/srv/firecracker/` (requires sudo) |
| `make run` | Build everything and run all three services in one terminal |
| `make run-server` | Build and run the server |
| `make run-vm` | Build, install images, and run the VM service |
| `make run-analyzer` | Build and run the analyzer |
| `make clean` | Remove compiled binaries and the rootfs image |

---

## Environment Variables

All services load from a `.env` file in the working directory (via godotenv). Variables can also be set in the environment.

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `debug` | Gin mode (`debug` or `release`) |
| `KEY_ENCRYPTION_KEY` | — | 64-char hex string; encrypts stored auth keys |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `MINIO_ENDPOINT` | — | MinIO host:port |
| `MINIO_ACCESS_KEY` | — | MinIO access key |
| `MINIO_SECRET_KEY` | — | MinIO secret key |
| `MINIO_USE_SSL` | `false` | Use TLS for MinIO connections |

### Analyzer

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API base URL |
| `AI_MODEL_NAME` | `qwen2.5:3b` | Ollama model to use for LLM judgments |
| `ANTIVIRUS_URL` | `tcp://localhost:3310` | ClamAV TCP socket address |
| `AUDITOR_PATH` | — | Absolute path to `pip-audit` binary |
| `SAST_PATH` | — | Absolute path to `semgrep` binary |
| `JSON_REPORT_OUTPUT_PATH` | `/tmp/reports/` | Directory for DAST JSON reports |
| `SEMGREP_APP_TOKEN` | — | Semgrep App token (for rule access) |

### VM Service

| Variable | Default | Description |
|----------|---------|-------------|
| `AMQP_URI` | — | RabbitMQ connection URI |
| `AMQP_VM_EXCHANGE_NAME` | `vm` | Exchange name for VM messages |
| `BUILD_QUEUE_NAME` | `build` | Queue for environment build tasks |
| `BUILD_DLQ_NAME` | `build_dlq` | Dead-letter queue for failed builds |
| `EXECUTE_QUEUE_NAME` | `execute` | Queue for function execution tasks |
| `MAX_CONCURRENT_BUILDS` | `3` | Max parallel environment builds |
| `MAX_CONCURRENT_EXECUTES` | `20` | Max parallel function executions |
| `MAX_CONCURRENT_VMS` | `20` | Overall VM concurrency cap |

Shared variables (`DATABASE_URL`, `MINIO_*`) apply to the VM service as well.
