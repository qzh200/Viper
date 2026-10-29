// Package theme 定义 Viper 界面的外观模式：亮色、暗色与自动。
//
// 这一层不碰任何图形界面，也不碰硬件：它只回答两个问题——
// 用户选的是哪一种模式，以及这一模式此刻该显示成什么样。
package theme

import "strings"

// Mode 是用户选择的外观模式。
type Mode string

const (
	// ModeAuto 让界面跟随系统的外观。
	ModeAuto Mode = "auto"
	// ModeLight 让界面用亮色，不理会系统的外观。
	ModeLight Mode = "light"
	// ModeDark 让界面用暗色，不理会系统的外观。
	ModeDark Mode = "dark"
)

// Appearance 是模式解析之后的实际外观。
type Appearance string

const (
	// AppearanceLight 是亮色外观。
	AppearanceLight Appearance = "light"
	// AppearanceDark 是暗色外观。
	AppearanceDark Appearance = "dark"
)

// modes 是界面上按顺序摆出的模式：自动在最前，因为它是最常用的那个。
var modes = []Mode{ModeAuto, ModeLight, ModeDark}

// Modes 返回界面上可选的外观模式，顺序即展示顺序。
func Modes() []Mode {
	out := make([]Mode, len(modes))
	copy(out, modes)
	return out
}

// Parse 解析一段文本，得到外观模式。空串与不认识的值都不算合法，
// 调用方应当退回自动模式，而不是替用户猜一个。
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

// Valid 报告这个模式是不是用户能选的那三种之一。
func (m Mode) Valid() bool {
	switch m {
	case ModeAuto, ModeLight, ModeDark:
		return true
	default:
		return false
	}
}

// Label 返回模式在界面上显示的名字。
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

// Next 返回下一个模式，用于「换一个」这类循环切换。
func (m Mode) Next() Mode {
	for i, candidate := range modes {
		if candidate == m {
			return modes[(i+1)%len(modes)]
		}
	}
	return ModeAuto
}

// Resolve 算出此刻该显示成什么样。自动模式跟着系统走，
// 亮色与暗色则不听系统的。
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

// Dark 报告这个外观是不是暗色。
func (a Appearance) Dark() bool { return a == AppearanceDark }

// Labels 返回所有模式在界面上显示的名字，顺序与 Modes 一致。
func Labels() []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, mode.Label())
	}
	return out
}

// Index 返回模式在 Modes 里的位置；不在其中的模式按自动处理。
func Index(mode Mode) int {
	for i, candidate := range modes {
		if candidate == mode {
			return i
		}
	}
	return 0
}

// At 返回 Modes 里第 i 个模式，越界时返回自动模式。
func At(i int) Mode {
	if i < 0 || i >= len(modes) {
		return ModeAuto
	}
	return modes[i]
}
