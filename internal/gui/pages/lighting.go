// Package pages 实现 Viper 界面的各个页面。
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

// Lighting 画键盘 RGB 灯效与背光控制页面。
func Lighting(c *ui.Context, ctrl *controller.Controller, state *State) {
	overview := ctrl.Snapshot()
	lighting := overview.Lighting

	// 初始化状态同步
	if !state.LightingBrightnessInited {
		if lighting.Brightness > 0 {
			state.LightingBrightness = float64(lighting.Brightness)
		} else {
			state.LightingBrightness = 100
		}
		state.LightingBrightnessInited = true
	}
	if !state.PerZoneInited {
		state.PerZone = lighting.PerZone
		state.PerZoneInited = true
	}
	if !state.ZoneColorsInited {
		hasColor := false
		for _, z := range lighting.Zones {
			if z.R > 0 || z.G > 0 || z.B > 0 {
				hasColor = true
				break
			}
		}
		if hasColor {
			state.ZoneColors = lighting.Zones
		}
		state.ZoneColorsInited = true
	}

	ui.Column(c).Fill().Gap(16).Children(func() {
		lightingHero(c, overview, ctrl, state)

		if overview.Error != "" {
			widgets.ErrorNote(c, overview.Error)
		}

		if state.LightingTab == 0 {
			lightingGlobalPanel(c, overview, ctrl, state)
		} else {
			lightingZonesPanel(c, overview, ctrl, state)
		}

		lightingStrategyCard(c)
	})
}

// lightingHero 画灯效页顶部的导航卡片（Tab 选择、30秒超时开关、恢复默认）。
func lightingHero(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()

	ui.Column(c).Padding(14, 16).Gap(10).Radius(14).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.10)).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			widgets.IconBadge(c, widgets.IconBolt, t.Accent, 40)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "键盘 RGB 灯效").FontSize(18).Bold().TextColor(t.Text)
				ui.Text(c, "四分区独立光谱 · 动态氛围流动与全域客制调光").FontSize(12).TextColor(t.TextMuted).NoWrap()
			})

			// 切换：全局设置 vs 分区设置
			tabIdx := state.LightingTab
			picker := ui.Segmented(c, &tabIdx, "全局设置", "分区设置").Label("灯效模式切换")
			if picker.Changed() {
				state.LightingTab = tabIdx
			}

			// 30 秒空闲自动休眠
			timeoutOn := overview.BacklightTimeout
			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 12).Radius(10).
				Background(t.Background).Border(1, t.Border).Children(func() {
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "30s 自动休眠").FontSize(12).Bold().TextColor(t.Text)
					ui.Text(c, "键盘空闲省电").FontSize(10).TextColor(t.TextMuted)
				})
				sw := ui.Switch(c, &timeoutOn)
				if sw.Changed() {
					ctrl.SetBacklightTimeout(timeoutOn, nil)
				}
			})

			// 恢复出厂默认灯效按钮
			resetBtn := ui.Button(c, "恢复默认").
				Padding(8, 14).Radius(8).
				Background(t.Background).Border(1, t.Border)
			if resetBtn.Clicked() {
				ctrl.ResetLighting(nil)
				// 重置本地 state 状态
				state.LightingBrightness = 100
				state.PerZone = true
				for i := range state.ZoneColors {
					state.ZoneColors[i] = domain.RGBColor{R: 255, G: 255, B: 255}
				}
			}
		})
	})
}

// lightingGlobalPanel 画全局动态模式与全局参数设置。
func lightingGlobalPanel(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()
	lighting := overview.Lighting

	effects := model.LightingEffects()
	currentEffect := lighting.Effect

	// 1. 动态灯效模式选择网格（分为两行，避免横向挤压截断）
	widgets.Card(c, "全局运转灯效模式", func() {
		renderBtn := func(item model.LightingEffectItem) {
			active := item.Effect == currentEffect

			bg := t.Background
			borderCol := t.Border
			textCol := t.Text
			if active {
				bg = t.Accent.Alpha(0.14)
				borderCol = t.Accent
				textCol = t.Accent
			}

			btn := ui.Column(c).Grow(1).MinWidth(0).Padding(10, 8).Gap(4).Radius(10).
				Background(bg).Border(1, borderCol).
				AlignItems(ui.Center).Cursor(ui.CursorPointer).Children(func() {
				ui.Text(c, item.Name).FontSize(13).Bold().TextColor(textCol).NoWrap()
				ui.Text(c, item.Desc).FontSize(10).TextColor(t.TextMuted).NoWrap()
			})
			if btn.Clicked() {
				cfg := buildLightingConfig(state, overview)
				cfg.Effect = item.Effect
				ctrl.SetLighting(cfg, nil)
			}
		}

		ui.Column(c).Gap(10).Children(func() {
			// 第一行：4 个模式
			ui.Row(c).Gap(10).Children(func() {
				for i := 0; i < 4 && i < len(effects); i++ {
					renderBtn(effects[i])
				}
			})
			// 第二行：3 个模式
			ui.Row(c).Gap(10).Children(func() {
				for i := 4; i < len(effects); i++ {
					renderBtn(effects[i])
				}
			})
		})
	})

	// 2. 亮度控制与动态参数
	ui.Row(c).Gap(16).Children(func() {
		// 左卡片：背光亮度调节
		widgets.Card(c, "背光亮度", func() {
			ui.Column(c).Gap(14).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, fmt.Sprintf("%d%%", int(state.LightingBrightness))).
						FontSize(28).Bold().TextColor(t.Accent)
					ui.Spacer(c)
					ui.Text(c, "全键盘发光强弱").FontSize(11).TextColor(t.TextMuted)
				})

				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "亮度").FontSize(12).TextColor(t.TextMuted)
					slider := ui.Slider(c, &state.LightingBrightness, 0, 100).Grow(1).MinWidth(140)
					if slider.Changed() {
						cfg := buildLightingConfig(state, overview)
						cfg.Brightness = uint64(state.LightingBrightness)
						ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
					}
					ui.Text(c, fmt.Sprintf("%d%%", int(state.LightingBrightness))).
						FontSize(13).Bold().TextColor(t.Accent).Width(42)
				})
			})
		}).Grow(1)

		// 右卡片：动画速度与流动方向
		widgets.Card(c, "动态参数", func() {
			ui.Column(c).Gap(14).Children(func() {
				// 动画速率
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "动画速度").FontSize(12).TextColor(t.TextMuted).Width(60)
					speedIdx := state.LightingSpeed
					picker := ui.Segmented(c, &speedIdx, "慢速", "正常", "快速").Label("动画速度")
					if picker.Changed() {
						state.LightingSpeed = speedIdx
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLighting(cfg, nil)
					}
				})

				// 流动方向
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "流动方向").FontSize(12).TextColor(t.TextMuted).Width(60)
					dirIdx := state.LightingDirection
					picker := ui.Segmented(c, &dirIdx, "向左 ◄", "向右 ►").Label("流动方向")
					if picker.Changed() {
						state.LightingDirection = dirIdx
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLighting(cfg, nil)
					}
				})
			})
		}).Grow(1)
	})
}

// lightingZonesPanel 画四分区独立色彩设置面板。
func lightingZonesPanel(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()

	// 1. 4 分区键盘拟物预览卡片行
	widgets.Card(c, "四分区键盘色彩预览", func() {
		ui.Column(c).Gap(12).Children(func() {
			// 分区独立调色开关
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "四分区独立调色").FontSize(13).Bold().TextColor(t.Text)
				ui.Text(c, "（关闭时 4 个分区同步应用相同色彩）").FontSize(11).TextColor(t.TextMuted)
				ui.Spacer(c)
				perZone := state.PerZone
				sw := ui.Switch(c, &perZone)
				if sw.Changed() {
					state.PerZone = perZone
					cfg := buildLightingConfig(state, overview)
					cfg.PerZone = perZone
					ctrl.SetLighting(cfg, nil)
				}
			})

			ui.Divider(c)

			// 并排 4 个分区大卡片
			ui.Row(c).Gap(12).Children(func() {
				for i := 0; i < 4; i++ {
					zoneIdx := i
					selected := state.SelectedZone == zoneIdx
					color := state.ZoneColors[zoneIdx]
					uiColor := ui.RGB(color.R, color.G, color.B)

					bg := t.Background
					borderCol := t.Border
					if selected {
						bg = t.Accent.Alpha(0.10)
						borderCol = t.Accent
					}

					card := ui.Column(c).Grow(1).Padding(12, 10).Gap(8).Radius(12).
						Background(bg).Border(1, borderCol).
						AlignItems(ui.Center).Cursor(ui.CursorPointer).Children(func() {
						ui.Text(c, model.ZoneName(zoneIdx)).FontSize(12).Bold().TextColor(t.Text).NoWrap()

						// 色块发光效果
						ui.Box(c).Size(46, 26).Radius(6).
							Background(uiColor).
							Shadow(0, 0, 10, 0, uiColor.Alpha(0.65))

						hex := fmt.Sprintf("#%02X%02X%02X", color.R, color.G, color.B)
						ui.Text(c, hex).FontSize(11).Bold().TextColor(t.TextMuted).NoWrap()

						if selected {
							ui.Text(c, "● 当前选中").FontSize(10).Bold().TextColor(t.Accent).NoWrap()
						}
					})
					if card.Clicked() {
						state.SelectedZone = zoneIdx
					}
				}
			})
		})
	})

	// 2. 当前选中分区的调色台（预设色盘 + RGB 微调）
	widgets.Card(c, fmt.Sprintf("色彩调控 · %s", model.ZoneName(state.SelectedZone)), func() {
		ui.Column(c).Gap(14).Children(func() {
			// 快捷预设色盘
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				ui.Text(c, "精选色盘:").FontSize(12).TextColor(t.TextMuted)
				for _, p := range model.LightingPresetColors() {
					preset := p
					pColor := ui.RGB(preset.Color.R, preset.Color.G, preset.Color.B)

					btn := ui.Row(c).Padding(5, 8).Gap(6).Radius(8).
						Background(t.Background).Border(1, t.Border).
						AlignItems(ui.Center).Cursor(ui.CursorPointer).Children(func() {
						ui.Box(c).Size(14, 14).Radius(3).Background(pColor)
						ui.Text(c, preset.Name).FontSize(11).TextColor(t.Text).NoWrap()
					})
					if btn.Clicked() {
						applyColor(state, preset.Color)
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLightingDebounced(cfg, 120*time.Millisecond, nil)
					}
				}
			})

			ui.Divider(c)

			// RGB 三分量微调滑块
			curColor := state.ZoneColors[state.SelectedZone]
			rVal := float64(curColor.R)
			gVal := float64(curColor.G)
			bVal := float64(curColor.B)

			changed := false
			rgbRow := func(label string, val *float64, tint ui.Color) {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).FontSize(12).Bold().TextColor(tint).Width(24)
					slider := ui.Slider(c, val, 0, 255).Grow(1).MinWidth(140)
					if slider.Changed() {
						changed = true
					}
					ui.Text(c, fmt.Sprintf("%d", int(*val))).FontSize(12).TextColor(t.Text).Width(36)
				})
			}

			rgbRow("R", &rVal, ui.RGB(255, 60, 60))
			rgbRow("G", &gVal, ui.RGB(60, 220, 60))
			rgbRow("B", &bVal, ui.RGB(60, 140, 255))

			if changed {
				newColor := domain.RGBColor{
					R: uint8(rVal),
					G: uint8(gVal),
					B: uint8(bVal),
				}
				applyColor(state, newColor)
				cfg := buildLightingConfig(state, overview)
				ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
			}
		})
	})
}

// applyColor 把颜色应用到当前选区或全部选区。
func applyColor(state *State, color domain.RGBColor) {
	if state.PerZone {
		state.ZoneColors[state.SelectedZone] = color
	} else {
		for i := range state.ZoneColors {
			state.ZoneColors[i] = color
		}
	}
}

// buildLightingConfig 根据页面状态构建待写入的 RGBConfig。
func buildLightingConfig(state *State, overview model.Overview) domain.RGBConfig {
	speed := uint64(5)
	switch state.LightingSpeed {
	case 0:
		speed = 1
	case 2:
		speed = 9
	}

	return domain.RGBConfig{
		Effect:     overview.Lighting.Effect,
		Speed:      speed,
		Direction:  uint64(state.LightingDirection),
		Brightness: uint64(state.LightingBrightness),
		Zones:      state.ZoneColors,
		PerZone:    state.PerZone,
	}
}

// lightingStrategyCard 画底部的灯效策略说明卡片。
func lightingStrategyCard(c *ui.Context) {
	t := c.Theme()

	item := func(title, desc string) {
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().TextColor(t.Text)
			ui.Text(c, desc).FontSize(11).TextColor(t.TextMuted)
		})
	}

	widgets.Card(c, "键盘背光与氛围策略指引", func() {
		ui.Row(c).Gap(20).Children(func() {
			item("静态与分区客制", "适合电竞按键高亮区分，WASD 区域使用高对比色，办公时单色常亮护眼专注。")
			item("动态流动光效", "全波浪与霓虹变色流动感极强，提供沉浸电竞氛围，建议在接通电源时体验。")
			item("30 秒自动休眠", "检测到键盘空闲 30 秒后自动熄灭背光，任意按键即刻唤醒，大幅提升电池续航。")
		})
	})
}
