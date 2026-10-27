package machine

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// lightingData is the shape of the lighting of the service.
//
// 原厂服务把四个分区的颜色、效果、速度、方向和亮度一起报回来，这是固件
// 读不到的东西。
type lightingData struct {
	LEDs []struct {
		LEDID  int    `json:"LED_id"`
		Color  string `json:"color"`
		Status int    `json:"status"`
	} `json:"LEDs"`
	Brightness int    `json:"brightness"`
	Color      string `json:"color"`
	Effect     string `json:"effect"`
	Speed      int    `json:"speed"`
	Direction  int    `json:"direction"`
}

// LightingBrightness reads the brightness of the keyboard.
func (b *Backend) LightingBrightness() (uint64, error) {
	data, err := b.lighting()
	if err != nil {
		return 0, err
	}
	return uint64(data.Brightness), nil
}

// Backlight reads the keyboard backlight.
func (b *Backend) Backlight() (hardware.BacklightState, error) {
	data, err := b.lighting()
	if err != nil {
		return hardware.BacklightState{}, err
	}

	timeout, err := b.switchStatus(agentservice.FunctionBacklightTimeout)
	if err != nil {
		return hardware.BacklightState{}, err
	}

	return hardware.BacklightState{
		Brightness: data.Brightness,
		Timeout:    timeout == 1,
	}, nil
}

// Lighting reads the keyboard lighting.
//
// 原厂服务读得回分区颜色与灯效，所以这一项给的是机器真正的样子，不用拿
// Viper 自己记着的设置顶替。
func (b *Backend) Lighting() (hardware.RGBConfig, error) {
	data, err := b.lighting()
	if err != nil {
		return hardware.RGBConfig{}, err
	}

	result := hardware.RGBConfig{
		Effect:         serviceEffect(data.Effect),
		Speed:          uint64(data.Speed),
		Direction:      uint64(data.Direction),
		Brightness:     uint64(data.Brightness),
		PerZone:        len(data.LEDs) > 1,
		ColourReadable: true,
		EffectReadable: true,
	}

	for _, led := range data.LEDs {
		if led.LEDID < 0 || led.LEDID >= hardware.LightingZones {
			continue
		}
		colour, ok := parseColour(led.Color)
		if !ok {
			continue
		}
		result.Zones[led.LEDID] = colour
	}

	return result, nil
}

// SetLighting applies a keyboard lighting configuration.
//
// 写入的是官方应用写给原厂服务的那一组数值：四个分区的颜色，效果、速度、
// 方向和亮度。
func (b *Backend) SetLighting(config hardware.RGBConfig) error {
	speed := clamp(config.Speed, 1, 9)
	direction := clamp(config.Direction, 0, 1)
	brightness := clamp(config.Brightness, 0, 100)

	leds := make([]any, 0, hardware.LightingZones)
	for i := 0; i < hardware.LightingZones; i++ {
		colour := config.Zones[i]
		if !config.PerZone {
			colour = config.Zones[0]
		}
		leds = append(leds, map[string]any{
			"LED_id": i,
			"color":  fmt.Sprintf("#%02x%02x%02x", colour.R, colour.G, colour.B),
			"status": 1,
		})
	}

	return b.set(agentservice.FunctionLighting, map[string]any{
		"LEDs":       leds,
		"brightness": int(brightness),
		"effect":     effectName(config.Effect),
		"speed":      int(speed),
		"direction":  int(direction),
	})
}

// SetBacklight sets the brightness of the keyboard backlight.
//
// 亮度和超时是两个设置，写亮度要把当前的灯效与颜色一起写回去，否则原厂
// 服务会把分区颜色清掉。
func (b *Backend) SetBacklight(brightness int, timeout bool) error {
	data, err := b.lighting()
	if err != nil {
		return err
	}

	leds := make([]any, 0, len(data.LEDs))
	for _, led := range data.LEDs {
		leds = append(leds, map[string]any{
			"LED_id": led.LEDID,
			"color":  led.Color,
			"status": 1,
		})
	}

	if err := b.set(agentservice.FunctionLighting, map[string]any{
		"LEDs":       leds,
		"brightness": int(clamp(uint64(brightness), 0, 100)),
		"effect":     data.Effect,
		"speed":      data.Speed,
		"direction":  data.Direction,
	}); err != nil {
		return err
	}

	return b.set(agentservice.FunctionBacklightTimeout, map[string]any{
		"status": boolStatus(timeout),
	})
}

// BacklightBuffer reads the raw buffer of the keyboard backlight.
//
// 原厂服务不报这个缓冲区，所以这一项问固件。
func (b *Backend) BacklightBuffer() ([]byte, error) {
	firmware, err := b.provider()
	if err != nil {
		return nil, err
	}
	return firmware.BacklightBuffer()
}

// ResetLighting restores the default keyboard lighting.
//
// 官方应用的复位按钮是静态白色、满亮度，这个动作在固件那一侧，所以这一项
// 问固件。
func (b *Backend) ResetLighting() error {
	firmware, err := b.provider()
	if err != nil {
		return err
	}
	return firmware.ResetLighting()
}

// lighting reads the lighting of the service.
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

// serviceEffect maps the name of a pattern onto a lighting effect.
func serviceEffect(name string) hardware.LightingEffect {
	switch strings.ToUpper(name) {
	case "BREATHING":
		return hardware.EffectBreathing
	case "NEON":
		return hardware.EffectNeon
	case "WAVE":
		return hardware.EffectWave
	case "SHIFTING":
		return hardware.EffectSwitch
	case "ZOOM":
		return hardware.EffectScale
	case "METEOR":
		return hardware.EffectMeteor
	case "TWINKLING":
		return hardware.EffectFlash
	default:
		return hardware.EffectStatic
	}
}

// effectName maps a lighting effect onto the name the service uses.
func effectName(effect hardware.LightingEffect) string {
	switch effect {
	case hardware.EffectBreathing:
		return "BREATHING"
	case hardware.EffectNeon:
		return "NEON"
	case hardware.EffectWave:
		return "WAVE"
	case hardware.EffectSwitch:
		return "SHIFTING"
	case hardware.EffectScale:
		return "ZOOM"
	case hardware.EffectMeteor:
		return "METEOR"
	case hardware.EffectFlash:
		return "TWINKLING"
	default:
		return "STATIC"
	}
}

// parseColour reads a colour of the form #RRGGBB.
//
// 服务报的颜色就是这个形状，实测过 #FF0000、#00FF00、#0000FF 和 #FFFF00。
// 不认识的样子直接丢掉，不去猜它是怎么排的。
func parseColour(text string) (hardware.RGBColor, bool) {
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

// clamp limits a number to the given range.
func clamp(value, low, high uint64) uint64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
