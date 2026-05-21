#!/usr/bin/env bash
# Builds the agent binary and injects it into the rootfs squashfs image.
# Requires: squashfs-tools (unsquashfs, mksquashfs).

set -euo pipefail

ROOTFS_SRC="./deployment/firecracker/ubuntu-24.04.squashfs"
ROOTFS_DEST="./deployment/firecracker/rootfs.squashfs"
AGENT_BIN="./bin/agent"

mkdir -p "$(dirname "$AGENT_BIN")"

echo "==> Building agent binary for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$AGENT_BIN" ./cmd/agent

echo "==> Extracting, injecting, and repacking rootfs..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pip_whl_url=$(python3 -c "
import urllib.request, json
with urllib.request.urlopen('https://pypi.org/pypi/pip/json') as r:
    for f in json.load(r)['urls']:
        if f['filename'].endswith('-py3-none-any.whl'):
            print(f['url']); break
")

fakeroot -- bash -c "
  set -euo pipefail
  unsquashfs -d '$TMPDIR/rootfs' '$ROOTFS_SRC'
  mkdir -p '$TMPDIR/rootfs/app' '$TMPDIR/rootfs/deps' '$TMPDIR/rootfs/tmp'
  curl -fsSL '$pip_whl_url' -o '$TMPDIR/pip.whl'
  unzip -q '$TMPDIR/pip.whl' -d '$TMPDIR/rootfs/usr/lib/python3/dist-packages/'
  cp '$AGENT_BIN' '$TMPDIR/rootfs/usr/local/bin/agent-runner'
  chmod 0755 '$TMPDIR/rootfs/usr/local/bin/agent-runner'
  mksquashfs '$TMPDIR/rootfs' '$TMPDIR/rootfs.squashfs' -noappend -comp xz
"
mv "$TMPDIR/rootfs.squashfs" "$ROOTFS_DEST"

echo ""
echo "==> Done! Rootfs ready at: $ROOTFS_DEST"
echo "    The agent binary is at /usr/local/bin/agent-runner inside the image."
