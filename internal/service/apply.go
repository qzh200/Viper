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
	default:
		return fmt.Errorf("service: the setting %q cannot be written", function)
	}
}

// applyOperatingMode writes the performance profile.
func (p *Provider) applyOperatingMode(parameter any) error {
	value, err := number(parameter, "mode")
	if err != nil {
		return err
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
	default:
		return fmt.Errorf("service: the profile %d is not known", value)
	}

	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}
	return p.backend.SetPowerMode(mode)
}

// applyGPUMode writes the graphics switch.
//
// The service counts the modes from zero, the provider counts them from
// one.
func (p *Provider) applyGPUMode(parameter any) error {
	value, err := number(parameter, "mode")
	if err != nil {
		return err
	}

	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}
	return p.backend.SetGPUMode(hardware.GPUMode(value + 1))
}

// applyStickyKey writes the sticky keys.
func (p *Provider) applyStickyKey(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return err
	}
	return hardware.SetStickyKeys(value != 0)
}

// applyBacklightTimeout writes the automatic switch off of the backlight.
func (p *Provider) applyBacklightTimeout(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return err
	}
	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}

	brightness, err := p.backend.LightingBrightness()
	if err != nil {
		return err
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
		return err
	}
	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}
	return p.backend.SetWinKeyLocked(value == 0)
}

// applyLCDOverdrive writes the overdrive of the display.
func (p *Provider) applyLCDOverdrive(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return err
	}
	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}
	return p.backend.SetLCDOverdrive(value != 0)
}

// applyBootSound writes the sound at the start of the machine.
func (p *Provider) applyBootSound(parameter any) error {
	value, err := number(parameter, "status")
	if err != nil {
		return err
	}
	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
	}
	return p.backend.SetBootSound(value != 0)
}

// applyLighting writes the keyboard lighting.
//
// The parameter carries the pattern, the speed, the direction, the
// brightness and the colour of every zone.
func (p *Provider) applyLighting(parameter any) error {
	values, ok := parameter.(map[string]any)
	if !ok {
		return fmt.Errorf("service: the lighting needs a set of values")
	}
	if p.backend == nil {
		return fmt.Errorf("service: the gaming provider is not available")
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
		lighting.Speed = uint64(speed)
	}
	if direction, err := number(values, "direction"); err == nil {
		lighting.Direction = uint64(direction)
	}
	if brightness, err := number(values, "brightness"); err == nil {
		lighting.Brightness = uint64(brightness)
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
		return err
	}
	if p.store != nil {
		return p.store.Update(func(s *config.Settings) {
			s.Lighting = storedLighting(lighting)
		})
	}
	return nil
}

// storedLighting converts a lighting configuration into its stored form.
func storedLighting(value hardware.RGBConfig) config.Lighting {
	stored := config.Lighting{
		Effect:     uint64(value.Effect),
		Speed:      value.Speed,
		Direction:  value.Direction,
		Brightness: value.Brightness,
		PerZone:    value.PerZone,
	}
	for i, colour := range value.Zones {
		stored.Zones[i] = config.LightingColor{R: colour.R, G: colour.G, B: colour.B}
	}
	return stored
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
func number(values any, name string) (int64, error) {
	table, ok := values.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("service: the value %q is not a number", name)
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
			return 0, fmt.Errorf("service: the value %q is not a number", name)
		}
		return value, nil
	default:
		return 0, fmt.Errorf("service: the value %q is not a number", name)
	}
}
