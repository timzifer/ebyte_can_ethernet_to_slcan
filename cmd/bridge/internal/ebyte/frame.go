// Package ebyte contains helpers for working with the proprietary
// EByte CAN-to-Ethernet frame format.
package ebyte

import (
	"encoding/binary"
	"fmt"
)

// FrameSize defines the fixed size of the binary frames exchanged with the
// EByte adapter.
const FrameSize = 13

// Frame represents a CAN frame in the EByte binary wire format.
type Frame struct {
	ID       uint32
	Extended bool
	Remote   bool
	DLC      uint8
	Data     [8]byte
}

// ParseFrame converts the 13-byte binary frame emitted by the adapter into a
// structured Frame instance.
func ParseFrame(raw []byte) (Frame, error) {
	if len(raw) != FrameSize {
		return Frame{}, fmt.Errorf("invalid frame size %d", len(raw))
	}

	header := raw[0]

	frame := Frame{}
	frame.DLC = header & 0x0F
	if frame.DLC > 8 {
		return Frame{}, fmt.Errorf("invalid DLC %d", frame.DLC)
	}
	frame.Remote = header&0x40 != 0
	frame.Extended = header&0x80 != 0

	frame.ID = binary.BigEndian.Uint32(raw[1:5])
	copy(frame.Data[:], raw[5:])
	return frame, nil
}

// SerializeFrame converts a structured Frame into the 13-byte binary
// representation expected by the adapter.
func SerializeFrame(frame Frame) ([]byte, error) {
	if frame.DLC > 8 {
		return nil, fmt.Errorf("invalid DLC %d", frame.DLC)
	}

	buf := make([]byte, FrameSize)
	header := frame.DLC & 0x0F
	if frame.Remote {
		header |= 0x40
	}
	if frame.Extended {
		header |= 0x80
	}
	buf[0] = header
	binary.BigEndian.PutUint32(buf[1:5], frame.ID)
	copy(buf[5:], frame.Data[:])
	return buf, nil
}
