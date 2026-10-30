package machine

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

// lightingLED 是服务报告的单个分区灯。
type lightingLED struct {
	LEDID  int    `json:"LED_id"`
	Color  string `json:"color"`
	Status int    `json:"status"`
}

// lightingData 是原厂服务 LIGHTING 功能返回的当前设置。
//
// 真机实测：这份数据是服务自己缓存的写入值，
// 不一定是硬件真正生效的状态，硬件真值要从设备列表读。
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

// lightingMode 是设备能力表里的一个灯效模式。
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

// lightingDevice 是设备列表里的一个灯光设备。
type lightingDevice struct {
	DeviceIndex     int            `json:"device_index"`
	DeviceName      string         `json:"device_name"`
	DeviceType      int            `json:"device_type"`
	Brightness      int            `json:"brightness"`
	ActiveModeIndex int            `json:"active_mode_index"`
	Modes           []lightingMode `json:"vector_of_modes"`
}

// 设备列表里的键盘灯光设备类型。
const lightingDeviceTypeKeyboard = 5

// LightingBrightness 查询当前键盘背光亮度档位。
func (b *Backend) LightingBrightness() (uint64, error) {
	device, err := b.lightingDevice()
	if err != nil {
		return 0, err
	}
	return uint64(device.Brightness), nil
}

// Backlight 读取背光亮度以及 30 秒自动休眠关闭状态。
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

// Lighting 读取完整的键盘 RGB 灯效配置。
//
// 亮度与灯效模式以设备列表为准：那才是硬件真正生效的状态；
// 颜色与速度从 LIGHTING 读，因为设备列表里没有这两项。
func (b *Backend) Lighting() (domain.RGBConfig, error) {
	device, err := b.lightingDevice()
	if err != nil {
		return domain.RGBConfig{}, err
	}

	// 分区直控写在 1 号设备上，读取时也要带上 1 号，
	// 否则服务只回 0 号那份设置，看不到分区颜色。
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

	// 设备列表里的 active_mode_index 是硬件真正生效的模式，
	// 它比 LIGHTING 缓存里的 effect 更可信。
	//
	// DIRECT 是分区直控模式：硬件按四个 LED 各自分色，
	// 这时对外就报静态常亮。
	if effect, ok := device.activeEffect(); ok {
		result.Effect = effect
	}

	found := false
	for _, led := range data.LEDs {
		// 真机实测：LED_id 的起点不统一，分区写用 0 起算、全局写用 1 起算。
		// 这里看第一个出现的编号来定基准，两种都能对上。
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

// SetLighting 向硬件下发新的 RGB 灯效设置并验证生效。
func (b *Backend) SetLighting(config domain.RGBConfig) error {
	speed := clamp(config.Speed, domain.SpeedMin, domain.SpeedMax)
	brightness := clamp(config.Brightness, domain.BrightnessMin, domain.BrightnessMax)

	// 方向只对波形与切换有意义，其余灯效固定用 0。
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
		// 分区独立配色要走 device 1，硬件才会切到 DIRECT 分区模式；
		// 它的 LED_id 从 0 起算，写 1 会被服务整次拒绝。
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

// verifyLighting 轮询设备列表，确认硬件真的换成了目标灯效与亮度。
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

	// 分区独立配色时硬件跑的是静态模式，不比对灯效名。
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

// SetBacklight 设置背光亮度及 30 秒自动关闭开关。
func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	device, err := b.lightingDevice()
	if err != nil {
		return err
	}

	// 分区直控写在 1 号设备上，改亮度时也要写回同一份，
	// 否则会把分区设置换成全局。
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

// ResetLighting 将键盘背光恢复为默认白色常亮状态。
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

// lightingFrom 读某个设备上的灯光设置。
//
// 真机实测：不带设备号时服务只回 0 号设备，
// 分区直控那一份要用 1 号才读得到。
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

// lightingDevice 读设备列表里的键盘灯光设备。
//
// 真机实测：LIGHTING 读回来的只是服务缓存的写入值，
// 设备列表里的 brightness 与 active_mode_index 才是硬件真值。
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

// activeEffect 返回设备当前真正生效的灯效模式。
func (d lightingDevice) activeEffect() (domain.LightingEffect, bool) {
	if d.ActiveModeIndex < 0 || d.ActiveModeIndex >= len(d.Modes) {
		return domain.EffectStatic, false
	}
	name := d.Modes[d.ActiveModeIndex].ModeName
	if name == "DIRECT" {
		// DIRECT 是分区直控模式，对外就报静态常亮。
		return domain.EffectStatic, true
	}
	return serviceEffect(name), true
}

// isZoneMode 判断硬件当前是不是分区直控模式。
//
// 真机实测：写 device 1 才会让硬件切到 DIRECT，
// 这时四个分区各走各的颜色。
func (d lightingDevice) isZoneMode() bool {
	if d.ActiveModeIndex < 0 || d.ActiveModeIndex >= len(d.Modes) {
		return false
	}
	return d.Modes[d.ActiveModeIndex].ModeName == "DIRECT"
}

// ledZeroBased 判断这组 LED 编号是不是从 0 起算。
//
// 真机实测：分区写用 0 起算，全局写用 1 起算。
func ledZeroBased(leds []lightingLED) bool {
	for _, led := range leds {
		if led.LEDID == 0 {
			return true
		}
	}
	return false
}

// serviceEffect 把服务里的灯效名换算成领域模型。
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

// effectName 把领域模型换算成服务里的灯效名。
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
