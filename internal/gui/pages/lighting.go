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
			if lighting.Brightness <= 1 {
				state.LightingBrightness = 0
			} else if lighting.Brightness >= domain.BrightnessMax {
				state.LightingBrightness = 4
			} else {
				state.LightingBrightness = float64(lighting.Brightness - 1)
			}
		} else {
			state.LightingBrightness = 4
		}
		state.LightingBrightnessInited = true
	}
	if !state.LightingSpeedInited {
		if lighting.Speed >= domain.SpeedMin && lighting.Speed <= domain.SpeedMax {
			state.LightingSpeed = float64(lighting.Speed)
		} else {
			state.LightingSpeed = 5
		}
		state.LightingSpeedInited = true
	}
	if !state.SelectedEffectInited {
		state.SelectedEffect = lighting.Effect
		state.SelectedEffectInited = true
	}
	if !state.LightingDirectionInited {
		state.LightingDirection = int(lighting.Direction)
		state.LightingDirectionInited = true
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
	if !state.ColorHueInited || !state.ColorSatInited {
		h, s, _ := model.RGBToHSV(state.ZoneColors[state.SelectedZone])
		state.ColorHue = h
		if s > 0 {
			state.ColorSat = s
		} else {
			state.ColorSat = 1.0
		}
		state.ColorHueInited = true
		state.ColorSatInited = true
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
				// 重置本地 state 状态（亮度恢复到最大 4 档即硬件 5 档）
				state.LightingBrightness = 4
				state.LightingSpeed = 5
				state.LightingDirection = 0
				state.SelectedEffect = domain.EffectStatic
				state.PerZone = false
				for i := range state.ZoneColors {
					state.ZoneColors[i] = domain.RGBColor{R: 255, G: 255, B: 255}
				}
				state.ColorHue = 0
			}
		})
	})
}

// lightingGlobalPanel 画全局动态模式与全局参数设置。
func lightingGlobalPanel(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()

	effects := model.LightingEffects()
	currentEffect := state.SelectedEffect

	// 1. 全局灯效模式选择网格（4 列 x 2 行，视野开阔紧凑）
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
				state.SelectedEffect = item.Effect
				state.PerZone = false
				cfg := buildLightingConfig(state, overview)
				cfg.Effect = item.Effect
				cfg.PerZone = false
				ctrl.SetLighting(cfg, nil)
			}
		}

		ui.Column(c).Gap(10).Children(func() {
			for row := 0; row < 2; row++ {
				ui.Row(c).Gap(10).Children(func() {
					for col := 0; col < 4; col++ {
						index := row*4 + col
						if index < len(effects) {
							renderBtn(effects[index])
						} else {
							ui.Box(c).Grow(1).MinWidth(0)
						}
					}
				})
			}
		})
	})

	// 2. 根据当前模式特性自适应排版调控区
	canColor := model.LightingEffectCanColor(currentEffect)
	hasDirection := currentEffect.HasDirection()

	if currentEffect == domain.EffectStatic {
		// 静态常亮：支持调亮度与选色板（无速度、无方向），左右对称并排
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
		renderColorPaletteCard(c, overview, ctrl, state, t, "静态全局色彩调控")
	} else if !canColor {
		// 霓虹与波形：不支持调色；波形可调方向；两者均可调速度和亮度
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
	} else {
		// 其余动态模式（呼吸、切换、缩放、流星雨、闪烁）：
		// 第一行：背光亮度 + 动态参数（速度与方向）
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
		// 第二行：全局选色板卡片
		renderColorPaletteCard(c, overview, ctrl, state, t, "灯效色彩调控")
	}
}

// renderBrightnessCard 画背光亮度卡片（0~4 档，0 档对应熄灭，4 档对应最高亮度）。
func renderBrightnessCard(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State, t *ui.Theme) {
	widgets.Card(c, "背光亮度", func() {
		ui.Column(c).Gap(14).Children(func() {
			level := int(state.LightingBrightness)
			titleText := fmt.Sprintf("%d 档", level)
			descText := "全键盘发光强弱"
			if level == 0 {
				descText = "背光已熄灭 (按键灯关闭)"
			} else if level == 4 {
				descText = "全键盘最高发光亮度"
			}

			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, titleText).FontSize(28).Bold().TextColor(t.Accent)
				if level == 0 {
					ui.Box(c).Padding(2, 6).Radius(4).Background(t.Border).Children(func() {
						ui.Text(c, "已熄灭").FontSize(11).Bold().TextColor(t.TextMuted)
					})
				}
				ui.Spacer(c)
				ui.Text(c, descText).FontSize(11).TextColor(t.TextMuted)
			})

			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "亮度").FontSize(12).TextColor(t.TextMuted)
				slider := ui.Slider(c, &state.LightingBrightness, 0, 4).Grow(1).MinWidth(140)
				if slider.Changed() {
					state.LightingBrightness = float64(int(state.LightingBrightness + 0.5))
					cfg := buildLightingConfig(state, overview)
					ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
				}
				ui.Text(c, fmt.Sprintf("%d 档", int(state.LightingBrightness))).
					FontSize(13).Bold().TextColor(t.Accent).Width(42)
			})
		})
	}).Grow(1)
}

// renderDynamicParamsCard 画动画速度与流动方向动态参数卡片。
func renderDynamicParamsCard(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State, t *ui.Theme, effect domain.LightingEffect, hasDirection bool) {
	canSpeed := model.LightingEffectCanSpeed(effect)

	widgets.Card(c, "动态参数", func() {
		ui.Column(c).Gap(12).Children(func() {
			if canSpeed {
				// 1. 动画速度调节（1~9），使用防抖并提供档位直选
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "动画速度").FontSize(12).TextColor(t.TextMuted).Width(60)
					slider := ui.Slider(c, &state.LightingSpeed,
						float64(domain.SpeedMin), float64(domain.SpeedMax)).Grow(1).MinWidth(120)
					if slider.Changed() {
						state.LightingSpeed = float64(int(state.LightingSpeed + 0.5))
						cfg := buildLightingConfig(state, overview)
						cfg.Speed = uint64(state.LightingSpeed)
						ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
					}
					ui.Text(c, fmt.Sprintf("%d 档", int(state.LightingSpeed))).
						FontSize(13).Bold().TextColor(t.Accent).Width(42)
				})

				// 快捷档位切换按钮
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "档位直选:").FontSize(11).TextColor(t.TextMuted).Width(60)
					for _, item := range []struct {
						name string
						val  int
					}{
						{"慢速 (1档)", 1},
						{"标准 (5档)", 5},
						{"极速 (9档)", 9},
					} {
						active := int(state.LightingSpeed) == item.val
						bg := t.Background
						bColor := t.Border
						txtColor := t.Text
						if active {
							bg = t.Accent.Alpha(0.14)
							bColor = t.Accent
							txtColor = t.Accent
						}

						btn := ui.Row(c).Padding(3, 8).Radius(6).
							Background(bg).Border(1, bColor).
							Cursor(ui.CursorPointer).Children(func() {
							ui.Text(c, item.name).FontSize(11).Bold().TextColor(txtColor)
						})
						if btn.Clicked() {
							state.LightingSpeed = float64(item.val)
							cfg := buildLightingConfig(state, overview)
							cfg.Speed = uint64(item.val)
							ctrl.SetLighting(cfg, nil)
						}
					}
				})
			} else {
				// 静态模式提示
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "● 当前为静态单色常亮，无需调节动态速度").
						FontSize(11).TextColor(t.TextMuted)
				})
			}

			// 2. 流动方向（仅波形与切换模式支持）
			if hasDirection {
				ui.Divider(c)
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "流动方向").FontSize(12).TextColor(t.TextMuted).Width(60)
					dirIdx := state.LightingDirection
					picker := ui.Segmented(c, &dirIdx, "向左 ◄", "向右 ►").Label("流动方向")
					if picker.Changed() {
						state.LightingDirection = dirIdx
						cfg := buildLightingConfig(state, overview)
						cfg.Direction = uint64(dirIdx)
						ctrl.SetLighting(cfg, nil)
					}
				})
			}

			// 3. 全光谱流动说明（霓虹与波形）
			if !model.LightingEffectCanColor(effect) {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "●").FontSize(8).TextColor(t.Accent)
					ui.Text(c, "全光谱流光模式，硬件光谱自动循环，无需设置色彩").
						FontSize(11).TextColor(t.TextMuted)
				})
			}
		})
	}).Grow(1)
}

// renderColorPaletteCard 画 PredatorSense 原厂风格的专业色轮取色板。
func renderColorPaletteCard(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State, t *ui.Theme, title string) {
	widgets.Card(c, title, func() {
		curColor := state.ZoneColors[state.SelectedZone]
		if state.LightingTab == 0 {
			curColor = state.ZoneColors[0]
		}
		uiColor := ui.RGB(curColor.R, curColor.G, curColor.B)
		hex := fmt.Sprintf("#%02X%02X%02X", curColor.R, curColor.G, curColor.B)

		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			// 左半部：极坐标平滑色轮（支持鼠标实时拖动与点击取色）+ 色相微调滑块
			ui.Column(c).Gap(10).AlignItems(ui.Center).Children(func() {
				// 1. 360 度原生圆盘平滑色轮，鼠标点击或直接按住拖动取色
				satVal := state.ColorSat
				if satVal <= 0 {
					satVal = 1.0
				}
				widgets.ColorWheel(c, widgets.ColorWheelSpec{
					Size:       146,
					Hue:        state.ColorHue,
					Saturation: satVal,
					Color:      curColor,
					OnChange: func(hue, sat float64) {
						state.ColorHue = hue
						state.ColorSat = sat
						newCol := model.HSVToRGB(hue, sat, 1.0)
						applyColor(state, newCol)
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
					},
				})

				// 2. 色轮下方：色相滑块
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "色相").FontSize(11).Bold().TextColor(t.TextMuted)
					slider := ui.Slider(c, &state.ColorHue, 0, 360).Width(100)
					if slider.Changed() {
						s := state.ColorSat
						if s <= 0 {
							s = 1.0
						}
						newCol := model.HSVToRGB(state.ColorHue, s, 1.0)
						applyColor(state, newCol)
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
					}
					ui.Text(c, fmt.Sprintf("%d°", int(state.ColorHue))).FontSize(11).TextColor(t.Accent).Width(32)
				})
			})

			// 右半部：原厂预设圆形色块排 + 垂直大色彩胶囊 + R/G/B 精准微调框
			ui.Column(c).Grow(1).Gap(14).Children(func() {
				// 1. 圆形快捷预设色盘（对应原厂面板：熄灭、青、绿、红、橙、紫、白）
				ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
					// 熄灭按钮（带斜杠 ⃠）
					offBtn := ui.Box(c).Size(22, 22).Radius(11).
						Background(ui.RGB(20, 20, 20)).Border(1, t.Border).
						Cursor(ui.CursorPointer).Label("熄灭").Draw(func(p *ui.Painter, r ui.Rect) {
						path := &ui.Path{}
						path.MoveTo(r.X+5, r.Y+5)
						path.LineTo(r.X+r.W-5, r.Y+r.H-5)
						p.StrokePath(path, 1.8, ui.RGB(255, 30, 30))
					})
					if offBtn.Clicked() {
						offColor := domain.RGBColor{R: 0, G: 0, B: 0}
						applyColor(state, offColor)
						state.ColorSat = 0
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLighting(cfg, nil)
					}

					// 快捷原厂常用纯色圆圈（带无障碍语义标签，与官方一致）
					for _, item := range []struct {
						name  string
						color domain.RGBColor
					}{
						{"掠夺蓝", domain.RGBColor{R: 0, G: 174, B: 243}},
						{"极光青", domain.RGBColor{R: 0, G: 230, B: 118}},
						{"烈焰红", domain.RGBColor{R: 255, G: 23, B: 68}},
						{"炽热橙", domain.RGBColor{R: 255, G: 145, B: 0}},
						{"赛博紫", domain.RGBColor{R: 213, G: 0, B: 249}},
						{"纯白", domain.RGBColor{R: 255, G: 255, B: 255}},
					} {
						col := item.color
						pColor := ui.RGB(col.R, col.G, col.B)
						isSelected := curColor.R == col.R && curColor.G == col.G && curColor.B == col.B

						borderCol := t.Border
						borderWidth := float32(1)
						if isSelected {
							borderCol = t.Accent
							borderWidth = 2
						}

						dot := ui.Box(c).Size(22, 22).Radius(11).
							Background(pColor).Border(borderWidth, borderCol).
							Cursor(ui.CursorPointer).Label(item.name)
						if dot.Clicked() {
							applyColor(state, col)
							h, s, _ := model.RGBToHSV(col)
							state.ColorHue = h
							state.ColorSat = s
							cfg := buildLightingConfig(state, overview)
							ctrl.SetLighting(cfg, nil)
						}
					}

					// 加号按钮（添加色彩收藏，与官方一致）
					ui.Box(c).Size(22, 22).Radius(11).
						Background(t.Border).Border(1, t.Border).
						Cursor(ui.CursorPointer).Label("添加色彩").Draw(func(p *ui.Painter, r ui.Rect) {
						cx, cy := r.X+r.W/2, r.Y+r.H/2
						path := &ui.Path{}
						path.MoveTo(cx-4, cy)
						path.LineTo(cx+4, cy)
						path.MoveTo(cx, cy-4)
						path.LineTo(cx, cy+4)
						p.StrokePath(path, 1.6, t.TextMuted)
					})
				})

				// 2. 下部：垂直大胶囊色块 + RGB 独立数值框 + HEX 标签
				ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
					// 垂直色彩指示胶囊（对应原厂竖长条色块）
					ui.Box(c).Size(24, 82).Radius(12).
						Background(uiColor).
						Shadow(0, 0, 14, 0, uiColor.Alpha(0.70)).
						Border(1, t.Border)

					// 纵向 R、G、B 数值行与微调
					rVal := float64(curColor.R)
					gVal := float64(curColor.G)
					bVal := float64(curColor.B)
					changed := false

					ui.Column(c).Grow(1).Gap(6).Children(func() {
						rgbRow := func(label string, val *float64, tint ui.Color) {
							ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
								ui.Text(c, label).FontSize(13).Bold().TextColor(tint).Width(16)
								slider := ui.Slider(c, val, 0, 255).Grow(1).MinWidth(100)
								if slider.Changed() {
									changed = true
								}
								// 对应原厂界面的数值边框盒子
								ui.Box(c).Padding(2, 8).Radius(4).
									Background(t.Background).Border(1, t.Border).Children(func() {
									ui.Text(c, fmt.Sprintf("%d", int(*val))).
										FontSize(12).Bold().TextColor(t.Text).Width(28)
								})
							})
						}

						rgbRow("R", &rVal, ui.RGB(255, 60, 60))
						rgbRow("G", &gVal, ui.RGB(60, 220, 60))
						rgbRow("B", &bVal, ui.RGB(60, 140, 255))
					})

					if changed {
						newColor := domain.RGBColor{
							R: uint8(rVal),
							G: uint8(gVal),
							B: uint8(bVal),
						}
						applyColor(state, newColor)
						h, s, _ := model.RGBToHSV(newColor)
						state.ColorHue = h
						state.ColorSat = s
						cfg := buildLightingConfig(state, overview)
						ctrl.SetLightingDebounced(cfg, 180*time.Millisecond, nil)
					}

					// HEX 色码标签
					ui.Column(c).AlignItems(ui.End).Children(func() {
						ui.Text(c, "HEX").FontSize(10).TextColor(t.TextMuted)
						ui.Text(c, hex).FontSize(13).Bold().TextColor(t.Accent)
					})
				})
			})
		})
	}).Grow(1)
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
					borderWidth := float32(1)
					if selected {
						bg = t.Accent.Alpha(0.12)
						borderCol = t.Accent
						borderWidth = 2
					}

					card := ui.Column(c).Grow(1).Padding(12, 10).Gap(8).Radius(12).
						Background(bg).Border(borderWidth, borderCol).
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
						h, s, _ := model.RGBToHSV(color)
						state.ColorHue = h
						state.ColorSat = s
					}
				}
			})
		})
	})

	// 2. 当前选中分区的调色台
	renderColorPaletteCard(c, overview, ctrl, state, t,
		fmt.Sprintf("色彩调控 · %s", model.ZoneName(state.SelectedZone)))
}

// applyColor 把颜色应用到当前选区或全部选区。
func applyColor(state *State, color domain.RGBColor) {
	if state.LightingTab == 1 && state.PerZone {
		state.ZoneColors[state.SelectedZone] = color
	} else {
		for i := range state.ZoneColors {
			state.ZoneColors[i] = color
		}
	}
}

// buildLightingConfig 根据页面状态构建待写入的 RGBConfig。
func buildLightingConfig(state *State, overview model.Overview) domain.RGBConfig {
	speed := uint64(state.LightingSpeed)
	if speed < domain.SpeedMin {
		speed = domain.SpeedMin
	}
	if speed > domain.SpeedMax {
		speed = domain.SpeedMax
	}

	// 界面 0~4 档映射到底层硬件 1~5 档（0 档对应熄灭即硬件 1，4 档对应最高亮度即硬件 5）
	uiBrightness := uint64(state.LightingBrightness)
	hwBrightness := uiBrightness + 1
	if hwBrightness < domain.BrightnessMin {
		hwBrightness = domain.BrightnessMin
	}
	if hwBrightness > domain.BrightnessMax {
		hwBrightness = domain.BrightnessMax
	}

	effect := state.SelectedEffect
	perZone := false

	// 只有在分区设置 Tab 下且开启了独立分区时，才生效 PerZone
	if state.LightingTab == 1 && state.PerZone {
		perZone = true
		effect = domain.EffectStatic
	}

	direction := uint64(0)
	if effect.HasDirection() {
		direction = uint64(state.LightingDirection)
	}

	return domain.RGBConfig{
		Effect:     effect,
		Speed:      speed,
		Direction:  direction,
		Brightness: hwBrightness,
		Zones:      state.ZoneColors,
		PerZone:    perZone,
	}
}
