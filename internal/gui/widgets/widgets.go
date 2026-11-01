package widgets

import (
	"github.com/egoist/mygo/ui"

	"viper/internal/gui/model"
	"viper/internal/gui/theme"
)

var (
	ColorCPU    = ui.Hex("#3b82f6")
	ColorGPU    = ui.Hex("#10b981")
	ColorMemory = ui.Hex("#f59e0b")
	ColorNet    = ui.Hex("#a855f7")
	ColorBoard  = ui.Hex("#f97316")
)

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

func IconBadge(c *ui.Context, icon Icon, tint ui.Color, size float32) ui.Element {
	return ui.Box(c).Size(size, size).Radius(size * 0.28).
		Background(tint.Alpha(0.14)).
		Children(func() {
			ui.Row(c).Fill().AlignItems(ui.Center).Center().Children(func() {
				IconElement(c, icon, size*0.58, tint)
			})
		})
}

func Pill(c *ui.Context, text string, tint ui.Color) ui.Element {
	t := c.Theme()
	return ui.Row(c).Gap(6).AlignItems(ui.Center).
		Padding(6, 10).Radius(999).Background(tint.Alpha(0.12)).Children(func() {
		ui.Box(c).Size(7, 7).Radius(4).Background(tint)
		ui.Text(c, text).FontSize(11).TextColor(t.Text)
	})
}

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

func SectionTitle(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(14).Bold().TextColor(t.Text)
}

func Note(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(12).TextColor(t.TextMuted)
}

func ErrorNote(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(12).TextColor(t.Danger)
}

func ModePicker(c *ui.Context, selected *int) ui.Element {
	return ui.Segmented(c, selected, theme.Labels()...).Label("外观")
}

func LabelsOf(modes []theme.Mode) []string {
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, mode.Label())
	}
	return out
}

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
