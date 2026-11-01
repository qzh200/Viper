package machine

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

type lightingLED struct {
	LEDID  int    `json:"LED_id"`
	Color  string `json:"color"`
	Status int    `json:"status"`
}

type lightingData struct {
	LEDs       []lightingLED     `json:"LEDs"`
	Brightness int               `json:"brightness"`
	Color      string            `json:"color"`
	Colortype  int               `json:"colortype"`
	Device     int               `json:"device"`
	Direction  int               `json:"direction"`
	Duration   int               `json:"duration"`
	Effect     string            `json:"effect"`
	Speed      int               `json:"speed"`
	Subindex   map[string]string `json:"subindex"`
}

type lightingMode struct {
	ModeName     string `json:"mode_name"`
	ModeValue    int    `json:"mode_value"`
	MinSpeed     int    `json:"minimum_speed"`
	MaxSpeed     int    `json:"maximum_speed"`
	Direction    uint32 `json:"direction"`
	ModeFlags    int    `json:"mode_flags"`
	ColorMode    int    `json:"color_mode"`
	SpeedValue   int    `json:"speed_value"`
	MaximumColor int    `json:"maximum_number_of_colors"`
}

type lightingDevice struct {
	DeviceIndex     int            `json:"device_index"`
	DeviceName      string         `json:"device_name"`
	DeviceType      int            `json:"device_type"`
	Brightness      int            `json:"brightness"`
	ActiveModeIndex int            `json:"active_mode_index"`
	Modes           []lightingMode `json:"vector_of_modes"`
}

const lightingDeviceTypeKeyboard = 5

func (b *Backend) LightingBrightness() (uint64, error) {
	device, err := b.lightingDevice()
	if err != nil {
		return 0, err
	}
	return uint64(device.Brightness), nil
}

func (b *Backend) Backlight() (domain.BacklightState, error) {
	device, err := b.lightingDevice()
	if err != nil {
		return domain.BacklightState{}, err
	}

	timeout, err := b.switchStatus(agentservice.FunctionBacklightTimeout)
	if err != nil {
		return domain.BacklightState{}, err
	}

	return domain.BacklightState{
		Brightness: device.Brightness,
		Timeout:    timeout == 1,
	}, nil
}

func (b *Backend) Lighting() (domain.RGBConfig, error) {
	device, err := b.lightingDevice()
	if err != nil {
		return domain.RGBConfig{}, err
	}

	target := 0
	if device.isZoneMode() {
		target = 1
	}

	data, err := b.lightingFrom(target)
	if err != nil {
		return domain.RGBConfig{}, err
	}

	result := domain.RGBConfig{
		Effect:         serviceEffect(data.Effect),
		Speed:          uint64(data.Speed),
		Direction:      uint64(data.Direction),
		Brightness:     uint64(device.Brightness),
		PerZone:        target == 1,
		ColourReadable: true,
		EffectReadable: true,
	}

	if effect, ok := device.activeEffect(); ok {
		result.Effect = effect
	}

	found := false
	for _, led := range data.LEDs {
		index := led.LEDID
		if !ledZeroBased(data.LEDs) {
			index = led.LEDID - 1
		}
		if index < 0 || index >= domain.LightingZones {
			continue
		}
		colour, ok := parseColour(led.Color)
		if !ok {
			continue
		}
		result.Zones[index] = colour
		found = true
	}

	if !found {
		if colour, ok := parseColour(data.Color); ok {
			result.Zones[0] = colour
		}
	}

	return result, nil
}

func (b *Backend) SetLighting(config domain.RGBConfig) error {
	speed := clamp(config.Speed, domain.SpeedMin, domain.SpeedMax)
	brightness := clamp(config.Brightness, domain.BrightnessMin, domain.BrightnessMax)

	direction := uint64(0)
	if config.Effect.HasDirection() {
		direction = config.Direction
	}

	effect := effectName(config.Effect)
	colour := fmt.Sprintf("#%02x%02x%02x",
		config.Zones[0].R, config.Zones[0].G, config.Zones[0].B)

	written := lightingData{
		Brightness: int(brightness),
		Color:      colour,
		Colortype:  1,
		Device:     0,
		Direction:  int(direction),
		Duration:   3,
		Effect:     effect,
		Speed:      int(speed),
		Subindex:   map[string]string{"1": effect, "2": effect},
	}

	payload := map[string]any{
		"brightness": int(brightness),
		"color":      colour,
		"colortype":  1,
		"device":     0,
		"direction":  int(direction),
		"duration":   3,
		"effect":     effect,
		"speed":      int(speed),
		"subindex":   map[string]any{"1": effect, "2": effect},
	}

	if config.PerZone {
		written.Device = 1
		written.Effect = "STATIC"
		written.Subindex = map[string]string{"1": "STATIC", "2": "STATIC"}
		payload["device"] = 1
		payload["effect"] = "STATIC"
		payload["subindex"] = map[string]any{"1": "STATIC", "2": "STATIC"}

		leds := make([]any, 0, domain.LightingZones)
		for i := 0; i < domain.LightingZones; i++ {
			text := fmt.Sprintf("#%02x%02x%02x",
				config.Zones[i].R, config.Zones[i].G, config.Zones[i].B)
			written.LEDs = append(written.LEDs, lightingLED{LEDID: i, Color: text, Status: 1})
			leds = append(leds, map[string]any{
				"LED_id": i,
				"color":  text,
				"status": 1,
			})
		}
		payload["LEDs"] = leds
	}

	if err := b.set(agentservice.FunctionLighting, payload); err != nil {
		return err
	}

	return b.verifyLighting(written)
}

func (b *Backend) verifyLighting(written lightingData) error {
	current := lightingDevice{}

	read := func() (bool, error) {
		device, err := b.lightingDevice()
		if err != nil {
			return false, err
		}
		current = device
		return device.Brightness == written.Brightness, nil
	}

	ok, err := poll(read)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("machine: 灯光的亮度没有生效，设备报的是 %d，不是 %d",
			current.Brightness, written.Brightness)
	}

	if len(written.LEDs) > 0 {
		return nil
	}

	effect, ok := current.activeEffect()
	if !ok {
		return nil
	}
	if effect != serviceEffect(written.Effect) {
		return fmt.Errorf("machine: 灯效没有生效，设备跑的是 %v，不是 %v",
			effect, serviceEffect(written.Effect))
	}

	return nil
}

func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	device, err := b.lightingDevice()
	if err != nil {
		return err
	}

	target := 0
	if device.isZoneMode() {
		target = 1
	}

	data, err := b.lightingFrom(target)
	if err != nil {
		return err
	}

	written := data
	written.Brightness = int(clamp(uint64(brightness), domain.BrightnessMin, domain.BrightnessMax))
	written.Colortype = 1
	written.Device = target
	written.Duration = 3
	written.Subindex = map[string]string{"1": written.Effect, "2": written.Effect}

	payload := map[string]any{
		"brightness": written.Brightness,
		"effect":     written.Effect,
		"speed":      written.Speed,
		"direction":  written.Direction,
		"color":      written.Color,
		"colortype":  1,
		"device":     target,
		"duration":   3,
		"subindex":   map[string]any{"1": written.Effect, "2": written.Effect},
	}
	if len(written.LEDs) > 0 {
		leds := make([]any, 0, len(written.LEDs))
		for _, led := range written.LEDs {
			leds = append(leds, map[string]any{
				"LED_id": led.LEDID,
				"color":  led.Color,
				"status": 1,
			})
		}
		payload["LEDs"] = leds
	}

	if err := b.set(agentservice.FunctionLighting, payload); err != nil {
		return err
	}
	if err := b.verifyLighting(written); err != nil {
		return err
	}

	if err := b.set(agentservice.FunctionBacklightTimeout, map[string]any{
		"status": boolStatus(timeout),
	}); err != nil {
		return err
	}
	return b.verifyStatus(agentservice.FunctionBacklightTimeout, boolStatus(timeout))
}

func (b *Backend) ResetLighting() error {
	white := domain.RGBColor{R: 255, G: 255, B: 255}
	config := domain.RGBConfig{
		Effect:     domain.EffectStatic,
		Speed:      3,
		Brightness: domain.BrightnessMax,
		PerZone:    false,
	}
	for i := range config.Zones {
		config.Zones[i] = white
	}
	return b.SetLighting(config)
}

func (b *Backend) lighting() (lightingData, error) {
	return b.lightingFrom(0)
}

func (b *Backend) lightingFrom(device int) (lightingData, error) {
	var data lightingData

	answer, err := b.getFrom(agentservice.FunctionLighting, device)
	if err != nil {
		return data, err
	}
	if err := answer.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}

func (b *Backend) lightingDevice() (lightingDevice, error) {
	answer, err := b.service.Devices()
	if err != nil {
		return lightingDevice{}, fmt.Errorf("machine: 读取灯光设备列表失败: %w", err)
	}
	if err, failed := answer.Err(); failed {
		return lightingDevice{}, err
	}

	var devices []lightingDevice
	if err := answer.Decode(&devices); err != nil {
		return lightingDevice{}, fmt.Errorf("machine: 灯光设备列表无法解析: %w", err)
	}

	for _, device := range devices {
		if device.DeviceType == lightingDeviceTypeKeyboard {
			return device, nil
		}
	}
	return lightingDevice{}, fmt.Errorf("machine: 设备列表里没有键盘灯光设备")
}

func (d lightingDevice) activeEffect() (domain.LightingEffect, bool) {
	if d.ActiveModeIndex < 0 || d.ActiveModeIndex >= len(d.Modes) {
		return domain.EffectStatic, false
	}
	name := d.Modes[d.ActiveModeIndex].ModeName
	if name == "DIRECT" {
		return domain.EffectStatic, true
	}
	return serviceEffect(name), true
}

func (d lightingDevice) isZoneMode() bool {
	if d.ActiveModeIndex < 0 || d.ActiveModeIndex >= len(d.Modes) {
		return false
	}
	return d.Modes[d.ActiveModeIndex].ModeName == "DIRECT"
}

func ledZeroBased(leds []lightingLED) bool {
	for _, led := range leds {
		if led.LEDID == 0 {
			return true
		}
	}
	return false
}

func serviceEffect(name string) domain.LightingEffect {
	switch strings.ToUpper(name) {
	case "BREATHING":
		return domain.EffectBreathing
	case "NEON":
		return domain.EffectNeon
	case "WAVE":
		return domain.EffectWave
	case "SHIFTING":
		return domain.EffectSwitch
	case "ZOOM":
		return domain.EffectScale
	case "METEOR":
		return domain.EffectMeteor
	case "TWINKLING":
		return domain.EffectFlash
	default:
		return domain.EffectStatic
	}
}

func effectName(effect domain.LightingEffect) string {
	switch effect {
	case domain.EffectBreathing:
		return "BREATHING"
	case domain.EffectNeon:
		return "NEON"
	case domain.EffectWave:
		return "WAVE"
	case domain.EffectSwitch:
		return "SHIFTING"
	case domain.EffectScale:
		return "ZOOM"
	case domain.EffectMeteor:
		return "METEOR"
	case domain.EffectFlash:
		return "TWINKLING"
	default:
		return "STATIC"
	}
}

func parseColour(text string) (domain.RGBColor, bool) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(text) != 6 {
		return domain.RGBColor{}, false
	}

	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return domain.RGBColor{}, false
	}
	return domain.RGBColor{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
	}, true
}

func clamp(value, low, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
