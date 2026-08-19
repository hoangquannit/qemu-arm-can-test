package protocol

import (
	"fmt"

	"example.com/qemu-arm-can-test/internal/socketcan"
)

const (
	HeadlightRequestID  uint32 = 0x100
	HeadlightResponseID uint32 = 0x101
	ProtocolErrorID     uint32 = 0x1FF
)

func HandleHeadlightRequest(request socketcan.Frame) (socketcan.Frame, bool) {
	if request.ID != HeadlightRequestID {
		return socketcan.Frame{}, false
	}
	if len(request.Data) != 1 || (request.Data[0] != 0x00 && request.Data[0] != 0x01) {
		return socketcan.Frame{ID: ProtocolErrorID, Data: []byte{0x01}}, true
	}
	return socketcan.Frame{ID: HeadlightResponseID, Data: []byte{request.Data[0]}}, true
}

func Describe(frame socketcan.Frame) string {
	if frame.ID == HeadlightRequestID && len(frame.Data) == 1 {
		switch frame.Data[0] {
		case 0x00:
			return "turn headlight off"
		case 0x01:
			return "turn headlight on"
		}
	}
	return fmt.Sprintf("frame %s", frame)
}
