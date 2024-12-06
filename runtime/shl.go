package runtime

type Mode int

var (
	mode Mode
)

const (
	ModeUnknown Mode = iota
	ModeProgramToken
	ModeSlot
)

func parseMode(modeStr string) Mode {
	switch modeStr {
	case "slot":
		return ModeSlot
	case "token":
		return ModeProgramToken
	default:
		return ModeUnknown
	}
}

func InitSys(_mode string) {
	mode = parseMode(_mode)
}

func SysMode() Mode {
	return mode
}
