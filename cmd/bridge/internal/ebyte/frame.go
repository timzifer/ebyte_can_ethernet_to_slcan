package ebyte

import (
	"encoding/binary"
	"fmt"
)

const FrameSize = 13

type Frame struct {
	ID       uint32
	Extended bool
	Remote   bool
	DLC      uint8
	Data     [8]byte
}

func ParseFrame(raw []byte) (Frame, error) {
	if len(raw) != FrameSize {
		return Frame{}, fmt.Errorf("invalid frame size %d", len(raw))
	}

	header := raw[0]
	if header&0x80 == 0 {
		return Frame{}, fmt.Errorf("invalid frame header 0x%02x", header)
	}

	frame := Frame{}
	frame.DLC = header & 0x0F
	frame.Remote = header&0x10 != 0
	frame.Extended = header&0x20 != 0

	frame.ID = binary.BigEndian.Uint32(raw[1:5])
	copy(frame.Data[:], raw[5:])
	return frame, nil
}

func SerializeFrame(frame Frame) ([]byte, error) {
	if frame.DLC > 8 {
		return nil, fmt.Errorf("invalid DLC %d", frame.DLC)
	}

	buf := make([]byte, FrameSize)
	header := uint8(0x80 | (frame.DLC & 0x0F))
	if frame.Remote {
		header |= 0x10
	}
	if frame.Extended {
		header |= 0x20
	}
	buf[0] = header
	binary.BigEndian.PutUint32(buf[1:5], frame.ID)
	copy(buf[5:], frame.Data[:])
	return buf, nil
}
