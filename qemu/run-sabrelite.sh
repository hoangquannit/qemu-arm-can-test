#!/usr/bin/env sh
set -eu

artifact_dir="${1:-build/guest-image}"
host_can="${HOST_CAN_INTERFACE:-vcan0}"
kernel="$artifact_dir/zImage"
device_tree="$artifact_dir/imx6q-sabrelite.dtb"
initramfs="$artifact_dir/rootfs.cpio.gz"

for required in "$kernel" "$device_tree" "$initramfs"; do
  if [ ! -f "$required" ]; then
    echo "run-sabrelite: missing guest artifact: $required" >&2
    exit 2
  fi
done

if ! ip link show "$host_can" >/dev/null 2>&1; then
  echo "run-sabrelite: host SocketCAN interface $host_can does not exist" >&2
  exit 2
fi

exec qemu-system-arm \
  -M sabrelite \
  -smp 4 \
  -m 1G \
  -object can-bus,id=qcan0 \
  -machine canbus0=qcan0 \
  -object "can-host-socketcan,if=$host_can,canbus=qcan0,id=hostcan0" \
  -kernel "$kernel" \
  -dtb "$device_tree" \
  -initrd "$initramfs" \
  -append "console=ttymxc0,115200 rdinit=/sbin/init" \
  -nographic \
  -no-reboot
