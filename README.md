# QEMU ARM + CAN automotive test PoC

This repository is an executable, local-first proof of concept for testing an
ARM virtual ECU over an emulated CAN bus:

```text
YAML test intent
       |
       v
ARMv7 testctl -- FlexCAN2 -- QEMU CAN bus -- FlexCAN1 -- ARMv7 body-ecu
       |
       v
JSON + JUnit XML + CAN trace + serial log
```

The complete portable path runs inside Docker, but QEMU still executes a real
32-bit ARM Linux guest using TCG. Inside the SabreLite guest, `body-ecu` owns
`can0`, `testctl` owns `can1`, and both modeled FlexCAN controllers share one
QEMU CAN bus. This topology also runs on Docker Desktop, whose LinuxKit kernel
does not provide the `vcan` module.

There is intentionally no API, queue, database, scheduler, Kubernetes, or cloud
provider in this PoC.

## Run the end-to-end PoC

Requirement: Docker with at least 8 GB available to its Linux VM.

```bash
make poc
```

The first build compiles a pinned Linux kernel and a pinned QEMU revision, so it
takes several minutes. Docker caches both layers; later Go/test changes rebuild
quickly. A successful run prints:

```text
PoC passed: QEMU ARM guest exchanged CAN frames across FlexCAN
Evidence: /work/runs/poc-<timestamp>
```

Evidence is persisted on the host under `runs/poc-<timestamp>/`:

```text
qemu-serial.log
guest-evidence/<run-id>/manifest.yaml
guest-evidence/<run-id>/can-trace.log
guest-evidence/<run-id>/result.json
guest-evidence/<run-id>/test-results.xml
```

The container exits `0` only after the ARM guest reports `POC_RESULT=passed` and
the evidence archive has been extracted successfully.

## What is exercised

- Linux 6.6.147 booting as ARMv7 on the QEMU SabreLite machine
- Both i.MX6 FlexCAN controllers enabled in the guest device tree
- CAN bit timing and link state configured through rtnetlink
- A Go body-control ECU receiving and responding on `can0`
- A YAML-driven Go test harness transmitting and validating on `can1`
- Positive response, state change, invalid payload, and expected-silence cases
- JSON, JUnit XML, CAN trace, manifest, and full serial boot log collection

QEMU's SabreLite FlexCAN model was merged after the QEMU 10.0 release. The
Docker build therefore pins QEMU commit
`fa19879df1658f96ac07365fca8835b7decd6995` instead of relying on the distro
binary. Kernel, QEMU, and Go binaries are all built in versioned, cached Docker
stages.

## Test manifest

The manifest expresses ECU behavior:

```yaml
name: headlight-regression
interface: vcan0
timeout: 20s
cases:
  - name: turn headlight on
    request: "100#01"
    expected: "101#01"
    timeout: 1s
```

Portable guest mode overrides only the runtime interface with
`testctl run --interface can1`; it does not modify the cases. See
`examples/headlight-test.yaml` for all four checks.

## Linux host + SocketCAN mode

The same Go processes can run without QEMU for quick development on Linux:

```bash
make setup-vcan
make run-ecu
```

Then, in another terminal:

```bash
make run-test
```

`qemu/run-sabrelite.sh` is the external-harness path for a Linux host with
SocketCAN. It bridges the guest FlexCAN controller to a host CAN interface via
QEMU's `can-host-socketcan` object. Docker Desktop cannot run this topology
because its LinuxKit kernel omits `vcan`; use portable dual-FlexCAN mode there.

## Development checks

```bash
make check
```

The CLI exit codes are `0` for pass, `1` for an ECU behavior failure, and `2`
for a manifest or harness error.

- [QEMU CAN bus emulation](https://www.qemu.org/docs/master/system/devices/can.html)
- [QEMU SabreLite board](https://www.qemu.org/docs/master/system/arm/sabrelite.html)
