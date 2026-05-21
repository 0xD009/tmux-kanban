package ui

const (
	KeyQuit        = "q"
	KeyCommand     = ":"
	KeyRefresh     = "r"
	KeyToggleView  = "tab"
	KeyToggleView2 = "v"
	KeyAttach      = "a"
	KeyStatus      = "s"
	KeyUnskip      = "u"
	KeyHermes      = "h"
	KeyRelay       = "x"
	KeyMessage     = "m"
	KeyMain        = "g"
	KeySnapshot    = "d"
	KeySnapshot2   = "D"
)

type InputMode string

const (
	InputNone    InputMode = ""
	InputCommand InputMode = "command"
	InputMessage InputMode = "message"
)

type InputBar struct {
	Mode   InputMode
	Target string
	Label  string
	Text   string
	Cursor int
}

func (b InputBar) Active() bool {
	return b.Mode != InputNone
}

func (b InputBar) Prompt() string {
	switch b.Mode {
	case InputCommand:
		return ":"
	case InputMessage:
		if b.Label != "" {
			return "message to " + b.Label + ": "
		}
		if b.Target != "" {
			return "message to selected target: "
		}
		return "message: "
	default:
		return ""
	}
}
