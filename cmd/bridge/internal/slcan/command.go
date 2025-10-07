package slcan

type CommandType int

const (
	CommandUnknown CommandType = iota
	CommandOpen
	CommandClose
)

type Command struct {
	Type CommandType
	Raw  string
}

func ParseCommand(raw string) Command {
	if raw == "" {
		return Command{Type: CommandUnknown, Raw: raw}
	}

	switch raw[0] {
	case 'O':
		return Command{Type: CommandOpen, Raw: raw}
	case 'C':
		return Command{Type: CommandClose, Raw: raw}
	default:
		return Command{Type: CommandUnknown, Raw: raw}
	}
}
