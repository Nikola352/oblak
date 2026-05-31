# Security

This document covers the threat model, analysis pipeline, sandboxing strategy, and API authentication scheme.

## Threat Model

Oblak assumes uploaded function code is untrusted. The goals are:

- Detect and reject malicious code **before** it is ever executed in production
- Contain any code that does execute within a hard isolation boundary
- Prevent executed code from reaching the host or internal network
- Prevent impersonation of users at the API level

---

## Analysis Pipeline

Every function passes through six sequential security checks before it is allowed to execute. Failure at any stage causes the function to be rejected with status `DETECTED` or `FAILED`; it is never moved out of the quarantine bucket.

### Stage 1 — Dependency Audit

**Tool:** [pip-audit](https://github.com/pypa/pip-audit)

If the function archive contains a `requirements.txt`, pip-audit checks every declared dependency against the OSV and PyPI advisory databases for known CVEs. A single flagged dependency fails the audit.

Functions with no `requirements.txt` pass this stage unconditionally.

### Stage 2 — Antivirus Scan

**Tool:** [ClamAV](https://www.clamav.net/) (multi-threaded, via TCP socket)

The extracted function directory is scanned against ClamAV's malware signature database. This catches known malware samples, trojans, and exploit payloads that may be included alongside the Python code.

### Stage 3 — Static Analysis (SAST)

**Tool:** [Semgrep](https://semgrep.dev/) with Python security rules

Semgrep performs pattern-based static analysis on the Python source. It flags:
- Dangerous built-in usage (`eval`, `exec`, `__import__`, subprocess with `shell=True`, etc.)
- Hardcoded credentials
- Insecure network operations
- Known vulnerability patterns

Each Semgrep finding is not automatically a rejection — instead, it is forwarded to the LLM judge (see below) for contextual evaluation. This prevents false positives from blocking legitimate code.

### Stage 4 — LLM SAST Review

**Model:** Qwen 2.5 3B (via [Ollama](https://ollama.com/))

For each Semgrep finding, the orchestrator calls the LLM judge with:
- The Semgrep rule message (describing what was flagged)
- The sanitized source code snippet containing the finding

The model returns a verdict: `MALICIOUS`, `SAFE`, or `FAILURE`. If any single finding is classified `MALICIOUS`, the function is rejected immediately. This design reduces false positives from SAST while still catching genuinely dangerous patterns.

The code sanitizer (`internal/analyzer/sanitizer/`) strips comments and normalizes whitespace before sending code to the LLM, to avoid prompt injection through carefully crafted code comments.

### Stage 5 — Dynamic Analysis (DAST)

**Runtime:** [gVisor](https://gvisor.dev/) (`runsc`)

The function code is actually executed in a sandboxed Docker container using the `runsc` runtime. gVisor sits between the container and the host kernel, intercepting every syscall and enforcing a restricted kernel surface. It logs all intercepted calls to `/tmp/gvisor-logs/`.

Container configuration:
- Image: `python:3.11-alpine`
- Memory limit: 256 MB
- Network: enabled (for dependency import), but isolated from the host's internal network via Docker's bridge
- Entry: installs `requirements.txt` if present, then calls `handler.handle()`

The stage collects stdout, stderr, and the gVisor syscall trace, and writes them as a structured JSON report.

> The gVisor container provides a defense-in-depth layer for this analysis stage, but the primary execution isolation is the Firecracker microVM used for production invocations. These are separate environments for different purposes.

### Stage 6 — LLM Behavioral Judge

**Model:** Qwen 2.5 3B (via Ollama)

The JSON report from the DAST stage is sent to the LLM with a prompt asking it to classify the function's runtime behavior. The model returns:

```json
{
  "verdict": "SAFE" | "MALICIOUS",
  "summary": "...",
  "confidence_score": 0-100
}
```

The judge looks for behavioral signals such as:
- Unexpected network connections
- Filesystem access outside expected paths
- Spawning child processes
- Privilege escalation attempts
- Cryptomining syscall patterns

A `MALICIOUS` verdict at this stage rejects the function.

---

## Execution Sandboxing (Firecracker)

Production function invocations run inside Firecracker microVMs. Firecracker is a KVM-based VMM designed for minimal attack surface; it implements only the devices necessary for running workloads (virtio-net, virtio-blk, serial console) with no legacy hardware emulation.

### Jailer

The `jailer` binary is setuid root. It performs all privileged setup (chroot, cgroups, device nodes) then drops to the `firecracker` user (uid/gid 900) before exec'ing Firecracker. The Go VM service does not run as root; it holds only specific Linux capabilities:

| Capability | Purpose |
|------------|---------|
| `cap_net_admin` | Configure CNI network namespaces |
| `cap_sys_admin` | Enter VM network namespaces |
| `cap_dac_override` | Hard-link images into jailer chroot |
| `cap_kill` | Kill VM processes on shutdown |

### Drive isolation

Each invocation gets its own set of drive images:

| Drive | Filesystem | Mount flags | Content |
|-------|-----------|-------------|---------|
| `vdb` (app) | squashfs | `MS_RDONLY \| MS_NOSUID \| MS_NOEXEC \| MS_NODEV` | User function code |
| `vdc` (deps) | ext4 | `MS_RDONLY \| MS_NOSUID \| MS_NODEV` | Pre-installed dependencies |
| `vdd` (tmp) | ext4 | `MS_NOSUID \| MS_NOEXEC \| MS_NODEV` | Scratch space |

- The application code is mounted read-only. User code cannot modify itself.
- `MS_NOSUID` prevents setuid binaries from escalating privileges.
- `MS_NOEXEC` on `/tmp` prevents user code from writing and executing binaries.
- `MS_NODEV` prevents device file access on mounted filesystems.

The rootfs (`vda`) is a read-only squashfs shared via hard-link into the jailer chroot.

### Network isolation

VMs receive an IP in `192.168.127.0/24`. Host iptables rules enforce:

1. **Block** VM → all RFC 1918 ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) — VMs cannot reach your internal network or other VMs
2. **Allow** VM → internet (for any outbound the function needs)
3. **Allow** established/related return traffic

These rules are applied before any per-VM rules, using iptables first-match-wins ordering. The CNI `firewall` plugin (which inserts per-VM ACCEPT rules dynamically) is deliberately excluded from the CNI config to preserve rule ordering control.

### VM-host communication

The only channel between the VM and the host is vsock (port 6000). The agent inside the VM communicates using a simple JSON protocol (see [Architecture — Agent](architecture.md#agent)). There is no SSH, no shared filesystem, no other socket. The host can send a job command and read output; it cannot be reached by the VM.

---

## API Authentication

All API endpoints (except `/health`) require HMAC-SHA256 request signing.

### Signing scheme

The signature is computed as:

```
canonical_string = METHOD + "_" + PATH + "_" + SHA256(body_hex) + "_" + RFC3339_timestamp
intermediate    = SHA256(canonical_string)
signature       = HMAC-SHA256(key=secret_key, msg=intermediate)
```

The `Authorization` header format:

```
Authorization: HMAC-SHA256 Credential=<auth-id>, Signature=<hex-signature>
X-Date: <RFC3339 timestamp>
```

The server:
1. Parses `auth-id` from the `Credential` field
2. Looks up the corresponding encrypted secret key and decrypts it
3. Reads the request body and computes the expected signature
4. Uses `hmac.Equal` (constant-time comparison) to verify
5. Rejects requests where `|now - X-Date| > 15 minutes` to prevent replay attacks

### Auth key storage

Secret keys are encrypted with `KEY_ENCRYPTION_KEY` (a 64-char hex value, 256-bit AES key) before being stored in PostgreSQL. The encryption uses the `authkey` package. Plain-text secret keys are never stored.

The `admin create-key` command generates a new key pair and prints the plain-text secret exactly once. It is not recoverable.
