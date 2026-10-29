// Package shell 装配 Viper 的窗口：侧边栏、标题栏与内容区。
//
// 它是界面层里唯一知道「整扇窗口长什么样」的地方：页面只管自己那一块，
// 主题的切换也由它落到 mygo 上。
package shell

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"viper/internal/gui/controller"
	"viper/internal/gui/pages"
	"viper/internal/gui/theme"
	"viper/internal/gui/widgets"
)

// Page 是侧边栏里的一个页面。
type Page struct {
	// ID 是页面的标识，也是侧边栏记住选中项的键。
	ID string
	// Label 是侧边栏上显示的名字。
	Label string
	// Icon 是页面的小图标。
	Icon widgets.Icon
}

// Pages 是 Viper 目前有的页面。
var Pages = []Page{
	{ID: "home", Label: "总览", Icon: widgets.IconDisplay},
	{ID: "fans", Label: "风扇", Icon: widgets.IconFan},
	{ID: "lighting", Label: "灯效", Icon: widgets.IconBolt},
	{ID: "power", Label: "性能", Icon: widgets.IconActivity},
	{ID: "settings", Label: "设置", Icon: widgets.IconGear},
}

// Shell 是一扇 Viper 窗口。
type Shell struct {
	ctrl *controller.Controller

	// page 是当前显示的页面。
	page string
	// themeIndex 是外观模式在 theme.Modes 里的下标。
	themeIndex int
	// state 是页面之间共享的状态。
	state *pages.State
	// services 让后台协程能请窗口重画。
	services ui.Services
	// started 记着后台协程起没起。
	started bool
	// apply 把外观模式落到窗口上；换掉它，测试就不必真去动 mygo。
	apply func(theme.Mode)
}

// Option 配置一扇窗口。
type Option func(*Shell)

// WithApply 换掉外观模式的落地方式。测试用它来确认界面确实把选择交出去了，
// 而不必真的去改 mygo 的全局外观。
func WithApply(apply func(theme.Mode)) Option {
	return func(s *Shell) {
		if apply != nil {
			s.apply = apply
		}
	}
}

// WithAppearance 指定这一次窗口用哪种外观，不改存下来的那个。
func WithAppearance(mode theme.Mode) Option {
	return func(s *Shell) {
		s.themeIndex = theme.Index(mode)
	}
}

// WithPage 指定这一次窗口初始打开哪一页。
func WithPage(page string) Option {
	return func(s *Shell) {
		if page != "" {
			s.page = page
		}
	}
}

// New 创建一扇窗口的骨架。
func New(ctrl *controller.Controller, opts ...Option) *Shell {
	window := &Shell{
		ctrl:       ctrl,
		page:       Pages[0].ID,
		state:      pages.NewState(),
		apply:      Apply,
		themeIndex: theme.Index(ctrl.ThemeMode()),
	} // 选项排在后面，因此能盖掉存下来的那个外观。
	for _, opt := range opts {
		opt(window)
	}
	return window
}

// View 画出整扇窗口。mygo 每需要一帧就调用它一次。
func (s *Shell) View(c *ui.Context) {
	s.services = c.Services()
	s.startWorker()

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		s.sidebar(c)
		// 内容那一列要占满剩下的宽：Grow 给它宽度，Stretch 拉满高度。
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

// sidebar 画左侧的导航：上面是招牌，下面是菜单，最下面是连接状态。
func (s *Shell) sidebar(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Width(200).FillHeight().Padding(14, 12).Gap(6).
		Background(t.Surface).
		Draw(func(p *ui.Painter, r ui.Rect) {
			// 右边一条细线，把侧边栏和内容分开。
			p.Line(r.X+r.W-0.5, r.Y, r.X+r.W-0.5, r.Y+r.H, 1, t.Border)
		}).
		Children(func() {
			// 招牌。
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

			// 最下面：连着没有。
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

// navItem 画导航里的一项。选中的那项有底色与强调色。
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

// header 画标题栏：程序名、当前页面与外观模式选择器。
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

// pageTitle 返回当前页面的名字。
func (s *Shell) pageTitle() string {
	for _, page := range Pages {
		if page.ID == s.page {
			return page.Label
		}
	}
	return Pages[0].Label
}

// themePicker 画外观模式的选择器，并把选中的模式落到 mygo 上。
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

// content 画当前页面的内容。
func (s *Shell) content(c *ui.Context) {
	switch s.page {
	case "home":
		pages.Home(c, s.ctrl, s.state)
	case "fans":
		pages.Fans(c, s.ctrl, s.state)
	default:
		widgets.Note(c, "这一页还在路上。")
	}
}

// startWorker 头一帧时起后台协程，按周期刷新快照并请窗口重画。
func (s *Shell) startWorker() {
	if s.started {
		return
	}
	s.started = true
	s.ctrl.Start(controller.DefaultRefresh, func() {
		s.services.Invalidate()
	})
}

// Apply 把外观模式落到 MyGo 上：亮色、暗色，或者跟着系统。
//
// mygo 的原生控件本来就跟着系统的外观走，所以「自动」就是交还给系统，
// 亮色与暗色则由这里强制。
func Apply(mode theme.Mode) {
	mygo.Theme.SetSource(Source(mode))
}

// Source 把外观模式翻译成 mygo 的外观来源。
//
// 外观是整个程序的事，因此由桌面（MyGo）来落地：
// mygo 的原生控件本来就跟着系统的外观走，所以「自动」就是交还给系统，
// 亮色与暗色则由这里强制。
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
