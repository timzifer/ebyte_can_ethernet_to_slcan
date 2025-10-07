package slcan

import "testing"

func TestParseCommand(t *testing.T) {
	cases := []struct {
		input string
		want  CommandType
	}{
		{"O", CommandOpen},
		{"C", CommandClose},
		{"", CommandUnknown},
		{"T123", CommandUnknown},
	}

	for _, tc := range cases {
		cmd := ParseCommand(tc.input)
		if cmd.Type != tc.want {
			t.Fatalf("for %q expected %v got %v", tc.input, tc.want, cmd.Type)
		}
	}
}

func TestParseCommandKeepsRaw(t *testing.T) {
	input := "O123"
	cmd := ParseCommand(input)
	if cmd.Raw != input {
		t.Fatalf("expected raw command to be preserved, got %q", cmd.Raw)
	}
}
