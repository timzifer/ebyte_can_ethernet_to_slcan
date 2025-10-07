// Package slcan implements helpers for the serial CAN (SLCAN) textual
// protocol that GVRET clients understand.
package slcan

import (
	"fmt"
	"strings"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

// EncodeFrame converts an internal CAN frame into the ASCII SLCAN string that
// GVRET-compatible clients expect.
func EncodeFrame(frame ebyte.Frame) string {
	var builder strings.Builder
	switch {
	case frame.Remote && frame.Extended:
		builder.WriteByte('R')
	case frame.Remote && !frame.Extended:
		builder.WriteByte('r')
	case !frame.Remote && frame.Extended:
		builder.WriteByte('T')
	default:
		builder.WriteByte('t')
	}

	if frame.Extended {
		builder.WriteString(fmt.Sprintf("%08X", frame.ID&0x1FFFFFFF))
	} else {
		builder.WriteString(fmt.Sprintf("%03X", frame.ID&0x7FF))
	}

	builder.WriteByte('0' + byte(frame.DLC&0x0F))

	if !frame.Remote {
		for i := uint8(0); i < frame.DLC && i < 8; i++ {
			builder.WriteString(fmt.Sprintf("%02X", frame.Data[i]))
		}
	}

	builder.WriteByte('\r')
	return builder.String()
}
