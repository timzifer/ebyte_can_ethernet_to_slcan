package app

import (
	"encoding/binary"
	"testing"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

func TestEncodeGVRETFrame(t *testing.T) {
	frame := ebyte.Frame{
		ID:   0x123,
		DLC:  2,
		Data: [8]byte{0x11, 0x22},
	}

	ts := uint32(0x11223344)
	data, err := encodeGVRETFrame(frame, ts, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := len(data), 2+4+4+1+int(frame.DLC)+1; got != want {
		t.Fatalf("unexpected payload length: got %d want %d", got, want)
	}

	if data[0] != 0xF1 || data[1] != 0x00 {
		t.Fatalf("unexpected header bytes: %x", data[:2])
	}

	if got := binary.LittleEndian.Uint32(data[2:6]); got != ts {
		t.Fatalf("unexpected timestamp: got 0x%08x want 0x%08x", got, ts)
	}

	if got := binary.LittleEndian.Uint32(data[6:10]); got != frame.ID {
		t.Fatalf("unexpected identifier: got 0x%08x want 0x%08x", got, frame.ID)
	}

	lengthBus := data[10]
	if lengthBus&0x0F != frame.DLC {
		t.Fatalf("unexpected DLC encoding: got 0x%02x want 0x%02x", lengthBus, frame.DLC)
	}
	if lengthBus>>4 != 1 {
		t.Fatalf("unexpected bus encoding: got %d want %d", lengthBus>>4, 1)
	}

	if got, want := data[11:13], []byte{0x11, 0x22}; !equalSlices(got, want) {
		t.Fatalf("data mismatch: got %x want %x", got, want)
	}

	if data[len(data)-1] != 0x00 {
		t.Fatalf("expected terminating zero byte")
	}
}

func TestEncodeGVRETFrameExtended(t *testing.T) {
	frame := ebyte.Frame{
		ID:       0x1ABCDE,
		Extended: true,
		DLC:      4,
		Data:     [8]byte{0xDE, 0xAD, 0xBE, 0xEF},
	}

	data, err := encodeGVRETFrame(frame, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := binary.LittleEndian.Uint32(data[6:10])
	const extendedMask = 1 << 31
	if id&extendedMask == 0 {
		t.Fatalf("expected extended flag to be set, got 0x%08x", id)
	}

	if id&^uint32(extendedMask) != frame.ID {
		t.Fatalf("identifier mismatch: got 0x%08x want 0x%08x", id&^uint32(extendedMask), frame.ID)
	}
}

func TestEncodeGVRETFrameRemote(t *testing.T) {
	frame := ebyte.Frame{
		ID:     0x321,
		DLC:    3,
		Remote: true,
	}

	data, err := encodeGVRETFrame(frame, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := binary.LittleEndian.Uint32(data[6:10])
	const remoteMask = 1 << 30
	if id&remoteMask == 0 {
		t.Fatalf("expected remote flag to be set, got 0x%08x", id)
	}

	if len(data) != 2+4+4+1+int(frame.DLC)+1 {
		t.Fatalf("unexpected length for remote frame: got %d", len(data))
	}

	payload := data[11 : 11+frame.DLC]
	for i, b := range payload {
		if b != 0x00 {
			t.Fatalf("expected zero padding for remote payload at %d, got 0x%02x", i, b)
		}
	}
}

func TestEncodeGVRETFrameAutoExtended(t *testing.T) {
	frame := ebyte.Frame{ID: 0x1ABCDE, DLC: 1}

	data, err := encodeGVRETFrame(frame, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := binary.LittleEndian.Uint32(data[6:10])
	const extendedMask = 1 << 31
	if id&extendedMask == 0 {
		t.Fatalf("expected extended flag to be set for identifier 0x%x", frame.ID)
	}
}

func TestEncodeGVRETFrameErrors(t *testing.T) {
	if _, err := encodeGVRETFrame(ebyte.Frame{DLC: 9}, 0, 0); err == nil {
		t.Fatalf("expected error for DLC > 8")
	}
}

func equalSlices(a, b []byte) bool {
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
