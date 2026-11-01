package theme

import "strings"

type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeLight Mode = "light"
	ModeDark  Mode = "dark"
)

type Appearance string

const (
	AppearanceLight Appearance = "light"
	AppearanceDark  Appearance = "dark"
)

var modes = []Mode{ModeAuto, ModeLight, ModeDark}

func Modes() []Mode {
	out := make([]Mode, len(modes))
	copy(out, modes)
	return out
}

func Parse(text string) (Mode, bool) {
	switch Mode(strings.ToLower(strings.TrimSpace(text))) {
	case ModeAuto:
		return ModeAuto, true
	case ModeLight:
		return ModeLight, true
	case ModeDark:
		return ModeDark, true
	default:
		return ModeAuto, false
	}
}

func (m Mode) Valid() bool {
	switch m {
	case ModeAuto, ModeLight, ModeDark:
		return true
	default:
		return false
	}
}

func (m Mode) Label() string {
	switch m {
	case ModeLight:
		return "亮色"
	case ModeDark:
		return "暗色"
	default:
		return "自动"
	}
}

func (m Mode) Next() Mode {
	for i, candidate := range modes {
		if candidate == m {
			return modes[(i+1)%len(modes)]
		}
	}
	return ModeAuto
}

func Resolve(mode Mode, systemDark bool) Appearance {
	switch mode {
	case ModeLight:
		return AppearanceLight
	case ModeDark:
		return AppearanceDark
	default:
		if systemDark {
			return AppearanceDark
		}
		return AppearanceLight
	}
}

func (a Appearance) Dark() bool { return a == AppearanceDark }

func Labels() []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, mode.Label())
	}
	return out
}

func Index(mode Mode) int {
	for i, candidate := range modes {
		if candidate == mode {
			return i
		}
	}
	return 0
}

func At(i int) Mode {
	if i < 0 || i >= len(modes) {
		return ModeAuto
	}
	return modes[i]
}
