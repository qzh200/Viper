package probe

import "viper/internal/agentservice"

// LightingLED is the colour of one lighting zone.
type LightingLED struct {
	// LEDID is the number of the zone.
	LEDID int `json:"LED_id"`
	// Color is the colour of the zone, "#RRGGBB".
	Color string `json:"color"`
}

// LightingReading is what the service reports about the keyboard lighting.
//
// 全局颜色与分区颜色是两回事：服务读回来有哪一个就写回哪一个。
type LightingReading struct {
	// LEDs are the colours of the zones. The service leaves the list empty
	// when the lighting is set as a whole.
	LEDs []LightingLED `json:"LEDs"`
	// Color is the global colour. The service uses it when there are no
	// zones of its own.
	Color      string `json:"color"`
	Brightness int    `json:"brightness"`
	Effect     string `json:"effect"`
	Speed      int    `json:"speed"`
	Direction  int    `json:"direction"`
}

// ReadLighting asks the service what the lighting looks like.
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

// LightingPayload builds the write that leaves the lighting as it is.
//
// 服务读回来有哪一个就写回哪一个。少写一个，另一个就被服务清掉——写回一个空的
// 分区数组，等于把全局颜色抹成黑色。实测过：这一条以前让「写回原值」把键盘的
// 颜色清掉了。
func LightingPayload(reading LightingReading) map[string]any {
	payload := map[string]any{
		"brightness": reading.Brightness,
		"effect":     reading.Effect,
		"speed":      reading.Speed,
		"direction":  reading.Direction,
	}

	if reading.Color != "" {
		payload["color"] = reading.Color
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

// WriteLighting writes the lighting back the way the service reported it.
//
// 它证明写通路是通的，机器的状态不变。
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
