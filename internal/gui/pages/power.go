// Package pages 实现 Viper 界面的各个页面。
package pages

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/model"
	"viper/internal/gui/widgets"
)

// Power 画性能与功耗调控页面。
func Power(c *ui.Context, ctrl *controller.Controller, state *State) {
	overview := ctrl.Snapshot()

	ui.Column(c).Fill().Gap(16).Children(func() {
		powerHero(c, overview)

		if overview.Error != "" {
			widgets.ErrorNote(c, overview.Error)
		}

		// 1. 性能模式选择卡片网格
		powerModesSection(c, ctrl, state, overview)

		// 2. 显卡工作模式与独显直连
		gpuModeSection(c, ctrl, state, overview)

		// 3. 左右双栏：左边是电竞硬件开关，右边是核心时钟与负载监控
		ui.Row(c).Gap(16).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				gamingSwitchesCard(c, ctrl, overview)
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				telemetryCard(c, overview)
			})
		})
	})
}

// powerHero 画性能页顶部的招牌横幅。
func powerHero(c *ui.Context, overview model.Overview) {
	t := c.Theme()

	ui.Column(c).Padding(14, 16).Gap(10).Radius(14).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconActivity, t.Accent, 40)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "性能与功耗调控").FontSize(18).Bold().TextColor(t.Text)
				ui.Text(c, "动态功耗释放 · 显卡直连输出 · 游戏竞技硬件加速").FontSize(12).TextColor(t.TextMuted).NoWrap()
			})

			// 当前档位状态胶囊
			statusTint := t.Accent
			if overview.PowerMode == domain.PowerTurbo {
				statusTint = t.Danger
			} else if overview.PowerMode == domain.PowerPerformance {
				statusTint = t.Warning
			}
			widgets.Pill(c, fmt.Sprintf("当前档位: %s", model.FormatPowerMode(overview.PowerMode)), statusTint)
		})
	})
}

// powerModesSection 画性能档位选择卡片网格。
func powerModesSection(c *ui.Context, ctrl *controller.Controller, state *State, overview model.Overview) {
	t := c.Theme()
	modes := overview.PowerModes
	if len(modes) == 0 {
		modes = domain.AvailablePowerModes()
	}

	// 计算当前选中的档位
	activeMode := overview.PowerMode
	if state.PowerIndex >= 0 && state.PowerIndex < len(modes) {
		if modes[state.PowerIndex] == overview.PowerMode {
			state.PowerIndex = -1
		} else {
			activeMode = modes[state.PowerIndex]
		}
	}

	widgets.Card(c, "系统电源与性能档位", func() {
		ui.Row(c).Gap(12).Children(func() {
			for i, mode := range modes {
				idx := i
				targetMode := mode
				isActive := targetMode == activeMode

				borderCol := t.Border
				bgCol := t.Background
				tint := t.TextMuted
				if isActive {
					borderCol = t.Accent
					bgCol = t.Accent.Alpha(0.10)
					tint = t.Accent
					if targetMode == domain.PowerTurbo {
						borderCol = t.Danger
						bgCol = t.Danger.Alpha(0.10)
						tint = t.Danger
					} else if targetMode == domain.PowerPerformance {
						borderCol = t.Warning
						bgCol = t.Warning.Alpha(0.10)
						tint = t.Warning
					}
				}

				icon := widgets.IconActivity
				switch targetMode {
				case domain.PowerQuiet:
					icon = widgets.IconMoon
				case domain.PowerPerformance:
					icon = widgets.IconBolt
				case domain.PowerTurbo:
					icon = widgets.IconBolt
				case domain.PowerEco:
					icon = widgets.IconSun
				}

				card := ui.Column(c).Grow(1).MinWidth(0).Padding(12, 10).Gap(6).Radius(10).
					Background(bgCol).Border(1, borderCol).Cursor(ui.CursorPointer).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						widgets.IconBadge(c, icon, tint, 26)
						ui.Text(c, model.FormatPowerMode(targetMode)).FontSize(14).Bold().TextColor(t.Text)
						ui.Spacer(c)
						if isActive {
							ui.Box(c).Size(8, 8).Radius(4).Background(tint)
						}
					})
					ui.Text(c, model.PowerModeDesc(targetMode)).FontSize(11).TextColor(t.TextMuted)
				})

				if card.Clicked() {
					state.PowerIndex = idx
					ctrl.SetPowerMode(targetMode, func(err error) {
						if err != nil {
							state.PowerIndex = -1
						}
					})
				}
			}
		})
	})
}

// gpuModeSection 画显卡工作模式与独显直连调控卡片。
func gpuModeSection(c *ui.Context, ctrl *controller.Controller, state *State, overview model.Overview) {
	t := c.Theme()

	activeGPUMode := overview.GPUMode
	if state.GPUModeIndex >= 0 && state.GPUModeIndex < len(model.GPUModeLabels()) {
		if model.GPUModeFromIndex(state.GPUModeIndex) == overview.GPUMode {
			state.GPUModeIndex = -1
		} else {
			activeGPUMode = model.GPUModeFromIndex(state.GPUModeIndex)
		}
	}

	widgets.Card(c, "显卡输出与显示模式 (GPU Display Mode)", func() {
		// 顶部状态条：显示独显直连与待机状态
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if overview.DiscreteGPU {
				widgets.Pill(c, "纯独显直连输出 (dGPU)", widgets.ColorGPU)
			} else {
				widgets.Pill(c, "动态混合输出 (Optimus)", widgets.ColorCPU)
			}

			if overview.Monitor.GPUIdle {
				widgets.Pill(c, "独显处于省电休眠", t.TextMuted)
			} else {
				widgets.Pill(c, "独显核心活跃中", widgets.ColorGPU)
			}

			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("当前模式: %s", model.FormatGPUMode(overview.GPUMode))).FontSize(12).TextColor(t.TextMuted)
		})

		// 3 项显卡工作模式选项卡片
		ui.Row(c).Gap(12).Children(func() {
			gpuModes := []domain.GPUMode{
				domain.GPUModeHybrid,
				domain.GPUModeDiscrete,
				domain.GPUModeAuto,
			}
			for i, m := range gpuModes {
				idx := i
				target := m
				isActive := target == activeGPUMode

				borderCol := t.Border
				bgCol := t.Background
				tint := t.TextMuted
				if isActive {
					borderCol = widgets.ColorGPU
					bgCol = widgets.ColorGPU.Alpha(0.10)
					tint = widgets.ColorGPU
				}

				card := ui.Column(c).Grow(1).MinWidth(0).Padding(12).Gap(6).Radius(10).
					Background(bgCol).Border(1, borderCol).Cursor(ui.CursorPointer).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, model.FormatGPUMode(target)).FontSize(13).Bold().TextColor(t.Text)
						ui.Spacer(c)
						if isActive {
							ui.Box(c).Size(8, 8).Radius(4).Background(tint)
						}
					})
					ui.Text(c, model.GPUModeDesc(target)).FontSize(11).TextColor(t.TextMuted)
				})

				if card.Clicked() {
					state.GPUModeIndex = idx
					ctrl.SetGPUMode(target, func(err error) {
						if err != nil {
							state.GPUModeIndex = -1
						}
					})
				}
			}
		})
	})
}

// gamingSwitchesCard 画游戏竞技与系统硬件开关。
func gamingSwitchesCard(c *ui.Context, ctrl *controller.Controller, overview model.Overview) ui.Element {
	t := c.Theme()

	return widgets.Card(c, "电竞硬件加速与防护", func() {
		ui.Column(c).Gap(12).Children(func() {
			// 1. LCD 响应时间超速加速 (Overdrive)
			lcdOverdrive := overview.LCDOverdrive
			rowLCD := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Cursor(ui.CursorPointer).Children(func() {
				widgets.IconBadge(c, widgets.IconDisplay, t.Accent, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "LCD 响应时间超速 (Overdrive)").FontSize(13).Bold().TextColor(t.Text)
					ui.Text(c, "液晶面板加速至 3ms 响应，显著抑制高速竞技画面拖影").FontSize(11).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &lcdOverdrive).Label("LCD 响应时间超速 (Overdrive)")
				if sw.Changed() {
					ctrl.SetLCDOverdrive(lcdOverdrive, nil)
				}
			})
			if rowLCD.Clicked() {
				ctrl.SetLCDOverdrive(!lcdOverdrive, nil)
			}

			// 2. CoolBoost 强冷散热加速
			coolBoost := overview.FanState.CoolBoost
			rowCool := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Cursor(ui.CursorPointer).Children(func() {
				widgets.IconBadge(c, widgets.IconFan, widgets.ColorBoard, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "CoolBoost 强冷加速").FontSize(13).Bold().TextColor(t.Text)
					ui.Text(c, "激进提升双 AeroBlade 风扇进风量，保障满载不降频").FontSize(11).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &coolBoost).Label("CoolBoost 强冷加速")
				if sw.Changed() {
					ctrl.SetCoolBoost(coolBoost, nil)
				}
			})
			if rowCool.Clicked() {
				ctrl.SetCoolBoost(!coolBoost, nil)
			}
		})
	})
}

// telemetryCard 画核心频率与实时负载监控看板。
func telemetryCard(c *ui.Context, overview model.Overview) ui.Element {
	t := c.Theme()
	monitor := overview.Monitor

	return widgets.Card(c, "核心时钟与负载看板", func() {
		ui.Column(c).Gap(12).Children(func() {
			// CPU 核心指标卡
			ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Children(func() {
				widgets.IconBadge(c, widgets.IconCPU, widgets.ColorCPU, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "CPU 处理器").FontSize(13).Bold().TextColor(t.Text)
					clockInfo := model.FormatMHz(monitor.CPUFrequencyMHz)
					if monitor.CPUMaxFrequencyMHz > 0 {
						clockInfo += " / 睿频 " + model.FormatMHz(monitor.CPUMaxFrequencyMHz)
					}
					ui.Text(c, clockInfo).FontSize(11).TextColor(t.TextMuted)
				})
				ui.Column(c).AlignItems(ui.End).Gap(2).Children(func() {
					ui.Text(c, model.FormatPercent(monitor.CPUUsagePercent)).FontSize(14).Bold().TextColor(t.Text)
					ui.Text(c, model.FormatCelsius(monitor.CPUTemperatureC)).FontSize(11).TextColor(widgets.LevelColor(c, model.TemperatureLevel(monitor.CPUTemperatureC)))
				})
			})

			// GPU 核心指标卡
			ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(10).Radius(10).
				Background(t.Background).Border(1, t.Border).Children(func() {
				widgets.IconBadge(c, widgets.IconGPU, widgets.ColorGPU, 32)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "GPU 独立显卡").FontSize(13).Bold().TextColor(t.Text)
					gpuClock := model.FormatMHz(monitor.GPUFrequencyMHz)
					if monitor.GPUMaxFrequencyMHz > 0 {
						gpuClock += " / 极频 " + model.FormatMHz(monitor.GPUMaxFrequencyMHz)
					}
					ui.Text(c, gpuClock).FontSize(11).TextColor(t.TextMuted)
				})
				ui.Column(c).AlignItems(ui.End).Gap(2).Children(func() {
					ui.Text(c, model.FormatPercent(float64(monitor.GPUUsagePercent))).FontSize(14).Bold().TextColor(t.Text)
					ui.Text(c, model.FormatCelsius(monitor.GPUTemperatureC)).FontSize(11).TextColor(widgets.LevelColor(c, model.TemperatureLevel(monitor.GPUTemperatureC)))
				})
			})

			// 底部小统计瓦片：内存与主板温度
			ui.Row(c).Gap(10).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Padding(8, 10).Radius(8).
					Background(t.Background).Border(1, t.Border).Children(func() {
					ui.Text(c, "内存占用").FontSize(11).TextColor(t.TextMuted)
					ui.Text(c, model.FormatPercent(monitor.RAMUsagePercent)).FontSize(14).Bold().TextColor(t.Text)
				})
				ui.Column(c).Grow(1).MinWidth(0).Padding(8, 10).Radius(8).
					Background(t.Background).Border(1, t.Border).Children(func() {
					ui.Text(c, "系统温度").FontSize(11).TextColor(t.TextMuted)
					ui.Text(c, model.FormatCelsius(monitor.SystemTempC)).FontSize(14).Bold().TextColor(t.Text)
				})
			})
		})
	})
}
