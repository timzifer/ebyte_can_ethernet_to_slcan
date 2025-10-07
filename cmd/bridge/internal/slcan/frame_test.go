package slcan

import (
	"strings"
	"testing"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

func TestEncodeFrame(t *testing.T) {
	frame := ebyte.Frame{
		ID:   0x123,
		DLC:  2,
		Data: [8]byte{0xAB, 0xCD},
	}

	encoded := EncodeFrame(frame)
	expected := "t1232ABCD\r"
	if encoded != expected {
		t.Fatalf("expected %q got %q", expected, encoded)
	}
}

func TestEncodeExtendedRemote(t *testing.T) {
	frame := ebyte.Frame{
		ID:       0x1ABCDEF0,
		DLC:      0,
		Extended: true,
		Remote:   true,
	}

	encoded := EncodeFrame(frame)
	if !strings.HasPrefix(encoded, "R1ABCDEF0") {
		t.Fatalf("unexpected encoding %q", encoded)
	}
	if encoded[len(encoded)-1] != '\r' {
		t.Fatalf("missing terminator in %q", encoded)
	}
}
