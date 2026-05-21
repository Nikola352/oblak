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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
git clone https://github.com/awslabs/tc-redirect-tap "$TMPDIR/tc-redirect-tap"
if [[ -n "${SUDO_USER:-}" ]]; then
    sudo -u "$SUDO_USER" bash -lc "go build -o '/opt/cni/bin/tc-redirect-tap' '$TMPDIR/tc-redirect-tap/cmd/tc-redirect-tap'"
else
    go build -o /opt/cni/bin/tc-redirect-tap "$TMPDIR/tc-redirect-tap/cmd/tc-redirect-tap"
fi

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
cat > /etc/cni/conf.d/oblak.conflist <<EOF
{
  "name": "oblak",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "ptp",
      "ipMasq": true,
      "ipam": {
        "type": "host-local",
        "subnet": "${VM_SUBNET}",
        "resolvConf": "/etc/resolv.conf"
      }
    },
    {
      "type": "tc-redirect-tap"
    }
  ]
}
EOF

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
