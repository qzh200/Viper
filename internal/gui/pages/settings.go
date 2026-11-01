package pages

import (
	"github.com/egoist/mygo/ui"

	"viper/internal/gui/controller"
	"viper/internal/gui/model"
	"viper/internal/gui/theme"
	"viper/internal/gui/widgets"
)

func Settings(c *ui.Context, ctrl *controller.Controller, state *State, onThemeChange func(theme.Mode)) {
	overview := ctrl.Snapshot()

	ui.Column(c).Fill().Gap(16).Children(func() {
		settingsHero(c, overview)

		if overview.Error != "" {
			widgets.ErrorNote(c, overview.Error)
		}

		appearanceSection(c, ctrl, onThemeChange)

		hardwareFeaturesSection(c, ctrl, overview)

		servicesSection(c, overview)

		aboutSection(c)
	})
}

func settingsHero(c *ui.Context, overview model.Overview) {
	t := c.Theme()

	ui.Column(c).Padding(14, 16).Gap(10).Radius(14).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconGear, t.Accent, 40)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "系统与控制中心设置").FontSize(18).Bold().TextColor(t.Text)
				ui.Text(c, "外观主题偏好 · 原厂外设调校 · 服务连接与驱动诊断").FontSize(12).TextColor(t.TextMuted).NoWrap()
			})

			dotCol := t.Danger
			dotText := "原厂服务未连接"
			if overview.Connected {
				dotCol = t.Success
				dotText = "原厂服务正常运行"
			}
			widgets.Pill(c, dotText, dotCol)
		})
	})
}

func appearanceSection(c *ui.Context, ctrl *controller.Controller, onThemeChange func(theme.Mode)) {
	t := c.Theme()
	currentMode := ctrl.ThemeMode()

	widgets.Card(c, "界面外观与主题偏好", func() {
		ui.Column(c).Gap(12).Children(func() {
			ui.Row(c).Gap(12).Children(func() {
				modes := []struct {
					mode theme.Mode
					name string
					desc string
					icon widgets.Icon
				}{
					{
						mode: theme.ModeAuto,
						name: "跟随系统",
						desc: "自动检测并匹配 Windows 浅色或深色主题外观",
						icon: widgets.IconDisplay,
					},
					{
						mode: theme.ModeLight,
						name: "浅色模式",
						desc: "明亮清晰的日间办公与阅读配色界面",
						icon: widgets.IconSun,
					},
					{
						mode: theme.ModeDark,
						name: "深色模式",
						desc: "沉浸式暗色电竞风格，低眩光呵护夜间视力",
						icon: widgets.IconMoon,
					},
				}

				for _, item := range modes {
					targetMode := item.mode
					isActive := targetMode == currentMode

					borderCol := t.Border
					bgCol := t.Background
					tint := t.TextMuted
					if isActive {
						borderCol = t.Accent
						bgCol = t.Accent.Alpha(0.10)
						tint = t.Accent
					}

					card := ui.Column(c).Grow(1).MinWidth(0).Padding(14, 12).Gap(6).Radius(10).
						Background(bgCol).Border(1, borderCol).Cursor(ui.CursorPointer).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							widgets.IconBadge(c, item.icon, tint, 28)
							ui.Text(c, item.name).FontSize(13).Bold().TextColor(t.Text)
							ui.Spacer(c)
							if isActive {
								ui.Box(c).Size(8, 8).Radius(4).Background(tint)
							}
						})
						ui.Text(c, item.desc).FontSize(11).TextColor(t.TextMuted)
					})

					if card.Clicked() && targetMode != currentMode {
						if err := ctrl.SetThemeMode(targetMode); err == nil {
							if onThemeChange != nil {
								onThemeChange(targetMode)
							}
						}
					}
				}
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8, 12).Radius(8).
				Background(t.Background).Children(func() {
				widgets.IconElement(c, widgets.IconActivity, 14, t.Accent)
				ui.Text(c, "原生动效引擎: 25ms 硬件指标弹簧插值缓动已启用 · 界面线程零卡顿").FontSize(11).TextColor(t.TextMuted)
			})
		})
	})
}

func hardwareFeaturesSection(c *ui.Context, ctrl *controller.Controller, overview model.Overview) {
	t := c.Theme()

	widgets.Card(c, "原厂硬件特色调校", func() {
		ui.Column(c).Gap(12).Children(func() {
			bootSound := overview.BootSound
			rowBoot := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Cursor(ui.CursorPointer).Children(func() {
				widgets.IconBadge(c, widgets.IconBolt, t.Accent, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "开机专属问候音效 (Boot Sound)").FontSize(13).Bold().TextColor(t.Text)
					ui.Text(c, "开机自检点亮屏幕时播放 Predator 原厂标志开机声效").FontSize(11).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &bootSound).Label("开机专属问候音效")
				if sw.Changed() {
					ctrl.SetBootSound(bootSound, nil)
				}
			})
			if rowBoot.Clicked() {
				ctrl.SetBootSound(!bootSound, nil)
			}

			backlightSleep := overview.BacklightTimeout
			rowSleep := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Cursor(ui.CursorPointer).Children(func() {
				widgets.IconBadge(c, widgets.IconMoon, widgets.ColorMemory, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "键盘背光 30 秒空闲自动休眠").FontSize(13).Bold().TextColor(t.Text)
					ui.Text(c, "系统闲置 30 秒后自动熄灭键盘背光以节约能耗，敲击任意按键即刻唤醒").FontSize(11).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &backlightSleep).Label("键盘背光 30 秒空闲自动休眠")
				if sw.Changed() {
					ctrl.SetBacklightTimeout(backlightSleep, nil)
				}
			})
			if rowSleep.Clicked() {
				ctrl.SetBacklightTimeout(!backlightSleep, nil)
			}
		})
	})
}

func servicesSection(c *ui.Context, overview model.Overview) {
	t := c.Theme()

	widgets.Card(c, "驱动服务与连接状态", func() {
		ui.Row(c).Gap(12).Children(func() {
			instance := overview.Instance
			if instance == "" {
				instance = "AcerAgentService (未载入)"
			}
			ui.Column(c).Grow(1).MinWidth(0).Padding(10).Gap(4).Radius(8).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Text(c, "后端硬件驱动").FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, instance).FontSize(13).Bold().TextColor(t.Text).NoWrap()
			})

			route := overview.Route
			if route == "" {
				route = "TCP 127.0.0.1:46933"
			}
			ui.Column(c).Grow(1).MinWidth(0).Padding(10).Gap(4).Radius(8).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Text(c, "协议通讯路由").FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, route).FontSize(13).Bold().TextColor(t.Text).NoWrap()
			})

			ui.Column(c).Grow(1).MinWidth(0).Padding(10).Gap(4).Radius(8).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Text(c, "本地配置存储").FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, "config.json (原子写保护)").FontSize(13).Bold().TextColor(t.Text).NoWrap()
			})

			timeStr := "—"
			if !overview.Updated.IsZero() {
				timeStr = overview.Updated.Format("15:04:05")
			}
			ui.Column(c).Grow(1).MinWidth(0).Padding(10).Gap(4).Radius(8).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Text(c, "最近心跳刷新").FontSize(11).TextColor(t.TextMuted)
				ui.Text(c, timeStr).FontSize(13).Bold().TextColor(t.Text).NoWrap()
			})
		})
	})
}

func aboutSection(c *ui.Context) {
	t := c.Theme()

	widgets.Card(c, "关于", func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconDisplay, t.Accent, 32)
			ui.Column(c).Gap(2).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, "Viper 控制中心").FontSize(14).Bold().TextColor(t.Text)
					ui.Text(c, "v1.0.0").FontSize(11).TextColor(t.TextMuted)
				})
				ui.Text(c, "轻量原生宏碁游戏本控制套件 · 纯 Go 自绘").FontSize(11).TextColor(t.TextMuted)
			})
		})
	})
}
