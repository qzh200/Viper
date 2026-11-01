package pages

import (
	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/model"
	"viper/internal/gui/widgets"
)

type State struct {
	PowerIndex    int
	GPUModeIndex  int
	FanModeIndex  int
	CPUDuty       float64
	GPUDuty       float64
	CPUDutyInited bool
	GPUDutyInited bool

	LightingTab              int
	LightingBrightness       float64
	LightingBrightnessInited bool
	LightingSpeed            float64
	LightingSpeedInited      bool
	LightingDirection        int
	LightingDirectionInited  bool
	SelectedEffect           domain.LightingEffect
	SelectedEffectInited     bool
	ColorHue                 float64
	ColorHueInited           bool
	ColorSat                 float64
	ColorSatInited           bool
	SelectedZone             int
	ZoneColors               [4]domain.RGBColor
	ZoneColorsInited         bool
	PerZone                  bool
	PerZoneInited            bool
}

func NewState() *State {
	return &State{
		PowerIndex:         -1,
		GPUModeIndex:       -1,
		FanModeIndex:       -1,
		CPUDuty:            50,
		GPUDuty:            50,
		LightingTab:        0,
		LightingBrightness: 4,
		LightingSpeed:      5,
		LightingDirection:  0,
		SelectedEffect:     domain.EffectStatic,
		ColorHue:           200,
		ColorSat:           1.0,
		SelectedZone:       0,
		PerZone:            false,
		ZoneColors: [4]domain.RGBColor{
			{R: 0, G: 174, B: 243},
			{R: 0, G: 230, B: 118},
			{R: 255, G: 145, B: 0},
			{R: 213, G: 0, B: 249},
		},
	}
}

func Home(c *ui.Context, ctrl *controller.Controller, state *State) {
	overview := ctrl.Snapshot()
	modes := overview.PowerModes
	if len(modes) == 0 {
		modes = domain.AvailablePowerModes()
	}

	ui.Column(c).Fill().Gap(16).Children(func() {
		hero(c, overview, ctrl, state, modes)

		if overview.Error != "" {
			widgets.ErrorNote(c, overview.Error)
		}

		if len(overview.Metrics) == 0 {
			widgets.Card(c, "还没有数据", func() {
				widgets.Note(c, "确认原厂服务 AcerAgentService 正在运行，然后等下一次刷新。")
			})
			return
		}

		ui.Row(c).Gap(16).Grow(1).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Gap(16).Basis(660).Grow(1).MinWidth(0).FillHeight().Children(func() {
				ui.Row(c).Gap(16).Children(func() {
					coreCard(c, overview, "CPU", widgets.IconCPU, widgets.ColorCPU,
						overview.Metrics[0], overview.Metrics[3])
					coreCard(c, overview, "GPU", widgets.IconGPU, widgets.ColorGPU,
						overview.Metrics[1], overview.Metrics[4])
				})
				widgets.Card(c, "温度走势", func() {
					widgets.Trend(c, overview.History)
				}).Grow(1)
			})

			ui.Column(c).Gap(16).Width(280).Children(func() {
				memoryCard(c, overview)
				boardCard(c, overview)
				networkCard(c, overview)
			})
		})
	})
}

func hero(c *ui.Context, overview model.Overview, ctrl *controller.Controller,
	state *State, modes []domain.PowerMode) {
	t := c.Theme()

	ui.Column(c).Padding(14, 16).Gap(10).Radius(14).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconDisplay, t.Accent, 40)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "Acer 游戏本控制中心").FontSize(18).Bold().TextColor(t.Text)
				ui.Text(c, subtitle(overview)).FontSize(12).TextColor(t.TextMuted).NoWrap()
			})
			statusPill(c, overview)
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "性能档位").FontSize(11).TextColor(t.TextMuted)
				powerPicker(c, ctrl, state, modes, overview)
			})
		})
	})
}

func statusPill(c *ui.Context, overview model.Overview) {
	t := c.Theme()
	if overview.Connected && overview.Error == "" {
		widgets.Pill(c, "已连接", t.Success)
		return
	}
	if overview.Connected {
		widgets.Pill(c, "读数有误", t.Warning)
		return
	}
	widgets.Pill(c, "未连接", t.Danger)
}

func subtitle(overview model.Overview) string {
	if !overview.Connected {
		return "没有连上原厂服务"
	}
	route := overview.Route
	if route == "" {
		route = overview.Instance
	}
	if overview.Updated.IsZero() {
		return route
	}
	return route + " · " + overview.Updated.Format("15:04:05")
}

func coreCard(c *ui.Context, overview model.Overview, name string, icon widgets.Icon,
	tint ui.Color, temperature, fan model.Metric) ui.Element {
	t := c.Theme()
	monitor := overview.Monitor

	stat := func(label, value string) {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, value).FontSize(15).Bold().TextColor(t.Text).NoWrap()
			ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).NoWrap()
		})
	}

	return widgets.Card(c, "", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, icon, tint, 34)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, name).FontSize(15).Bold().TextColor(t.Text)
				ui.Text(c, temperature.Detail).FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
			widgets.Pill(c, temperature.Level.Label(), gaugeTint(c, temperature.Level, tint))
		})

		ui.Spacer(c)

		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Grow(1).Center().Children(func() {
				widgets.Gauge(c, widgets.RadialGauge{
					Value:     temperature.Number,
					Low:       30,
					High:      100,
					ValueText: temperature.Value,
					Label:     "温度",
					Tint:      gaugeTint(c, temperature.Level, tint),
					Size:      102,
				})
			})
			ui.Column(c).Gap(4).Grow(1).AlignItems(ui.Center).Children(func() {
				widgets.Fan(c, int(fan.Number), 64, tint)
				ui.Text(c, fan.Value).FontSize(14).Bold().TextColor(t.Text).NoWrap()
				ui.Text(c, "风扇转速").FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
		})

		ui.Spacer(c)
		ui.Divider(c)

		ui.Row(c).Gap(8).Children(func() {
			if name == "CPU" {
				stat("占用", model.FormatPercent(monitor.CPUUsagePercent))
				stat("频率", model.FormatMHz(monitor.CPUFrequencyMHz))
			} else {
				stat("占用", model.FormatPercent(float64(monitor.GPUUsagePercent)))
				stat("频率", model.FormatMHz(monitor.GPUFrequencyMHz))
			}
		})
	}).Grow(1)
}

func memoryCard(c *ui.Context, overview model.Overview) {
	t := c.Theme()
	memory := overview.Metric("内存占用")
	monitor := overview.Monitor

	stat := func(label, value string) {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, value).FontSize(14).Bold().TextColor(t.Text).NoWrap()
			ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).NoWrap()
		})
	}

	widgets.Card(c, "", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconMemory, widgets.ColorMemory, 34)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "内存").FontSize(15).Bold().TextColor(t.Text)
				ui.Text(c, model.FormatMB(monitor.RAMTotalMB)).FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
			widgets.Pill(c, memory.Level.Label(), gaugeTint(c, memory.Level, widgets.ColorMemory))
		})

		ui.Column(c).Gap(6).Children(func() {
			ui.Row(c).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, memory.Value).FontSize(26).Bold().TextColor(gaugeTint(c, memory.Level, widgets.ColorMemory))
				ui.Spacer(c)
				ui.Text(c, model.FormatMB(monitor.RAMTotalMB*monitor.RAMUsagePercent/100)+" / "+model.FormatMB(monitor.RAMTotalMB)).
					FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
			ui.Meter(c, memory.Percent, 0, 1, nil).Height(8)
		})

		ui.Divider(c)

		ui.Row(c).Gap(6).Children(func() {
			stat("已用", model.FormatMB(monitor.RAMTotalMB*monitor.RAMUsagePercent/100))
			stat("总量", model.FormatMB(monitor.RAMTotalMB))
			stat("频率", model.FormatMemoryClock(monitor.RAMFrequencyMHz))
		})
	})
}

func boardCard(c *ui.Context, overview model.Overview) {
	t := c.Theme()
	board := overview.Metric("主板温度")

	widgets.Card(c, "", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconThermometer, widgets.ColorBoard, 34)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "主板温度").FontSize(15).Bold().TextColor(t.Text)
				ui.Text(c, board.Detail).FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
			widgets.Pill(c, board.Level.Label(), gaugeTint(c, board.Level, widgets.ColorBoard))
		})

		ui.Spacer(c)

		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			widgets.Gauge(c, widgets.RadialGauge{
				Value:     board.Number,
				Low:       30,
				High:      100,
				ValueText: board.Value,
				Label:     "主板",
				Tint:      gaugeTint(c, board.Level, widgets.ColorBoard),
				Size:      104,
			})
			ui.Column(c).Gap(6).Grow(1).Children(func() {
				ui.Text(c, "温度档位").FontSize(13).Bold().TextColor(t.Text)
				ui.Text(c, "低于 70° 正常").FontSize(11).TextColor(t.TextMuted).NoWrap()
				ui.Text(c, "70° 到 85° 偏高").FontSize(11).TextColor(t.TextMuted).NoWrap()
				ui.Text(c, "高于 85° 过高").FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
		})
	}).Grow(1)
}

func networkCard(c *ui.Context, overview model.Overview) {
	t := c.Theme()
	monitor := overview.Monitor

	cell := func(value, label string) {
		ui.Column(c).Gap(3).Children(func() {
			ui.Text(c, value).FontSize(15).Bold().TextColor(t.Text).NoWrap()
			ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).NoWrap()
		})
	}

	widgets.Card(c, "", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconActivity, widgets.ColorNet, 34)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "网络").FontSize(15).Bold().TextColor(t.Text)
				ui.Text(c, "实时收发速率").FontSize(11).TextColor(t.TextMuted).NoWrap()
			})
		})
		ui.Spacer(c)
		ui.Grid(c).Columns(2).GapY(14).GapX(8).Children(func() {
			cell("↓ "+model.FormatKbps(monitor.WIFIDownloadKbps), "无线下载")
			cell("↑ "+model.FormatKbps(monitor.WIFIUploadKbps), "无线上传")
			cell("↓ "+model.FormatKbps(monitor.EthernetDownloadKbps), "有线下载")
			cell("↑ "+model.FormatKbps(monitor.EthernetUploadKbps), "有线上传")
		})
	}).Grow(1)
}

func powerPicker(c *ui.Context, ctrl *controller.Controller, state *State,
	modes []domain.PowerMode, overview model.Overview) {
	selected := model.PowerModeIndex(modes, overview.PowerMode)
	if state.PowerIndex >= 0 && state.PowerIndex < len(modes) {
		if modes[state.PowerIndex] == overview.PowerMode {
			state.PowerIndex = -1
		} else {
			selected = state.PowerIndex
		}
	}

	index := selected
	picker := ui.Segmented(c, &index, model.PowerModeLabels(modes)...).Label("性能档位")
	if !picker.Changed() {
		return
	}
	if index < 0 || index >= len(modes) {
		return
	}
	state.PowerIndex = index
	ctrl.SetPowerMode(modes[index], func(err error) {
		if err != nil {
			state.PowerIndex = -1
		}
	})
}

func gaugeTint(c *ui.Context, level model.Level, base ui.Color) ui.Color {
	switch level {
	case model.LevelWarn:
		return c.Theme().Warning
	case model.LevelDanger:
		return c.Theme().Danger
	default:
		return base
	}
}
