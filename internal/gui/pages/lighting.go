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

func Lighting(c *ui.Context, ctrl *controller.Controller, state *State) {
	overview := ctrl.Snapshot()
	lighting := overview.Lighting

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

			tabIdx := state.LightingTab
			picker := ui.Segmented(c, &tabIdx, "全局设置", "分区设置").Label("灯效模式切换")
			if picker.Changed() {
				state.LightingTab = tabIdx
			}

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

			resetBtn := ui.Button(c, "恢复默认").
				Padding(8, 14).Radius(8).
				Background(t.Background).Border(1, t.Border)
			if resetBtn.Clicked() {
				ctrl.ResetLighting(nil)
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

func lightingGlobalPanel(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()

	effects := model.LightingEffects()
	currentEffect := state.SelectedEffect

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

	canColor := model.LightingEffectCanColor(currentEffect)
	hasDirection := currentEffect.HasDirection()

	if currentEffect == domain.EffectStatic {
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
		renderColorPaletteCard(c, overview, ctrl, state, t, "静态全局色彩调控")
	} else if !canColor {
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
	} else {
		ui.Row(c).Gap(16).Children(func() {
			renderBrightnessCard(c, overview, ctrl, state, t)
			renderDynamicParamsCard(c, overview, ctrl, state, t, currentEffect, hasDirection)
		})
		renderColorPaletteCard(c, overview, ctrl, state, t, "灯效色彩调控")
	}
}

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

func renderDynamicParamsCard(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State, t *ui.Theme, effect domain.LightingEffect, hasDirection bool) {
	canSpeed := model.LightingEffectCanSpeed(effect)

	widgets.Card(c, "动态参数", func() {
		ui.Column(c).Gap(12).Children(func() {
			if canSpeed {
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
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "● 当前为静态单色常亮，无需调节动态速度").
						FontSize(11).TextColor(t.TextMuted)
				})
			}

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

func renderColorPaletteCard(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State, t *ui.Theme, title string) {
	widgets.Card(c, title, func() {
		curColor := state.ZoneColors[state.SelectedZone]
		if state.LightingTab == 0 {
			curColor = state.ZoneColors[0]
		}
		uiColor := ui.RGB(curColor.R, curColor.G, curColor.B)
		hex := fmt.Sprintf("#%02X%02X%02X", curColor.R, curColor.G, curColor.B)

		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(10).AlignItems(ui.Center).Children(func() {
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

			ui.Column(c).Grow(1).Gap(14).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
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

				ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(24, 82).Radius(12).
						Background(uiColor).
						Shadow(0, 0, 14, 0, uiColor.Alpha(0.70)).
						Border(1, t.Border)

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

					ui.Column(c).AlignItems(ui.End).Children(func() {
						ui.Text(c, "HEX").FontSize(10).TextColor(t.TextMuted)
						ui.Text(c, hex).FontSize(13).Bold().TextColor(t.Accent)
					})
				})
			})
		})
	}).Grow(1)
}

func lightingZonesPanel(c *ui.Context, overview model.Overview, ctrl *controller.Controller, state *State) {
	t := c.Theme()

	widgets.Card(c, "四分区键盘色彩预览", func() {
		ui.Column(c).Gap(12).Children(func() {
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

	renderColorPaletteCard(c, overview, ctrl, state, t,
		fmt.Sprintf("色彩调控 · %s", model.ZoneName(state.SelectedZone)))
}

func applyColor(state *State, color domain.RGBColor) {
	if state.LightingTab == 1 && state.PerZone {
		state.ZoneColors[state.SelectedZone] = color
	} else {
		for i := range state.ZoneColors {
			state.ZoneColors[i] = color
		}
	}
}

func buildLightingConfig(state *State, overview model.Overview) domain.RGBConfig {
	speed := uint64(state.LightingSpeed)
	if speed < domain.SpeedMin {
		speed = domain.SpeedMin
	}
	if speed > domain.SpeedMax {
		speed = domain.SpeedMax
	}

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
