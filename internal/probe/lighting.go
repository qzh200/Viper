package probe

import "viper/internal/agentservice"

// LightingLED 表示单个背光分区 LED 的 ID 与 HEX 颜色。
type LightingLED struct {
	LEDID int    `json:"LED_id"`
	Color string `json:"color"`
}

// LightingReading 记录从服务获取的原始灯效完整参数。
type LightingReading struct {
	LEDs       []LightingLED     `json:"LEDs"`
	Color      string            `json:"color"`
	Brightness int               `json:"brightness"`
	Effect     string            `json:"effect"`
	Speed      int               `json:"speed"`
	Direction  int               `json:"direction"`
	Colortype  int               `json:"colortype"`
	Device     int               `json:"device"`
	Duration   int               `json:"duration"`
	Subindex   map[string]string `json:"subindex"`
}

// ReadLighting 读取当前的原始键盘灯效参数。
func ReadLighting(service Service) (LightingReading, error) {
	answer, err := service.Get(agentservice.FunctionLighting)
	if err != nil {
		return LightingReading{}, err
	}

	var reading LightingReading
	if err := decode(answer, &reading); err != nil {
		return LightingReading{}, err
	}
	return reading, nil
}

// LightingPayload 构造用于写入原厂服务的灯效字典。
//
// 真机实测：服务会把没写的字段清成 null，
// 所以这里把读到的字段原样写回去，避免丢设置。
func LightingPayload(reading LightingReading) map[string]any {
	payload := map[string]any{
		"brightness": reading.Brightness,
		"effect":     reading.Effect,
		"speed":      reading.Speed,
		"direction":  reading.Direction,
		"colortype":  reading.Colortype,
		"device":     reading.Device,
		"duration":   reading.Duration,
	}

	if reading.Color != "" {
		payload["color"] = reading.Color
	}
	if len(reading.Subindex) > 0 {
		payload["subindex"] = reading.Subindex
	}
	if len(reading.LEDs) > 0 {
		leds := make([]any, 0, len(reading.LEDs))
		for _, led := range reading.LEDs {
			leds = append(leds, map[string]any{
				"LED_id": led.LEDID,
				"color":  led.Color,
				"status": 1,
			})
		}
		payload["LEDs"] = leds
	}
	return payload
}

// WriteLighting 向服务下发原始灯效数据。
func WriteLighting(service Service, reading LightingReading) error {
	answer, err := service.Set(agentservice.FunctionLighting, LightingPayload(reading))
	if err != nil {
		return err
	}
	if err, failed := answer.Err(); failed {
		return err
	}
	return nil
}
