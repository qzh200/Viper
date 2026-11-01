package shell

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"viper/internal/gui/controller"
	"viper/internal/gui/pages"
	"viper/internal/gui/theme"
	"viper/internal/gui/widgets"
)

type Page struct {
	ID    string
	Label string
	Icon  widgets.Icon
}

var Pages = []Page{
	{ID: "home", Label: "总览", Icon: widgets.IconDisplay},
	{ID: "fans", Label: "风扇", Icon: widgets.IconFan},
	{ID: "lighting", Label: "灯效", Icon: widgets.IconBolt},
	{ID: "power", Label: "性能", Icon: widgets.IconActivity},
	{ID: "settings", Label: "设置", Icon: widgets.IconGear},
}

type Shell struct {
	ctrl *controller.Controller

	page       string
	themeIndex int
	state      *pages.State
	services   ui.Services
	started    bool
	apply      func(theme.Mode)
}

type Option func(*Shell)

func WithApply(apply func(theme.Mode)) Option {
	return func(s *Shell) {
		if apply != nil {
			s.apply = apply
		}
	}
}

func WithAppearance(mode theme.Mode) Option {
	return func(s *Shell) {
		s.themeIndex = theme.Index(mode)
	}
}

func WithPage(page string) Option {
	return func(s *Shell) {
		if page != "" {
			s.page = page
		}
	}
}

func New(ctrl *controller.Controller, opts ...Option) *Shell {
	window := &Shell{
		ctrl:       ctrl,
		page:       Pages[0].ID,
		state:      pages.NewState(),
		apply:      Apply,
		themeIndex: theme.Index(ctrl.ThemeMode()),
	}
	for _, opt := range opts {
		opt(window)
	}
	return window
}

func (s *Shell) View(c *ui.Context) {
	s.services = c.Services()
	s.startWorker()

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		s.sidebar(c)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			s.header(c)
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).FillWidth().Padding(20).Children(func() {
					s.content(c)
				})
			})
		})
	})
}

func (s *Shell) sidebar(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Width(200).FillHeight().Padding(14, 12).Gap(6).
		Background(t.Surface).
		Draw(func(p *ui.Painter, r ui.Rect) {
			p.Line(r.X+r.W-0.5, r.Y, r.X+r.W-0.5, r.Y+r.H, 1, t.Border)
		}).
		Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(8, 8).Radius(10).
				Background(t.Background).Children(func() {
				widgets.IconBadge(c, widgets.IconDisplay, t.Accent, 32)
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "Viper").FontSize(15).Bold().TextColor(t.Text)
					ui.Text(c, "控制中心").FontSize(10).TextColor(t.TextMuted)
				})
			})

			ui.Text(c, "控制").FontSize(10).TextColor(t.TextMuted).Padding(2, 10)
			for _, page := range Pages {
				s.navItem(c, page)
			}

			ui.Spacer(c)

			ui.Row(c).Gap(7).AlignItems(ui.Center).Padding(10, 8).Children(func() {
				dot := t.Danger
				if s.ctrl.Snapshot().Connected {
					dot = t.Success
				}
				ui.Box(c).Size(7, 7).Radius(4).Background(dot)
				ui.Text(c, "原厂服务").FontSize(11).TextColor(t.TextMuted)
			})
		})
}

func (s *Shell) navItem(c *ui.Context, page Page) {
	t := c.Theme()
	active := s.page == page.ID

	tint := t.TextMuted
	fill := t.Surface
	if active {
		tint = t.Accent
		fill = t.Accent.Alpha(0.13)
	}

	item := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10, 10).Radius(9).
		Background(fill).Cursor(ui.CursorPointer).Children(func() {
		widgets.IconElement(c, page.Icon, 16, tint)
		ui.Text(c, page.Label).FontSize(13).TextColor(tint)
	})
	if item.Clicked() {
		s.page = page.ID
	}
}

func (s *Shell) header(c *ui.Context) {
	t := c.Theme()

	ui.Row(c).FillWidth().Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).Children(func() {
		ui.Text(c, "Viper").FontSize(15).Bold().TextColor(t.Text)
		ui.Text(c, "/").FontSize(13).TextColor(t.TextMuted)
		ui.Text(c, s.pageTitle()).FontSize(13).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, "外观").FontSize(12).TextColor(t.TextMuted)
		s.themePicker(c)
	})
}

func (s *Shell) pageTitle() string {
	for _, page := range Pages {
		if page.ID == s.page {
			return page.Label
		}
	}
	return Pages[0].Label
}

func (s *Shell) themePicker(c *ui.Context) {
	index := s.themeIndex
	picker := widgets.ModePicker(c, &index)
	if !picker.Changed() {
		return
	}
	if index < 0 || index >= len(theme.Modes()) {
		return
	}
	s.themeIndex = index
	mode := theme.At(index)
	if err := s.ctrl.SetThemeMode(mode); err != nil {
		return
	}
	s.apply(mode)
}

func (s *Shell) content(c *ui.Context) {
	switch s.page {
	case "home":
		pages.Home(c, s.ctrl, s.state)
	case "fans":
		pages.Fans(c, s.ctrl, s.state)
	case "lighting":
		pages.Lighting(c, s.ctrl, s.state)
	case "power":
		pages.Power(c, s.ctrl, s.state)
	case "settings":
		pages.Settings(c, s.ctrl, s.state, s.apply)
	default:
		widgets.Note(c, "这一页还在路上。")
	}
}

func (s *Shell) startWorker() {
	if s.started {
		return
	}
	s.started = true
	s.ctrl.Start(controller.DefaultRefresh, func() {
		s.services.Invalidate()
	})
}

func Apply(mode theme.Mode) {
	mygo.Theme.SetSource(Source(mode))
}

func Source(mode theme.Mode) mygo.ThemeSource {
	switch mode {
	case theme.ModeLight:
		return mygo.ThemeLight
	case theme.ModeDark:
		return mygo.ThemeDark
	default:
		return mygo.ThemeSystem
	}
}
