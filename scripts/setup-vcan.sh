#!/usr/bin/env sh
set -eu

interface_name="${1:-vcan0}"

if [ "$(uname -s)" != "Linux" ]; then
  echo "setup-vcan: SocketCAN requires Linux" >&2
  exit 2
fi

if ! ip link show "$interface_name" >/dev/null 2>&1; then
  sudo modprobe vcan
  sudo ip link add dev "$interface_name" type vcan
fi

sudo ip link set up "$interface_name"
ip -details link show "$interface_name"
