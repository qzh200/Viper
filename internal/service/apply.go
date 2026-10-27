package service

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/config"
	"viper/internal/hardware"
)

// Apply writes a setting.
//
// The parameter is the value the official application sends, the shape of
// it depends on the setting.
//
// The settings that the firmware cannot give back are remembered by the
// service itself, see the comment of rememberedSettings.
func (p *Provider) Apply(function string, parameter any) error {
	switch function {
	case "OPERATING_MODE":
		return p.applyOperatingMode(parameter)
	case "GPU_MODE":
		return p.applyGPUMode(parameter)
	case "STICKY_KEY":
		return p.applyStickyKey(parameter)
	case "BACKLIGHT_30S":
		return p.applyBacklightTimeout(parameter)
	case "WIN_KEY":
		return p.applyWinKey(parameter)
	case "LCD_OVERDRIVE":
		return p.applyLCDOverdrive(parameter)
	case "BOOT_SOUND":
		return p.applyBootSound(parameter)
	case "LIGHTING":
		return p.applyLighting(parameter)
	case "BATTERY_BOOST":
		return p.applyBatteryBoost(parameter)
	case "SOUND_MODE":
		return p.applySoundMode(parameter)
	case "CUSTOM_BOOT_LOGO":
		return p.applyBootLogo(parameter)
	default:
		return fmt.Errorf("service: 设置 %q 无法写入，未知的设置名", function)
	}
}

// applyOperatingMode writes the performance profile.
func (p *Provider) applyOperatingMode(parameter any) error {
	value, err := number(parameter, "mode")
	if err != nil {
		return fmt.Errorf("service: 性能档位的数值无效: %w", err)
	}

	var mode hardware.PowerMode
	switch value {
	case 0:
		mode = hardware.PowerQuiet
	case 1:
		mode = hardware.PowerBalanced
	case 4:
		mode = hardware.PowerPerformance
	case 5:
		mode = hardware.PowerTurbo
	case 6:
		mode = hardware.PowerEco
	default:
		return fmt.Errorf("service: 性能档位 %d 不可用，可用的是 0、1、4、5、6", value)
	}

	if !p.hasBackend() {
		return errNoBackend
	}
	return p.backend.SetPowerMode(mode)
}

// applyGPUMode writes the graphics switch.
//
// The service counts the modes from zero, the provider counts them from
// one. Only the three modes that the machine knows are accepted, a number
// outside of them would put a mode into the firmware that it does not
// know.
func (p *Provider) applyGPUMode(parameter any) error {
	value, err := number(parameter, "mode")
	if err != nil {
		return fmt.Errorf("service: 图形模式的数值无效: %w", err)
	}

	switch value {
	case 0, 1, 2:
	default:
		return fmt.Errorf("service: 图形模式 %d 不可用，可用的是 0、1、2", value)
	}

	if !p.hasBackend() {
		return errNoBackend
	}
	return p.backend.SetGPUMode(hardware.GPUMode(value + 1))
}

// applyStickyKey writes the sticky keys.
func (p *Provider) applyStickyKey(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: 粘滞键的数值无效: %w", err)
	}
	return hardware.SetStickyKeys(value != 0)
}

// applyBacklightTimeout writes the automatic switch off of the backlight.
func (p *Provider) applyBacklightTimeout(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: 背光超时的数值无效: %w", err)
	}
	if !p.hasBackend() {
		return errNoBackend
	}

	brightness, err := p.backend.LightingBrightness()
	if err != nil {
		return fmt.Errorf("service: 键盘亮度读取失败，背光超时无法写入: %w", err)
	}
	return p.backend.SetBacklight(int(brightness), value != 0)
}

// applyWinKey writes the lock of the Windows key.
//
// The service tells whether the Windows key may be used, so a status of
// one means that the key works.
func (p *Provider) applyWinKey(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: Windows 键的数值无效: %w", err)
	}
	if !p.hasBackend() {
		return errNoBackend
	}
	return p.backend.SetWinKeyLocked(value == 0)
}

// applyLCDOverdrive writes the overdrive of the display.
func (p *Provider) applyLCDOverdrive(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: 屏幕超频的数值无效: %w", err)
	}
	if !p.hasBackend() {
		return errNoBackend
	}
	return p.backend.SetLCDOverdrive(value != 0)
}

// applyBootSound writes the sound at the start of the machine.
func (p *Provider) applyBootSound(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return fmt.Errorf("service: 开机音效的数值无效: %w", err)
	}
	if !p.hasBackend() {
		return errNoBackend
	}
	return p.backend.SetBootSound(value != 0)
}

// applyLighting writes the keyboard lighting.
//
// The parameter carries the pattern, the speed, the direction, the
// brightness and the colour of every zone.
//
// The numbers are limited to the range that the firmware accepts, the same
// limits that the application layer uses: the speed runs from one to nine,
// the direction is a switch and the brightness runs from zero to one
// hundred. A number outside of them is pulled into the range instead of
// being handed to the firmware.
func (p *Provider) applyLighting(parameter any) error {
	values, ok := parameter.(map[string]any)
	if !ok {
		return fmt.Errorf("service: 灯效需要一组数值")
	}
	if !p.hasBackend() {
		return errNoBackend
	}

	lighting := hardware.RGBConfig{
		Effect:     hardware.EffectStatic,
		Speed:      3,
		Direction:  0,
		Brightness: 50,
		PerZone:    true,
	}

	if effect, ok := values["effect"].(string); ok {
		if value, known := lightingEffectValue(effect); known {
			lighting.Effect = value
		}
	}
	if speed, err := number(values, "speed"); err == nil {
		lighting.Speed = uint64(limitValue(speed, 1, 9))
	}
	if direction, err := number(values, "direction"); err == nil {
		lighting.Direction = uint64(limitValue(direction, 0, 1))
	}
	if brightness, err := number(values, "brightness"); err == nil {
		lighting.Brightness = uint64(limitValue(brightness, 0, 100))
	}

	if leds, ok := values["LEDs"].([]any); ok {
		for _, entry := range leds {
			led, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			index, err := number(led, "LED_id")
			if err != nil || index < 0 || index >= int64(len(lighting.Zones)) {
				continue
			}
			text, _ := led["color"].(string)
			colour, ok := parseColor(text)
			if !ok {
				continue
			}
			lighting.Zones[index] = colour
		}
		lighting.PerZone = true
	} else if text, ok := values["color"].(string); ok {
		colour, known := parseColor(text)
		if known {
			for i := range lighting.Zones {
				lighting.Zones[i] = colour
			}
			lighting.PerZone = false
		}
	}

	if err := p.backend.SetLighting(lighting); err != nil {
		return fmt.Errorf("service: 灯效写入失败: %w", err)
	}
	if p.store != nil {
		return p.store.Update(func(s *config.Settings) {
			s.Lighting = storedLighting(lighting)
		})
	}
	return nil
}

// storedLighting converts a lighting configuration into its stored form.
//
// The values are limited to the range that the firmware accepts, so that
// the stored settings never claim more than the keyboard can do. The
// limits are the ones of the application layer: the brightness runs from
// zero to one hundred, the speed from one to nine and the direction is a
// switch. Both layers have to mean the same thing, otherwise the value
// that the official application sees would differ from the value that
// Viper keeps.
func storedLighting(value hardware.RGBConfig) config.Lighting {
	stored := config.Lighting{
		Effect:     uint64(value.Effect),
		Speed:      clamp(value.Speed, 1, 9),
		Direction:  clamp(value.Direction, 0, 1),
		Brightness: clamp(value.Brightness, 0, 100),
		PerZone:    value.PerZone,
	}
	for i, colour := range value.Zones {
		stored.Zones[i] = config.LightingColor{R: colour.R, G: colour.G, B: colour.B}
	}
	return stored
}

// clamp limits a number to the given range.
func clamp(value uint64, low uint64, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// limitValue limits a number of a request to the given range.
func limitValue(value int64, low int64, high int64) int64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// lightingEffectValue gives the pattern of a name.
func lightingEffectValue(name string) (hardware.LightingEffect, bool) {
	switch strings.ToUpper(name) {
	case "DIRECT":
		return hardware.EffectStatic, true
	case "STATIC":
		return hardware.EffectStatic, true
	case "BREATHING":
		return hardware.EffectBreathing, true
	case "NEON":
		return hardware.EffectNeon, true
	case "WAVE":
		return hardware.EffectWave, true
	case "SHIFTING":
		return hardware.EffectSwitch, true
	case "ZOOM":
		return hardware.EffectScale, true
	case "METEOR":
		return hardware.EffectMeteor, true
	case "TWINKLING":
		return hardware.EffectFlash, true
	default:
		return hardware.EffectStatic, false
	}
}

// parseColor reads a colour of the form #RRGGBB.
func parseColor(text string) (hardware.RGBColor, bool) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(text) != 6 {
		return hardware.RGBColor{}, false
	}
	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return hardware.RGBColor{}, false
	}
	return hardware.RGBColor{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
	}, true
}

// number reads a number of a set of values.
//
// The numbers of a request come out of a JSON document, so they arrive as
// a float. A client that sends a text with a number in it is understood
// as well.
func number(values any, name string) (int64, error) {
	table, ok := values.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("service: 数值 %q 不在一组数值里", name)
	}
	switch typed := table[name].(type) {
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case string:
		value, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("service: 数值 %q 不是数字", name)
		}
		return value, nil
	default:
		return 0, fmt.Errorf("service: 数值 %q 不是数字", name)
	}
}
