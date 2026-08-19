package socketcan

import "testing"

func TestParseFrameRoundTrip(t *testing.T) {
	frame, err := ParseFrame("100#01aB")
	if err != nil {
		t.Fatalf("ParseFrame returned an error: %v", err)
	}
	if got, want := frame.String(), "100#01AB"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestParseFrameRejectsInvalidFrames(t *testing.T) {
	for _, value := range []string{"100", "800#01", "100#1", "100#000102030405060708"} {
		if _, err := ParseFrame(value); err == nil {
			t.Errorf("ParseFrame(%q) unexpectedly succeeded", value)
		}
	}
}
