#!/usr/bin/env bash
# One-time host setup for running the oblak server with the Firecracker jailer.
#
# Usage:
#   sudo ./deployment/host-setup.sh              # production: creates oblak service user
#   sudo ./deployment/host-setup.sh <username>   # development: adds <username> to oblak group
#
# Safe to re-run.
set -euo pipefail

[[ $EUID -eq 0 ]] || { echo "error: run as root: sudo $0 [dev-username]" >&2; exit 1; }

DEV_USER="${1:-}"

FIRECRACKER_UID=900
FIRECRACKER_GID=900
JAILER_BIN=/usr/local/bin/jailer
JAILER_BASE=/srv/jailer
IMAGE_DIR=/srv/firecracker

# ── Users & Groups ─────────────────────────────────────────────────────────
#
# Two identities are involved:
#
#   firecracker:   unprivileged user that each jailed VM runs as (uid/gid 900).
#                  Hardcoded in JailerCfg. Must exist on the host so the jailer
#                  can look it up by UID, but it never logs in or owns files.
#
#   oblak group:   the group that owns /srv/jailer. The Go service runs as a
#                  member of this group so it can create per-VM chroot dirs and
#                  hard-link images into them. In dev, add your personal user to
#                  this group. In prod, create a dedicated oblak service user.

if ! getent group firecracker >/dev/null 2>&1; then
    groupadd -g "$FIRECRACKER_GID" firecracker
fi
if ! id firecracker >/dev/null 2>&1; then
    useradd -r -u "$FIRECRACKER_UID" -g firecracker \
        -s /sbin/nologin -M -d "$JAILER_BASE" firecracker
fi

groupadd -f oblak

if [[ -n "$DEV_USER" ]]; then
    usermod -aG oblak "$DEV_USER"
    echo "note: '$DEV_USER' added to oblak group — log out and back in, or run: newgrp oblak"
else
    if ! id oblak >/dev/null 2>&1; then
        useradd -r -g oblak -s /sbin/nologin -M -d "$JAILER_BASE" oblak
    fi
    usermod -aG oblak oblak
fi

# ── Filesystem ─────────────────────────────────────────────────────────────

# Jailer chroot base. The Go service (oblak group member) creates per-VM
# subdirs here at runtime. The jailer (setuid root) then takes over each subdir.
mkdir -p "$JAILER_BASE"
chown root:oblak "$JAILER_BASE"
chmod 775 "$JAILER_BASE"

# Kernel and rootfs image store.
# Owned by oblak because Linux protected_hardlinks (on by default since kernel
# 3.6) prevents unprivileged users from hard-linking files they don't own or
# lack read+write access to. The Go service hard-links images from here into
# each VM's chroot dir (NaiveChrootStrategy), so it must own these files.
# Images are installed here by `make install` (see Makefile), not by this script.
mkdir -p "$IMAGE_DIR"
chown oblak:oblak "$IMAGE_DIR"
chmod 755 "$IMAGE_DIR"

# ── Jailer setuid ──────────────────────────────────────────────────────────
#
# The jailer binary is setuid root so the Go service can exec it without
# holding any elevated capabilities itself. The jailer performs all privileged
# operations (pivot_root, mknod, cgroup setup, chown), then drops to
# uid/gid 900 before exec-ing Firecracker.

[[ -f "$JAILER_BIN" ]] || {
    echo "error: $JAILER_BIN not found — install Firecracker and jailer first" >&2
    exit 1
}
chown root:root "$JAILER_BIN"
chmod u+s "$JAILER_BIN"

# ── KVM ────────────────────────────────────────────────────────────────────

if grep -q vmx /proc/cpuinfo 2>/dev/null; then
    KVM_MOD=kvm_intel
elif grep -q svm /proc/cpuinfo 2>/dev/null; then
    KVM_MOD=kvm_amd
else
    echo "warning: no hardware virtualization found in /proc/cpuinfo — KVM may be unavailable" >&2
    KVM_MOD=
fi

if [[ -n "$KVM_MOD" ]]; then
    modprobe "$KVM_MOD" 2>/dev/null || true
    echo "$KVM_MOD" > /etc/modules-load.d/firecracker.conf
fi

# ── Summary ────────────────────────────────────────────────────────────────

echo
echo "Host setup complete."
echo "  Jailed VM identity : firecracker (uid=$FIRECRACKER_UID gid=$FIRECRACKER_GID)"
echo "  Service group      : oblak"
echo "  Chroot base        : $JAILER_BASE  (root:oblak 775)"
echo "  Image dir          : $IMAGE_DIR  (oblak:oblak 755)"
echo "  Jailer             : $JAILER_BIN  (setuid root)"
[[ -n "$KVM_MOD" ]] && \
echo "  KVM module         : $KVM_MOD  (persisted via /etc/modules-load.d/firecracker.conf)"
echo
echo "Next steps:"
echo "  Install kernel and rootfs : sudo make install"
echo "  Network setup             : sudo ./deployment/network-setup.sh"
[[ -n "$DEV_USER" ]] && \
echo "  Re-login as $DEV_USER or run 'newgrp oblak' before starting the server"
