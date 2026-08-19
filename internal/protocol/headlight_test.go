package protocol

import (
	"testing"

	"example.com/qemu-arm-can-test/internal/socketcan"
)

func TestHandleHeadlightRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  socketcan.Frame
		expected socketcan.Frame
		handled  bool
	}{
		{"on", socketcan.Frame{ID: 0x100, Data: []byte{0x01}}, socketcan.Frame{ID: 0x101, Data: []byte{0x01}}, true},
		{"off", socketcan.Frame{ID: 0x100, Data: []byte{0x00}}, socketcan.Frame{ID: 0x101, Data: []byte{0x00}}, true},
		{"invalid", socketcan.Frame{ID: 0x100, Data: []byte{0xFF}}, socketcan.Frame{ID: 0x1FF, Data: []byte{0x01}}, true},
		{"unrelated", socketcan.Frame{ID: 0x200, Data: []byte{0x01}}, socketcan.Frame{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, handled := HandleHeadlightRequest(test.request)
			if handled != test.handled {
				t.Fatalf("handled = %v, want %v", handled, test.handled)
			}
			if handled && !socketcan.Equal(response, test.expected) {
				t.Fatalf("response = %s, want %s", response, test.expected)
			}
		})
	}
}
