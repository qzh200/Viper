package machine

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// lightingLED is one zone of the lighting.
type lightingLED struct {
	// LEDID is the number of the zone, it starts at zero.
	LEDID int `json:"LED_id"`
	// Color is the colour of the zone, of the form #RRGGBB.
	Color string `json:"color"`
	// Status is the state of the zone.
	Status int `json:"status"`
}

// lightingData is the shape of the lighting of the service.
//
// 原厂服务把四个分区的颜色、效果、速度、方向和亮度一起报回来，这是固件
// 读不到的东西。
type lightingData struct {
	LEDs       []lightingLED `json:"LEDs"`
	Brightness int           `json:"brightness"`
	Color      string        `json:"color"`
	Effect     string        `json:"effect"`
	Speed      int           `json:"speed"`
	Direction  int           `json:"direction"`
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

	found := false
	for _, led := range data.LEDs {
		if led.LEDID < 0 || led.LEDID >= hardware.LightingZones {
			continue
		}
		colour, ok := parseColour(led.Color)
		if !ok {
			continue
		}
		result.Zones[led.LEDID] = colour
		found = true
	}

	// 服务在 "LEDs" 为空时用的是全局颜色，这时把它放进第一个分区，上层
	// 回写的时候拿到的就是机器现在的样子。
	if !found {
		if colour, ok := parseColour(data.Color); ok {
			result.Zones[0] = colour
		}
	}

	return result, nil
}

// SetLighting applies a keyboard lighting configuration.
//
// 写入的是官方应用写给原厂服务的那一组数值：颜色，效果、速度、方向和亮度。
//
// 分区颜色与全局颜色是两回事：服务读回来的灯光里，"LEDs" 是四个分区各一个
// 颜色，"color" 是全局的颜色。分区设置写 "LEDs"，全局设置只写 "color"——
// 实测服务在 "LEDs" 为空时用的是全局颜色。
//
// 写完之后把配置读回来核对，机器没有接受就在错误信息里说清楚是哪一个。
func (b *Backend) SetLighting(config hardware.RGBConfig) error {
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
		// 全局设置：只写全局颜色。
		text := fmt.Sprintf("#%02x%02x%02x", first.R, first.G, first.B)
		written.Color = text
		payload["color"] = text
	} else {
		leds := make([]any, 0, hardware.LightingZones)
		for i := 0; i < hardware.LightingZones; i++ {
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

// verifyLighting reads the lighting back and compares it with what was written.
//
// 原厂服务把写下的值原样报回来，所以读回来的不一样，就是机器没有接受。
func (b *Backend) verifyLighting(written lightingData) error {
	current := lightingData{}
	read := func() (bool, error) {
		data, err := b.lighting()
		if err != nil {
			return false, err
		}
		current = data
		return lightingMatches(data, written), nil
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
		return fmt.Errorf("machine: 键盘亮度没有生效，服务报的是 %d，不是 %d",
			current.Brightness, written.Brightness)
	case current.Speed != written.Speed:
		return fmt.Errorf("machine: 灯效速度没有生效，服务报的是 %d，不是 %d",
			current.Speed, written.Speed)
	case current.Direction != written.Direction:
		return fmt.Errorf("machine: 灯效方向没有生效，服务报的是 %d，不是 %d",
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

// lightingMatches tells whether a reading is the lighting that was written.
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

// findLED finds the zone of a number.
func findLED(data lightingData, index int) (lightingLED, bool) {
	for _, led := range data.LEDs {
		if led.LEDID == index {
			return led, true
		}
	}
	return lightingLED{}, false
}

// SetBacklight sets the brightness of the keyboard backlight.
//
// 亮度和超时是两个设置，写亮度要把当前的灯效与颜色一起写回去，否则原厂
// 服务会把分区颜色清掉。写完两个设置都要读回核对。
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
