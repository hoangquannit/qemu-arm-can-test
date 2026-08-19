#!/bin/sh
set -eu

run_root="${RUN_ROOT:-/work/runs}"
run_stamp="$(date -u +%Y%m%d-%H%M%S)"
host_epoch="$(date -u +%s)"
poc_root="$run_root/poc-$run_stamp"
serial_log="$poc_root/qemu-serial.log"
guest_evidence="$poc_root/guest-evidence"
mkdir -p "$poc_root" "$guest_evidence"

cleanup() {
  if [ -n "${qemu_pid:-}" ]; then
    kill "$qemu_pid" 2>/dev/null || true
    wait "$qemu_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

qemu-system-arm \
  -M sabrelite \
  -smp 4 \
  -m 512M \
  -object can-bus,id=qcan0 \
  -machine canbus0=qcan0 \
  -machine canbus1=qcan0 \
  -kernel /opt/poc/zImage \
  -dtb /opt/poc/imx6q-sabrelite.dtb \
  -initrd /opt/poc/rootfs.cpio.gz \
  -append "console=ttymxc0,115200 rdinit=/init panic=-1 poc_epoch=$host_epoch" \
  -display none \
  -monitor none \
  -serial stdio \
  -no-reboot >"$serial_log" 2>&1 &
qemu_pid=$!

finished=0
attempt=0
while [ "$attempt" -lt 90 ]; do
  if grep -q "POC_RESULT=" "$serial_log"; then
    finished=1
    break
  fi
  if ! kill -0 "$qemu_pid" 2>/dev/null; then
    echo "PoC failed: QEMU exited without a result marker" >&2
    tail -n 80 "$serial_log" >&2
    exit 2
  fi
  attempt=$((attempt + 1))
  sleep 1
done

if [ "$finished" -ne 1 ]; then
  echo "PoC failed: timed out waiting for the guest test result" >&2
  tail -n 80 "$serial_log" >&2
  exit 2
fi

awk '/POC_EVIDENCE_BEGIN/{capture=1;next}/POC_EVIDENCE_END/{capture=0}capture' "$serial_log" \
  | tr -d '\r' \
  | base64 -d \
  | tar -xzf - -C "$guest_evidence"

if ! grep -q "POC_RESULT=passed" "$serial_log"; then
  echo "PoC behavior tests failed" >&2
  tail -n 100 "$serial_log" >&2
  exit 1
fi

echo "PoC passed: QEMU ARM guest exchanged CAN frames across FlexCAN"
echo "Evidence: $poc_root"
