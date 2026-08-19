package harness

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/qemu-arm-can-test/internal/manifest"
	"example.com/qemu-arm-can-test/internal/protocol"
	"example.com/qemu-arm-can-test/internal/socketcan"
)

type loopbackECU struct{ pending *socketcan.Frame }

func (bus *loopbackECU) Send(frame socketcan.Frame) error {
	response, handled := protocol.HandleHeadlightRequest(frame)
	if handled {
		bus.pending = &response
	} else {
		bus.pending = nil
	}
	return nil
}

func (bus *loopbackECU) Receive(context.Context) (socketcan.Frame, error) {
	if bus.pending == nil {
		return socketcan.Frame{}, socketcan.ErrTimeout
	}
	frame := *bus.pending
	bus.pending = nil
	return frame, nil
}

func (bus *loopbackECU) Close() error { return nil }

func TestRunnerExecutesProtocolAndWritesTrace(t *testing.T) {
	evidenceDir := t.TempDir()
	run := manifest.TestRun{Name: "headlight", Interface: "vcan0"}
	cases := []manifest.PreparedCase{
		{Name: "on", Request: socketcan.Frame{ID: 0x100, Data: []byte{1}}, Expected: socketcan.Frame{ID: 0x101, Data: []byte{1}}, Timeout: time.Second},
		{Name: "ignore", Request: socketcan.Frame{ID: 0x200}, ExpectSilence: true, Timeout: time.Millisecond},
	}

	result, err := (Runner{Output: io.Discard}).Run(context.Background(), run, cases, &loopbackECU{}, "run-1", evidenceDir)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q, want passed: %+v", result.Status, result.Cases)
	}
	trace, err := os.ReadFile(filepath.Join(evidenceDir, "can-trace.log"))
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	if len(trace) == 0 {
		t.Fatal("CAN trace is empty")
	}
}
