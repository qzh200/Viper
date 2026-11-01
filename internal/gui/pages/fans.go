package pages

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/model"
	"viper/internal/gui/widgets"
)

func Fans(c *ui.Context, ctrl *controller.Controller, state *State) {
	overview := ctrl.Snapshot()
	fanState := overview.FanState
	monitor := overview.Monitor

	if !state.CPUDutyInited {
		if fanState.CPUPercent > 0 {
			state.CPUDuty = float64(fanState.CPUPercent)
		} else {
			state.CPUDuty = 50
		}
		state.CPUDutyInited = true
	}
	if !state.GPUDutyInited {
		if fanState.GPUPercent > 0 {
			state.GPUDuty = float64(fanState.GPUPercent)
		} else {
			state.GPUDuty = 50
		}
		state.GPUDutyInited = true
	}

	ui.Column(c).Fill().Gap(16).Children(func() {
		fansHero(c, overview, ctrl, state)

		if overview.Error != "" {
			widgets.ErrorNote(c, overview.Error)
		}

		ui.Row(c).Gap(16).Children(func() {
			fanUnitCard(c, ctrl, state, overview, "CPU", widgets.IconCPU, widgets.ColorCPU,
				fanState.CPUMode, fanState.CPUPercent, monitor.CPUFanRPM, monitor.CPUTemperatureC,
				model.FormatPercent(monitor.CPUUsagePercent), "负载")

			fanUnitCard(c, ctrl, state, overview, "GPU", widgets.IconGPU, widgets.ColorGPU,
				fanState.GPUMode, fanState.GPUPercent, monitor.GPUFanRPM, monitor.GPUTemperatureC,
				model.FormatMHz(monitor.GPUFrequencyMHz), "频率")
		})

		fanStrategyCard(c)
	})
}

func fansHero(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()
	fanState := overview.FanState

	currentModeIndex := model.FanModeIndex(fanState.CPUMode)
	if state.FanModeIndex >= 0 {
		if model.FanModeFromIndex(state.FanModeIndex) == fanState.CPUMode {
			state.FanModeIndex = -1
		} else {
			currentModeIndex = state.FanModeIndex
		}
	}

	ui.Column(c).Padding(14, 16).Gap(10).Radius(14).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconFan, t.Accent, 40)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "风扇与散热控制").FontSize(18).Bold().TextColor(t.Text)
				ui.Text(c, "双 AeroBlade 3D 金属风扇 · 动态温控调校与强冷加速").FontSize(12).TextColor(t.TextMuted).NoWrap()
			})

			coolBoostOn := fanState.CoolBoost
			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 12).Radius(10).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "CoolBoost 强冷").FontSize(12).Bold().TextColor(t.Text)
					ui.Text(c, "提升全速进风量").FontSize(10).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &coolBoostOn)
				if sw.Changed() {
					ctrl.SetCoolBoost(coolBoostOn, nil)
				}
			})

			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "运转模式").FontSize(11).TextColor(t.TextMuted)
				modeIdx := currentModeIndex
				picker := ui.Segmented(c, &modeIdx, model.FanModeLabels()...).Label("运转模式")
				if picker.Changed() && modeIdx >= 0 && modeIdx < len(model.FanModeLabels()) {
					state.FanModeIndex = modeIdx
					targetMode := model.FanModeFromIndex(modeIdx)
					cfg := domain.FanConfig{
						CPUMode:    targetMode,
						GPUMode:    targetMode,
						CPUPercent: int(state.CPUDuty),
						GPUPercent: int(state.GPUDuty),
						CoolBoost:  fanState.CoolBoost,
					}
					ctrl.SetFanConfig(cfg, func(err error) {
						if err != nil {
							state.FanModeIndex = -1
						}
					})
				}
			})
		})
	})
}

func fanUnitCard(c *ui.Context, ctrl *controller.Controller, state *State, overview model.Overview,
	name string, icon widgets.Icon, tint ui.Color, mode domain.FanMode, duty int,
	rpm int, tempC int, auxValue, auxLabel string) ui.Element {
	t := c.Theme()

	modeText := "自动调速"
	switch mode {
	case domain.FanMax:
		modeText = "全速运转"
	case domain.FanCustom:
		currentDuty := duty
		if name == "CPU" && state.CPUDutyInited {
			currentDuty = int(state.CPUDuty)
		} else if name == "GPU" && state.GPUDutyInited {
			currentDuty = int(state.GPUDuty)
		}
		modeText = fmt.Sprintf("自定义 %d%%", currentDuty)
	}

	return widgets.Card(c, "", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, icon, tint, 34)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, name+" 散热风扇").FontSize(15).Bold().TextColor(t.Text)
				ui.Text(c, modeText).FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
			widgets.Pill(c, model.FormatCelsius(tempC), tint)
		})

		ui.Spacer(c)

		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).Center().Children(func() {
				widgets.Fan(c, rpm, 88, tint)
			})
			ui.Column(c).Gap(4).Grow(1).AlignItems(ui.Center).Children(func() {
				ui.Text(c, model.FormatRPM(rpm)).FontSize(24).Bold().TextColor(t.Text).NoWrap()
				ui.Text(c, "实时转速").FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
		})

		ui.Spacer(c)
		ui.Divider(c)

		ui.Column(c).Gap(10).Children(func() {
			ui.Row(c).Gap(12).Children(func() {
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, model.FormatCelsius(tempC)).FontSize(15).Bold().TextColor(t.Text).NoWrap()
					ui.Text(c, "当前温度").FontSize(11).TextColor(t.TextMuted).NoWrap()
				})
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, auxValue).FontSize(15).Bold().TextColor(t.Text).NoWrap()
					ui.Text(c, auxLabel).FontSize(11).TextColor(t.TextMuted).NoWrap()
				})
			})

			if mode == domain.FanCustom {
				var dutyPtr *float64
				if name == "CPU" {
					dutyPtr = &state.CPUDuty
				} else {
					dutyPtr = &state.GPUDuty
				}

				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "占空比").FontSize(12).TextColor(t.TextMuted)
					slider := ui.Slider(c, dutyPtr, 0, 100).Grow(1).MinWidth(140)
					if slider.Changed() {
						cfg := domain.FanConfig{
							CPUMode:    overview.FanState.CPUMode,
							GPUMode:    overview.FanState.GPUMode,
							CPUPercent: int(state.CPUDuty),
							GPUPercent: int(state.GPUDuty),
							CoolBoost:  overview.FanState.CoolBoost,
						}
						ctrl.SetFanConfigDebounced(cfg, 180*time.Millisecond, nil)
					}
					ui.Text(c, fmt.Sprintf("%d%%", int(*dutyPtr))).FontSize(13).Bold().TextColor(tint).Width(42)
				})
			} else {
				fraction := float64(rpm) / 5000.0
				if fraction > 1 {
					fraction = 1
				}
				ui.Column(c).Gap(4).Children(func() {
					ui.Meter(c, fraction, 0, 1, nil).Height(6)
					desc := "智能温控响应中"
					if mode == domain.FanMax {
						desc = "全速暴力散热运转"
					}
					ui.Text(c, desc).FontSize(10).TextColor(t.TextMuted)
				})
			}
		})
	}).Grow(1)
}

func fanStrategyCard(c *ui.Context) {
	t := c.Theme()

	item := func(title, desc string) {
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().TextColor(t.Text)
			ui.Text(c, desc).FontSize(11).TextColor(t.TextMuted)
		})
	}

	widgets.Card(c, "散热模式与策略指引", func() {
		ui.Row(c).Gap(20).Children(func() {
			item("自动模式 (Auto)", "平衡风道与静音，日常办公与影音娱乐推荐，根据芯片温度平缓动态变速。")
			item("全速模式 (Max)", "风扇直接拉满至最高转速，大型 3A 游戏与重度渲染必备，提供极限瞬态压温。")
			item("CoolBoost 强冷", "激活宏碁特调的高速风道增压策略，突破标准转速上限，压制机身热堆积。")
		})
	})
}
