package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// The keyboard backlight hotkey identifier used by the action class.
const backlightHotkey = 132

// Keyboard lighting related bits of the action class input word.
const (
	backlightReadBit  = 1
	backlightWriteBit = 2
	backlightFlagBit  = 0x80000
)

// BacklightState describes the keyboard backlight.
type BacklightState struct {
	// Brightness is the backlight level, 0 to 100.
	Brightness int
	// Timeout reports whether the backlight switches off after 30 seconds.
	Timeout bool
}

// SetBacklight sets the keyboard backlight brightness.
//
// The provider expects the level in the generic action class and again in
// the Acer gaming class, because the two classes drive different parts of
// the keyboard controller.
func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	level := clampPercent(brightness)

	timeoutByte := uint64(0)
	if timeout {
		timeoutByte = 0x1E
	}

	actionInput := uint64(backlightWriteBit) |
		(uint64(backlightHotkey) << 8) |
		backlightFlagBit |
		(uint64(level) << 32) |
		(timeoutByte << 40)

	if _, err := b.callAction("SetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint64(actionInput),
	}); err != nil {
		return fmt.Errorf("hardware: set keyboard backlight: %w", err)
	}

	// The Acer gaming class takes the level as a sixteen byte buffer.
	buffer := make([]byte, 16)
	value := uint64(level) << 16
	for i := 0; i < 8; i++ {
		buffer[i] = byte(value >> (8 * i))
	}
	buffer[9] = 1

	if _, err := b.callGaming("SetGamingKBBacklight", map[string]wmi.Value{
		"gmInput": wmi.Bytes(buffer),
	}); err != nil {
		return fmt.Errorf("hardware: set keyboard backlight buffer: %w", err)
	}

	if _, err := b.callGaming("SetGamingRgbKb", map[string]wmi.Value{
		"gmInput": wmi.Uint64(1 | (uint64(level) << 8)),
	}); err != nil {
		return fmt.Errorf("hardware: set keyboard backlight level: %w", err)
	}

	return nil
}

// Backlight reads the keyboard backlight state.
func (b *Backend) Backlight() (BacklightState, error) {
	var state BacklightState

	query := uint64(backlightReadBit) |
		(uint64(backlightHotkey) << 8) |
		backlightFlagBit

	outputs, err := b.callAction("GetFunction", map[string]wmi.Value{
		"uiInput": wmi.Uint32(uint32(query)),
	})
	if err != nil {
		return state, err
	}
	raw, err := decodeUint(outputs["uiOutput"])
	if err != nil {
		return state, err
	}
	if raw&0xFF == 0 && raw > 0 {
		state.Brightness = int((raw >> 32) & 0xFF)
		state.Timeout = (raw>>40)&0xFF == 30
	}
	return state, nil
}

// SetLighting applies a keyboard lighting configuration.
func (b *Backend) SetLighting(config RGBConfig) error {
	for zone := 0; zone < 4; zone++ {
		color := config.Zones[zone]
		input := (uint64(color.B) << 24) |
			(uint64(color.G) << 16) |
			(uint64(color.R) << 8) |
			uint64(zone+1)

		if _, err := b.callGaming("SetGamingLEDColor", map[string]wmi.Value{
			"gmInput": wmi.Uint64(input),
		}); err != nil {
			return fmt.Errorf("hardware: set lighting zone %d: %w", zone+1, err)
		}
	}

	behavior := (config.Effect << 16) |
		((config.Speed & 0xFF) << 8) |
		(config.Direction & 0xFF)

	if _, err := b.callGaming("SetGamingLEDBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint64(behavior),
	}); err != nil {
		return fmt.Errorf("hardware: set lighting behaviour: %w", err)
	}

	return b.SetBacklight(clampPercent(int(config.Brightness)), true)
}

// Lighting reads the keyboard lighting configuration.
func (b *Backend) Lighting() (RGBConfig, error) {
	var config RGBConfig

	outputs, err := b.callGaming("GetGamingLEDBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint32(0),
	})
	if err != nil {
		return config, err
	}
	if raw, err := decodeUint(outputs["gmOutput"]); err == nil {
		config.Effect = (raw >> 16) & 0xFF
		config.Speed = (raw >> 8) & 0xFF
		config.Direction = raw & 0xFF
	}

	for zone := 0; zone < 4; zone++ {
		outputs, err := b.callGaming("GetGamingLEDColor", map[string]wmi.Value{
			"gmInput": wmi.Uint32(uint32(zone + 1)),
		})
		if err != nil {
			continue
		}
		raw, err := decodeUint(outputs["gmOutput"])
		if err != nil {
			continue
		}
		config.Zones[zone] = RGBColor{
			B: uint8(raw >> 24),
			G: uint8(raw >> 16),
			R: uint8(raw >> 8),
		}
	}

	if state, err := b.Backlight(); err == nil {
		config.Brightness = uint64(state.Brightness)
	}

	return config, nil
}
