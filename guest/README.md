# ARM guest image

`Dockerfile.poc` builds and packages the guest as three pinned artifacts:

```text
zImage                    Linux 6.6.147
imx6q-sabrelite.dtb       FlexCAN1 and FlexCAN2 enabled
rootfs.cpio.gz             static BusyBox and ARMv7 test binaries
```

The initramfs contains:

- `/init`: mounts pseudo filesystems, starts the ECU, runs tests, exports evidence
- `/usr/bin/body-ecu`: static ARMv7 ECU process
- `/usr/bin/testctl`: static ARMv7 test harness
- `/usr/bin/can-up`: static rtnetlink helper for CAN bitrate/link setup
- `/etc/headlight-test.yaml`: behavior manifest

The guest sends a compressed evidence archive through its serial console between
`POC_EVIDENCE_BEGIN` and `POC_EVIDENCE_END`. The container launcher extracts it
and requires a final `POC_RESULT=passed` marker.

For external SocketCAN mode, the artifact contract remains `zImage`,
`imx6q-sabrelite.dtb`, and `rootfs.cpio.gz`; use an ECU-only initramfs and
`qemu/run-sabrelite.sh` on a Linux host with a CAN or `vcan` interface.
