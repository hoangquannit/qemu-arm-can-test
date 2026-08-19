package socketcan

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	MaxStandardID = 0x7FF
	MaxDataLength = 8
)

var (
	ErrUnsupported = errors.New("socketcan is supported only on Linux")
	ErrTimeout     = errors.New("waiting for CAN frame timed out")
)

// Frame is a Classical CAN 2.0 frame. The initial PoC intentionally supports
// standard 11-bit identifiers and payloads of at most eight bytes.
type Frame struct {
	ID   uint32 `json:"id"`
	Data []byte `json:"data"`
}

func ParseFrame(value string) (Frame, error) {
	idText, dataText, found := strings.Cut(strings.TrimSpace(value), "#")
	if !found {
		return Frame{}, fmt.Errorf("CAN frame %q must use ID#DATA format", value)
	}

	idValue, err := strconv.ParseUint(idText, 16, 32)
	if err != nil {
		return Frame{}, fmt.Errorf("parse CAN ID %q: %w", idText, err)
	}
	if idValue > MaxStandardID {
		return Frame{}, fmt.Errorf("CAN ID 0x%X exceeds standard 11-bit range", idValue)
	}
	if len(dataText)%2 != 0 {
		return Frame{}, fmt.Errorf("CAN data %q must contain complete hexadecimal bytes", dataText)
	}
	if len(dataText) > MaxDataLength*2 {
		return Frame{}, fmt.Errorf("CAN data contains %d bytes; maximum is %d", len(dataText)/2, MaxDataLength)
	}

	data := make([]byte, len(dataText)/2)
	if _, err := hex.Decode(data, []byte(dataText)); err != nil {
		return Frame{}, fmt.Errorf("parse CAN data %q: %w", dataText, err)
	}

	return Frame{ID: uint32(idValue), Data: data}, nil
}

func (f Frame) Validate() error {
	if f.ID > MaxStandardID {
		return fmt.Errorf("CAN ID 0x%X exceeds standard 11-bit range", f.ID)
	}
	if len(f.Data) > MaxDataLength {
		return fmt.Errorf("CAN payload contains %d bytes; maximum is %d", len(f.Data), MaxDataLength)
	}
	return nil
}

func (f Frame) String() string {
	return fmt.Sprintf("%03X#%s", f.ID, strings.ToUpper(hex.EncodeToString(f.Data)))
}

func Equal(left, right Frame) bool {
	if left.ID != right.ID || len(left.Data) != len(right.Data) {
		return false
	}
	for index := range left.Data {
		if left.Data[index] != right.Data[index] {
			return false
		}
	}
	return true
}
