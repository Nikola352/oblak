#!/bin/bash
set -e

if [ "$(id -u)" -ne 0 ]; then
    exec sudo "$0" "$@"
fi

ARCH="$(uname -m)"
UBUNTU_VERSION="24.04"
KERNEL_VERSION="6.1.155"

# Determine CI version from Firecracker releases
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest_version=$(basename $(curl -fsSLI -o /dev/null -w %{url_effective} ${release_url}/latest))
CI_VERSION=${latest_version%.*}

echo "Using Firecracker CI version: $CI_VERSION"

# Download the specific kernel
KERNEL_KEY="firecracker-ci/${CI_VERSION}/${ARCH}/vmlinux-${KERNEL_VERSION}"
KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/${KERNEL_KEY}"

mkdir -p ./deployment/firecracker

echo "Downloading kernel vmlinux-${KERNEL_VERSION}..."
if wget -O "./deployment/firecracker/vmlinux-${KERNEL_VERSION}" "$KERNEL_URL"; then
    echo "✓ Kernel downloaded: vmlinux-${KERNEL_VERSION}"
else
    echo "✗ Failed to download kernel. The version ${KERNEL_VERSION} may not be available for ${ARCH} in CI version ${CI_VERSION}"
    exit 1
fi

# Download Ubuntu 24.04 rootfs (squashfs)
ROOTFS_KEY="firecracker-ci/${CI_VERSION}/${ARCH}/ubuntu-${UBUNTU_VERSION}.squashfs"
ROOTFS_URL="https://s3.amazonaws.com/spec.ccfc.min/${ROOTFS_KEY}"

echo "Downloading Ubuntu ${UBUNTU_VERSION} squashfs rootfs..."
if wget -O "./deployment/firecracker/ubuntu-${UBUNTU_VERSION}.squashfs.upstream" "$ROOTFS_URL"; then
    echo "✓ Rootfs downloaded: ubuntu-${UBUNTU_VERSION}.squashfs.upstream"
else
    echo "✗ Failed to download Ubuntu ${UBUNTU_VERSION} rootfs for ${ARCH}"
    exit 1
fi

cd ./deployment/firecracker

# Repack squashfs
echo "Repacking squashfs..."
unsquashfs ubuntu-${UBUNTU_VERSION}.squashfs.upstream
chown -R root:root squashfs-root
mksquashfs squashfs-root "ubuntu-${UBUNTU_VERSION}.squashfs" -comp zstd -noappend

# Clean up temporary files
rm -rf squashfs-root
rm "ubuntu-${UBUNTU_VERSION}.squashfs.upstream"

# Verification
echo
echo "========================================="
echo "Setup complete! The following files are ready:"
echo "========================================="
[ -f "vmlinux-${KERNEL_VERSION}" ] && echo "✓ Kernel: vmlinux-${KERNEL_VERSION}" || echo "✗ Kernel missing"
[ -f "ubuntu-${UBUNTU_VERSION}.squashfs" ] && echo "✓ Rootfs: ubuntu-${UBUNTU_VERSION}.squashfs (read-only)" || echo "✗ Rootfs missing"

# Validate squashfs
if [ -f "ubuntu-${UBUNTU_VERSION}.squashfs" ]; then
    UNSQUASHFS_CHECK=$(unsquashfs -s "ubuntu-${UBUNTU_VERSION}.squashfs" 2>&1 | grep -q "Found a valid" && echo "valid")
    [ -n "$UNSQUASHFS_CHECK" ] && echo "✓ Rootfs is valid squashfs filesystem" || echo "⚠ Rootfs may be corrupted"
fi

if [ -n "${SUDO_USER:-}" ]; then
    chown -R "$SUDO_USER:$SUDO_USER" .
fi

echo
echo "To use with Firecracker (squashfs as read-only root):"
echo "  --kernel vmlinux-${KERNEL_VERSION}"
echo "  --rootfs ubuntu-${UBUNTU_VERSION}.squashfs"
echo
echo "Note: When using squashfs, add 'ro' to your kernel command line:"
echo "  --boot-args 'console=ttyS0 reboot=k panic=1 pci=off ro'"
