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
	LEDs       []lightingLED `json:"LEDs"`
	Brightness int           `json:"brightness"`
	Color      string        `json:"color"`
	Effect     string        `json:"effect"`
	Speed      int           `json:"speed"`
	Direction  int           `json:"direction"`
}

// LightingBrightness 查询当前键盘背光亮度数值。
func (b *Backend) LightingBrightness() (uint64, error) {
	data, err := b.lighting()
	if err != nil {
		return 0, err
	}
	return uint64(data.Brightness), nil
}

// Backlight 读取背光亮度以及 30 秒自动休眠关闭状态。
func (b *Backend) Backlight() (domain.BacklightState, error) {
	data, err := b.lighting()
	if err != nil {
		return domain.BacklightState{}, err
	}

	timeout, err := b.switchStatus(agentservice.FunctionBacklightTimeout)
	if err != nil {
		return domain.BacklightState{}, err
	}

	return domain.BacklightState{
		Brightness: data.Brightness,
		Timeout:    timeout == 1,
	}, nil
}

// Lighting 读取完整的键盘 RGB 灯效配置（分区颜色、效果模式、速度、方向与亮度）。
func (b *Backend) Lighting() (domain.RGBConfig, error) {
	data, err := b.lighting()
	if err != nil {
		return domain.RGBConfig{}, err
	}

	result := domain.RGBConfig{
		Effect:         serviceEffect(data.Effect),
		Speed:          uint64(data.Speed),
		Direction:      uint64(data.Direction),
		Brightness:     uint64(data.Brightness),
		PerZone:        len(data.LEDs) > 1,
		ColourReadable: true,
		EffectReadable: true,
	}

	found := false
	for _, led := range data.LEDs {
		if led.LEDID < 0 || led.LEDID >= domain.LightingZones {
			continue
		}
		colour, ok := parseColour(led.Color)
		if !ok {
			continue
		}
		result.Zones[led.LEDID] = colour
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
	speed := clamp(config.Speed, 1, 9)
	direction := clamp(config.Direction, 0, 1)
	brightness := clamp(config.Brightness, 0, 100)

	written := lightingData{
		Brightness: int(brightness),
		Effect:     effectName(config.Effect),
		Speed:      int(speed),
		Direction:  int(direction),
	}

	payload := map[string]any{
		"brightness": int(brightness),
		"effect":     written.Effect,
		"speed":      int(speed),
		"direction":  int(direction),
	}

	first := config.Zones[0]
	if !config.PerZone {
		text := fmt.Sprintf("#%02x%02x%02x", first.R, first.G, first.B)
		written.Color = text
		payload["color"] = text
	} else {
		leds := make([]any, 0, domain.LightingZones)
		for i := 0; i < domain.LightingZones; i++ {
			colour := config.Zones[i]
			text := fmt.Sprintf("#%02x%02x%02x", colour.R, colour.G, colour.B)
			written.LEDs = append(written.LEDs, lightingLED{LEDID: i, Color: text, Status: 1})
			leds = append(leds, map[string]any{
				"LED_id": i,
				"color":  text,
				"status": 1,
			})
		}
		written.Color = written.LEDs[0].Color
		payload["color"] = written.Color
		payload["LEDs"] = leds
	}

	if err := b.set(agentservice.FunctionLighting, payload); err != nil {
		return err
	}

	return b.verifyLighting(written)
}

func (b *Backend) verifyLighting(written lightingData) error {
	current := lightingData{}
	read := func() (bool, error) {
		data, err := b.lighting()
		if err != nil {
			return false, err
		}
		current = data
		return lightingMatches(current, written), nil
	}

	ok, err := poll(read)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	switch {
	case current.Brightness != written.Brightness:
		return fmt.Errorf("machine: 灯光的亮度没有生效，服务报的是 %d，不是 %d",
			current.Brightness, written.Brightness)
	case current.Speed != written.Speed:
		return fmt.Errorf("machine: 灯光的速度没有生效，服务报的是 %d，不是 %d",
			current.Speed, written.Speed)
	case current.Direction != written.Direction:
		return fmt.Errorf("machine: 灯光的方向没有生效，服务报的是 %d，不是 %d",
			current.Direction, written.Direction)
	case !strings.EqualFold(current.Effect, written.Effect):
		return fmt.Errorf("machine: 灯效没有生效，服务报的是 %q，不是 %q",
			current.Effect, written.Effect)
	case !strings.EqualFold(current.Color, written.Color):
		return fmt.Errorf("machine: 灯光的颜色没有生效，服务报的是 %q，不是 %q",
			current.Color, written.Color)
	}

	for _, led := range written.LEDs {
		found, ok := findLED(current, led.LEDID)
		if !ok {
			return fmt.Errorf("machine: 第 %d 个分区的颜色没有生效，服务没有报这个分区", led.LEDID)
		}
		if !strings.EqualFold(found.Color, led.Color) {
			return fmt.Errorf("machine: 第 %d 个分区的颜色没有生效，服务报的是 %q，不是 %q",
				led.LEDID, found.Color, led.Color)
		}
	}

	return fmt.Errorf("machine: 灯光没有生效，服务报的是 %#v", current)
}

func lightingMatches(current lightingData, written lightingData) bool {
	if current.Brightness != written.Brightness ||
		current.Speed != written.Speed ||
		current.Direction != written.Direction {
		return false
	}
	if !strings.EqualFold(current.Effect, written.Effect) {
		return false
	}
	if !strings.EqualFold(current.Color, written.Color) {
		return false
	}
	for _, led := range written.LEDs {
		found, ok := findLED(current, led.LEDID)
		if !ok || !strings.EqualFold(found.Color, led.Color) {
			return false
		}
	}
	return true
}

func findLED(data lightingData, index int) (lightingLED, bool) {
	for _, led := range data.LEDs {
		if led.LEDID == index {
			return led, true
		}
	}
	return lightingLED{}, false
}

// SetBacklight 设置背光亮度及 30 秒自动关闭开关。
func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	data, err := b.lighting()
	if err != nil {
		return err
	}

	written := data
	written.Brightness = int(clamp(uint64(brightness), 0, 100))

	payload := map[string]any{
		"brightness": written.Brightness,
		"effect":     written.Effect,
		"speed":      written.Speed,
		"direction":  written.Direction,
		"color":      written.Color,
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
		Direction:  0,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range config.Zones {
		config.Zones[i] = white
	}
	return b.SetLighting(config)
}

func (b *Backend) lighting() (lightingData, error) {
	var data lightingData

	answer, err := b.get(agentservice.FunctionLighting)
	if err != nil {
		return data, err
	}
	if err := answer.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
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
