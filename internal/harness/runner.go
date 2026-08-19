package harness

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"example.com/qemu-arm-can-test/internal/evidence"
	"example.com/qemu-arm-can-test/internal/manifest"
	"example.com/qemu-arm-can-test/internal/socketcan"
)

type Runner struct {
	Output io.Writer
	Now    func() time.Time
}

func (r Runner) Run(
	ctx context.Context,
	run manifest.TestRun,
	cases []manifest.PreparedCase,
	bus socketcan.Bus,
	runID string,
	evidenceDir string,
) (evidence.RunResult, error) {
	if r.Output == nil {
		r.Output = io.Discard
	}
	if r.Now == nil {
		r.Now = time.Now
	}

	startedAt := r.Now()
	result := evidence.RunResult{
		RunID:       runID,
		Name:        run.Name,
		Status:      "passed",
		StartedAt:   startedAt,
		EvidenceDir: evidenceDir,
	}

	traceFile, err := os.Create(filepath.Join(evidenceDir, "can-trace.log"))
	if err != nil {
		return result, fmt.Errorf("create CAN trace: %w", err)
	}
	defer traceFile.Close()
	trace := bufio.NewWriter(traceFile)
	defer trace.Flush()

	for _, testCase := range cases {
		caseStarted := r.Now()
		fmt.Fprintf(r.Output, "  RUN  %-28s %s\n", testCase.Name, testCase.Request)
		fmt.Fprintf(trace, "%s TX %s\n", caseStarted.UTC().Format(time.RFC3339Nano), testCase.Request)

		caseResult := evidence.CaseResult{Name: testCase.Name, Status: "passed"}
		if err := bus.Send(testCase.Request); err != nil {
			caseResult.Status = "failed"
			caseResult.Message = fmt.Sprintf("send request: %v", err)
		} else {
			caseResult.Message = r.waitForExpected(ctx, bus, trace, testCase)
			if caseResult.Message != "" {
				caseResult.Status = "failed"
			}
		}

		caseResult.Duration = r.Now().Sub(caseStarted)
		result.Cases = append(result.Cases, caseResult)
		if caseResult.Status == "passed" {
			fmt.Fprintf(r.Output, "  PASS %-28s %s\n", testCase.Name, caseResult.Duration.Round(time.Millisecond))
		} else {
			result.Status = "failed"
			fmt.Fprintf(r.Output, "  FAIL %-28s %s\n       %s\n", testCase.Name, caseResult.Duration.Round(time.Millisecond), caseResult.Message)
		}
	}

	result.Duration = r.Now().Sub(startedAt)
	return result, nil
}

func (r Runner) waitForExpected(
	parent context.Context,
	bus socketcan.Bus,
	trace *bufio.Writer,
	testCase manifest.PreparedCase,
) string {
	ctx, cancel := context.WithTimeout(parent, testCase.Timeout)
	defer cancel()

	for {
		frame, err := bus.Receive(ctx)
		if err != nil {
			if testCase.ExpectSilence && (errors.Is(err, socketcan.ErrTimeout) || errors.Is(err, context.DeadlineExceeded)) {
				return ""
			}
			if errors.Is(err, socketcan.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
				return fmt.Sprintf("expected %s within %s, received no matching response", testCase.Expected, testCase.Timeout)
			}
			return fmt.Sprintf("receive response: %v", err)
		}

		fmt.Fprintf(trace, "%s RX %s\n", r.Now().UTC().Format(time.RFC3339Nano), frame)
		if testCase.ExpectSilence {
			return fmt.Sprintf("expected no response for %s, received %s", testCase.Timeout, frame)
		}
		if socketcan.Equal(frame, testCase.Expected) {
			return ""
		}
	}
}
