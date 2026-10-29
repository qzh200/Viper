// Package widgets 提供界面层复用的小组件。
//
// 它们都建立在 mygo 的原生控件之上：MyGo 自己画这些控件，不经过网页，
// 因此界面在 Windows 上是真正的原生窗口。
package widgets

import (
	"github.com/egoist/mygo/ui"

	"viper/internal/gui/model"
	"viper/internal/gui/theme"
)

// 几支固定颜色：亮色与暗色里都一样好看。
var (
	// ColorCPU 是处理器的颜色。
	ColorCPU = ui.Hex("#3b82f6")
	// ColorGPU 是显卡的颜色。
	ColorGPU = ui.Hex("#10b981")
	// ColorMemory 是内存的颜色。
	ColorMemory = ui.Hex("#f59e0b")
	// ColorNet 是网络的颜色。
	ColorNet = ui.Hex("#a855f7")
	// ColorBoard 是主板的温度色。
	ColorBoard = ui.Hex("#f97316")
)

// Card 画一张卡片：圆角、描边与一道浅影，里面放标题与内容。
func Card(c *ui.Context, title string, body func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(16).Gap(12).Radius(12).Background(t.Surface).Border(1, t.Border).
		Shadow(0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(14).Bold().TextColor(t.Text)
		}
		body()
	})
}

// IconBadge 画一枚图标：圆角方块底色上放一枚线条图标。
func IconBadge(c *ui.Context, icon Icon, tint ui.Color, size float32) ui.Element {
	return ui.Box(c).Size(size, size).Radius(size * 0.28).
		Background(tint.Alpha(0.14)).
		Children(func() {
			ui.Row(c).Fill().AlignItems(ui.Center).Center().Children(func() {
				IconElement(c, icon, size*0.58, tint)
			})
		})
}

// Pill 画一个药丸：一个小圆点加一段字，用来说明状态。
func Pill(c *ui.Context, text string, tint ui.Color) ui.Element {
	t := c.Theme()
	return ui.Row(c).Gap(6).AlignItems(ui.Center).
		Padding(6, 10).Radius(999).Background(tint.Alpha(0.12)).Children(func() {
		ui.Box(c).Size(7, 7).Radius(4).Background(tint)
		ui.Text(c, text).FontSize(11).TextColor(t.Text)
	})
}

// StatTile 画一个小统计块：图标、名字、读数与补充说明。
func StatTile(c *ui.Context, icon Icon, label, value, detail string, tint ui.Color) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(8).Padding(14).Radius(10).
		Background(t.Background).Border(1, t.Border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			IconElement(c, icon, 16, tint)
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		})
		ui.Text(c, value).FontSize(20).Bold().TextColor(t.Text).NoWrap()
		if detail != "" {
			ui.Text(c, detail).FontSize(11).TextColor(t.TextMuted).NoWrap()
		}
	})
}

// SectionTitle 画一个小节的标题。
func SectionTitle(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(14).Bold().TextColor(t.Text)
}

// Note 画一行补充说明，用较淡的颜色。
func Note(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(12).TextColor(t.TextMuted)
}

// ErrorNote 画一行错误说明，用危险色。
func ErrorNote(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(12).TextColor(t.Danger)
}

// ModePicker 画外观模式的选择器。selected 是 Modes 里的下标，
// 界面上按 Modes 的顺序摆开。调用者拿着返回值问 Changed，
// 就知道用户是不是刚点了别的。
func ModePicker(c *ui.Context, selected *int) ui.Element {
	return ui.Segmented(c, selected, theme.Labels()...).Label("外观")
}

// LabelsOf 把一组模式写成界面上显示的名字。
func LabelsOf(modes []theme.Mode) []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, mode.Label())
	}
	return out
}

// LevelColor 按指标的严重程度挑颜色。
func LevelColor(c *ui.Context, level model.Level) ui.Color {
	t := c.Theme()
	switch level {
	case model.LevelWarn:
		return t.Warning
	case model.LevelDanger:
		return t.Danger
	default:
		return t.Text
	}
}
