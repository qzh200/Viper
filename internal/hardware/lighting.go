package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// The keyboard backlight hotkey identifier used by the action class.
const backlightHotkey = 132

// backlightTimeoutSeconds is the number of seconds after which the
// backlight switches off on its own.
const backlightTimeoutSeconds = 30

// Keyboard lighting related bits of the action class input word.
const (
	backlightReadBit  = 1
	backlightWriteBit = 2
	backlightFlagBit  = 0x80000
)

// The number of lighting zones the keyboard has.
const LightingZones = 4

// BacklightState describes the keyboard backlight.
type BacklightState struct {
	// Brightness is the backlight level, 0 to 100.
	Brightness int
	// Timeout reports whether the backlight switches off after 30 seconds.
	Timeout bool
}

// SetLighting applies a keyboard lighting configuration.
//
// The provider applies the effect, the speed and the direction to the whole
// keyboard at once, while the colour is written per zone. A global setup
// therefore writes the same colour to every zone.
func (b *Backend) SetLighting(config RGBConfig) error {
	colours := config.Zones
	if !config.PerZone {
		for i := 1; i < LightingZones; i++ {
			colours[i] = colours[0]
		}
	}

	for zone := 0; zone < LightingZones; zone++ {
		if _, err := b.callGaming("SetGamingLEDColor", map[string]wmi.Value{
			"gmInput": wmi.Uint64(colorWord(colours[zone], uint64(zone+1))),
		}); err != nil {
			return fmt.Errorf("hardware: set lighting zone %d: %w", zone+1, err)
		}
	}

	behavior := (uint64(config.Effect) << 16) |
		((config.Speed & 0xFF) << 8) |
		(config.Direction & 0xFF)

	if _, err := b.callGaming("SetGamingLEDBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint64(behavior),
	}); err != nil {
		return fmt.Errorf("hardware: set lighting behaviour: %w", err)
	}

	// The automatic switch off is a setting of its own, changing the
	// lighting must not change it.
	timeout := false
	if state, err := b.Backlight(); err == nil {
		timeout = state.Timeout
	}
	return b.SetBacklight(clampPercent(int(config.Brightness)), timeout)
}

// ResetLighting returns the keyboard to the default appearance.
//
// The default is a static white effect in every zone at full brightness,
// which is what the official application does with its reset button.
func (b *Backend) ResetLighting() error {
	white := RGBColor{R: 255, G: 255, B: 255}
	config := RGBConfig{
		Effect:     EffectStatic,
		Speed:      3,
		Direction:  0,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range config.Zones {
		config.Zones[i] = white
	}
	return b.SetLighting(config)
}

// Lighting reads the keyboard lighting configuration.
//
// Only the brightness can be read back from the firmware. The provider
// reports neither the colour nor the effect of the zones, so those fields
// are reported as unsupported and the caller keeps its own record.
func (b *Backend) Lighting() (RGBConfig, error) {
	var config RGBConfig

	brightness, err := b.LightingBrightness()
	if err != nil {
		return config, err
	}
	config.Brightness = brightness

	config.ColourReadable = false
	config.EffectReadable = false
	return config, nil
}

// LightingBrightness reads the keyboard brightness.
//
// GetGamingRgbKb reports the level of the keyboard in its second byte. The
// provider clamps the level to the range from zero to one hundred, so a
// larger value in the firmware is an uninitialised state rather than a
// brighter keyboard.
func (b *Backend) LightingBrightness() (uint64, error) {
	outputs, err := b.callGaming("GetGamingRgbKb", map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return 0, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return 0, err
	}
	level := (raw >> 8) & 0xFF
	if level > 100 {
		level = 100
	}
	return level, nil
}

// colorWord builds the word that carries the colour of one zone.
//
// The word is blue, green and red from the top down and the zone number in
// the lowest byte. The zones are counted from one.
func colorWord(color RGBColor, zone uint64) uint64 {
	return (uint64(color.B) << 24) |
		(uint64(color.G) << 16) |
		(uint64(color.R) << 8) |
		zone
}

// backlightBuffer builds the sixteen byte buffer of the keyboard backlight.
//
// The layout was read back from the provider with GetGamingKBBacklight:
// the level sits in byte two and in byte five, byte eight holds the flags
// of the backlight and byte nine has to be one, otherwise the provider
// drops the buffer and zeroes the level.
func backlightBuffer(level int) []byte {
	buffer := make([]byte, 16)
	buffer[2] = byte(level)
	buffer[5] = byte(level)
	buffer[8] = backlightFlags
	buffer[9] = 1
	return buffer
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
		timeoutByte = backlightTimeoutSeconds
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
	if _, err := b.callGaming("SetGamingKBBacklight", map[string]wmi.Value{
		"gmInput": wmi.Bytes(backlightBuffer(level)),
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

// backlightFlags is the value of the flags byte of the backlight buffer.
//
// The provider reports three for a keyboard that switches its backlight off
// after thirty seconds.
const backlightFlags = 3

// Backlight reads the keyboard backlight state.
//
// The level comes from the gaming class, the automatic switch off comes
// from the action class, which keeps it in the sixth byte of its answer.
func (b *Backend) Backlight() (BacklightState, error) {
	var state BacklightState

	brightness, err := b.LightingBrightness()
	if err != nil {
		return state, err
	}
	state.Brightness = int(brightness)

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
		state.Timeout = (raw>>40)&0xFF == backlightTimeoutSeconds
	}
	return state, nil
}
