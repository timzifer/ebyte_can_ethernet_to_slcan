package app

import (
	"encoding/binary"
	"testing"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

func TestEncodeCANserverFrame(t *testing.T) {
	frame := ebyte.Frame{
		ID:  0x123,
		DLC: 8,
		Data: [8]byte{
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		},
	}

	data, err := encodeCANserverFrame(frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 16 {
		t.Fatalf("expected 16-byte payload, got %d", len(data))
	}

	header1 := binary.LittleEndian.Uint32(data[0:4])
	if header1 != frame.ID<<21 {
		t.Fatalf("unexpected header1: got 0x%08x want 0x%08x", header1, frame.ID<<21)
	}

	header2 := binary.LittleEndian.Uint32(data[4:8])
	if header2 != uint32(frame.DLC) {
		t.Fatalf("unexpected header2: got 0x%08x want 0x%08x", header2, frame.DLC)
	}

	if got, want := data[8:], frame.Data[:]; !equalBytes(got, want) {
		t.Fatalf("data mismatch: got %x want %x", got, want)
	}
}

func TestEncodeCANserverFrameErrors(t *testing.T) {
	_, err := encodeCANserverFrame(ebyte.Frame{DLC: 9})
	if err == nil {
		t.Fatalf("expected error for DLC > 8")
	}

	_, err = encodeCANserverFrame(ebyte.Frame{Extended: true})
	if err == nil {
		t.Fatalf("expected error for extended frame")
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
