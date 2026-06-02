#!/usr/bin/env bash
# One-time host setup for Firecracker CNI networking.
# Run as root: sudo ./deployment/network-setup.sh
#
# Safe to re-run: iptables rules are checked before insertion.
# Does NOT touch INPUT/OUTPUT chains or your existing FORWARD rules.
# iptables rules are not persisted — they reset on reboot. Re-run this
# script after reboot, or use iptables-persistent to save them yourself.
set -euo pipefail

VM_SUBNET="192.168.127.0/24"

# ---------------------------------------------------------------------------
# CNI plugins
# ---------------------------------------------------------------------------

mkdir -p /opt/cni/bin

CNI_VERSION=v1.9.1
curl -fsSL "https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-amd64-${CNI_VERSION}.tgz" \
    | tar -xz -C /opt/cni/bin

if [[ -n "${SUDO_USER:-}" ]]; then
    USER_SHELL=$(getent passwd "$SUDO_USER" | cut -d: -f7)
    BUILD_OUT=$(sudo -Hu "$SUDO_USER" mktemp)
    sudo -Hu "$SUDO_USER" "$USER_SHELL" -lic "
        set -e
        src=\$(mktemp -d)
        trap 'rm -rf \"\$src\"' EXIT
        git clone https://github.com/awslabs/tc-redirect-tap \"\$src\"
        cd \"\$src\"
        go build -o '$BUILD_OUT' ./cmd/tc-redirect-tap
    "
    mv "$BUILD_OUT" /opt/cni/bin/tc-redirect-tap
    chmod 755 /opt/cni/bin/tc-redirect-tap
else
    src=$(mktemp -d)
    trap 'rm -rf "$src"' EXIT
    git clone https://github.com/awslabs/tc-redirect-tap "$src"
    (cd "$src" && go build -o /opt/cni/bin/tc-redirect-tap ./cmd/tc-redirect-tap)
fi

# tc-redirect-tap and ptp need CAP_NET_ADMIN + CAP_SYS_ADMIN to call setns()
# when entering a VM's network namespace. They are exec'd as child processes
# by the SDK and do not inherit file capabilities from the parent binary.
setcap cap_net_admin,cap_sys_admin+eip /opt/cni/bin/tc-redirect-tap
setcap cap_net_admin,cap_sys_admin+eip /opt/cni/bin/ptp

# ---------------------------------------------------------------------------
# CNI network config
#
# We omit the "firewall" plugin here and manage iptables rules ourselves
# below. The firewall plugin inserts per-VM ACCEPT rules at the top of the
# FORWARD chain at VM start time. This would push our RFC1918 DROP rules
# down and bypass them, because iptables evaluates first-match-wins.
#
# With static subnet-wide rules, we maintain control of rule order at all
# times — no race between our DROPs and dynamically-inserted ACCEPTs.
# ---------------------------------------------------------------------------

mkdir -p /etc/cni/conf.d

# Ubuntu uses 127.0.0.53 (systemd-resolved) as its nameserver, which is a
# loopback address unreachable from inside a VM. Provide a file with public
# DNS servers for host-local to pass through CNI into the vm's ip= kernel arg.
cat > /etc/cni/vm-resolv.conf <<'EOF'
nameserver 8.8.8.8
nameserver 1.1.1.1
EOF

cat > /etc/cni/conf.d/oblak.conflist <<EOF
{
  "name": "oblak",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "ptp",
      "ipMasq": false,
      "ipam": {
        "type": "host-local",
        "subnet": "${VM_SUBNET}",
        "resolvConf": "/etc/cni/vm-resolv.conf"
      }
    },
    {
      "type": "tc-redirect-tap"
    }
  ]
}
EOF

# ---------------------------------------------------------------------------
# Network namespace directory
#
# The SDK creates a netns for each VM under /var/run/netns. /var/run is a
# tmpfs owned by root (755), so the server process — even with CAP_NET_ADMIN
# and CAP_SYS_ADMIN — cannot create it without CAP_DAC_OVERRIDE.
# Pre-creating it here and giving ownership to the invoking user fixes this.
# Recreate after reboot (same as the iptables rules below).
# ---------------------------------------------------------------------------

SERVER_USER="${SUDO_USER:-$(whoami)}"
mkdir -p /var/run/netns
chown "$SERVER_USER" /var/run/netns
mkdir -p /var/lib/cni
chown -R "$SERVER_USER" /var/lib/cni

# ---------------------------------------------------------------------------
# IP forwarding
# ---------------------------------------------------------------------------

sysctl -w net.ipv4.ip_forward=1

# Persist across reboots. This only adds one key; it does not touch any
# other sysctl settings.
cat > /etc/sysctl.d/99-firecracker-net.conf <<'EOF'
net.ipv4.ip_forward = 1
EOF

# ---------------------------------------------------------------------------
# iptables FORWARD rules
#
# Your existing FORWARD policy is DROP, which is what we want. We only add
# rules specific to the VM subnet; nothing else in your chain is modified.
#
# Rule order (iptables is first-match-wins):
#   1. DROP VM → RFC1918 private ranges  (VMs cannot reach your LAN or
#      host-local services on internal networks, and cannot reach each other
#      via layer-3 routing through the host)
#   2. ACCEPT VM → internet              (pip install, etc.)
#   3. ACCEPT ESTABLISHED/RELATED        (return packets from the internet
#      back to the VM)
#
# We use -C to check before inserting so re-running is safe.
# ---------------------------------------------------------------------------

ipt_ensure() {
    # Usage: ipt_ensure -I|-A CHAIN [pos] rule...
    # Checks if the rule exists; skips insertion if it does.
    local op=$1; shift
    if ! iptables -C "$@" 2>/dev/null; then
        iptables "$op" "$@"
    fi
}

# NAT masquerade for VM internet access (replaces ptp's ipMasq which requires
# iptables to run with elevated caps as a child of the CNI plugin process).
ipt_ensure -A POSTROUTING -t nat -s "$VM_SUBNET" -j MASQUERADE

# Block VM traffic to all RFC 1918 private ranges.
# Inserted at position 1 so they precede any existing rules.
ipt_ensure -I FORWARD 1 -s "$VM_SUBNET" -d 10.0.0.0/8    -j DROP
ipt_ensure -I FORWARD 1 -s "$VM_SUBNET" -d 172.16.0.0/12  -j DROP
ipt_ensure -I FORWARD 1 -s "$VM_SUBNET" -d 192.168.0.0/16 -j DROP

# Allow VM subnet outbound to the internet (non-RFC1918 passes the DROPs above).
ipt_ensure -A FORWARD -s "$VM_SUBNET" -j ACCEPT

# Allow return traffic for established connections.
ipt_ensure -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT

# ---------------------------------------------------------------------------

echo ""
echo "Setup complete."
echo "  CNI plugins : /opt/cni/bin"
echo "  CNI config  : /etc/cni/conf.d/oblak.conflist (subnet: ${VM_SUBNET})"
echo "  IP forward  : enabled (persisted via /etc/sysctl.d/99-firecracker-net.conf)"
echo "  FORWARD rules added for VM subnet (not persisted — reset on reboot)"
