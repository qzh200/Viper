package probe

import "viper/internal/agentservice"

type LightingLED struct {
	LEDID int    `json:"LED_id"`
	Color string `json:"color"`
}

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
