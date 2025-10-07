package ebyte

import "testing"

func TestParseFrame(t *testing.T) {
	raw := []byte{0x88, 0x12, 0x34, 0x56, 0x78, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	frame, err := ParseFrame(raw)
	if err != nil {
		t.Fatalf("ParseFrame returned error: %v", err)
	}

	if frame.DLC != 8 {
		t.Fatalf("expected DLC 8, got %d", frame.DLC)
	}
	if frame.Remote {
		t.Fatalf("expected data frame")
	}
	if !frame.Extended {
		t.Fatalf("expected extended frame")
	}
	if frame.ID != 0x12345678 {
		t.Fatalf("unexpected ID %08x", frame.ID)
	}
	if frame.Data[0] != 0x11 || frame.Data[7] != 0x88 {
		t.Fatalf("unexpected data payload %x", frame.Data)
	}
}

func TestParseFrameStandard(t *testing.T) {
	raw := []byte{0x06, 0x00, 0x00, 0x03, 0xF0, 0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x00, 0x00}
	frame, err := ParseFrame(raw)
	if err != nil {
		t.Fatalf("ParseFrame returned error: %v", err)
	}

	if frame.Extended {
		t.Fatalf("expected standard frame")
	}
	if frame.Remote {
		t.Fatalf("expected data frame")
	}
	if frame.DLC != 6 {
		t.Fatalf("expected DLC 6, got %d", frame.DLC)
	}
	if frame.ID != 0x3F0 {
		t.Fatalf("unexpected ID %08x", frame.ID)
	}
	if frame.Data[0] != 0xAA || frame.Data[3] != 0xDD {
		t.Fatalf("unexpected data payload %x", frame.Data)
	}
}

func TestSerializeFrame(t *testing.T) {
	frame := Frame{
		ID:       0x1abcdef0,
		DLC:      8,
		Extended: true,
		Remote:   false,
		Data:     [8]byte{0, 1, 2, 3, 4, 5, 6, 7},
	}

	raw, err := SerializeFrame(frame)
	if err != nil {
		t.Fatalf("SerializeFrame returned error: %v", err)
	}

	if len(raw) != FrameSize {
		t.Fatalf("expected length %d got %d", FrameSize, len(raw))
	}

	parsed, err := ParseFrame(raw)
	if err != nil {
		t.Fatalf("ParseFrame returned error: %v", err)
	}

	if parsed.ID != frame.ID || parsed.DLC != frame.DLC || !parsed.Extended {
		t.Fatalf("parsed frame mismatch: %+v", parsed)
	}
}

func TestSerializeFrameRemote(t *testing.T) {
	frame := Frame{
		ID:     0x123,
		DLC:    2,
		Remote: true,
	}

	raw, err := SerializeFrame(frame)
	if err != nil {
		t.Fatalf("SerializeFrame returned error: %v", err)
	}

	if raw[0]&0x40 == 0 {
		t.Fatalf("expected remote flag to be set in header byte")
	}

	parsed, err := ParseFrame(raw)
	if err != nil {
		t.Fatalf("ParseFrame returned error: %v", err)
	}

	if !parsed.Remote {
		t.Fatalf("expected remote frame, got %+v", parsed)
	}
	if parsed.ID != frame.ID {
		t.Fatalf("unexpected identifier: got 0x%x want 0x%x", parsed.ID, frame.ID)
	}
}
