package manifest

import (
	"testing"
	"time"
)

func TestValidateExampleShape(t *testing.T) {
	run := TestRun{
		Name: "headlight-regression", Interface: "vcan0", Timeout: Duration{Duration: 10 * time.Second},
		Cases: []TestCase{
			{Name: "on", Request: "100#01", Expected: "101#01", Timeout: Duration{Duration: time.Second}},
			{Name: "ignore", Request: "200#01", ExpectSilence: true},
		},
	}
	cases, err := run.Validate()
	if err != nil {
		t.Fatalf("Validate returned an error: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("len(cases) = %d, want 2", len(cases))
	}
	if cases[1].Timeout != time.Second {
		t.Fatalf("default timeout = %s, want 1s", cases[1].Timeout)
	}
}

func TestValidateRequiresOneExpectation(t *testing.T) {
	run := TestRun{
		Name: "bad", Interface: "vcan0", Timeout: Duration{Duration: time.Second},
		Cases: []TestCase{{Name: "ambiguous", Request: "100#01"}},
	}
	if _, err := run.Validate(); err == nil {
		t.Fatal("Validate unexpectedly accepted a case without an expectation")
	}
}
