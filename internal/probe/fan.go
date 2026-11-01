package probe

import "viper/internal/agentservice"

type FanData struct {
	Name  string `json:"fan_name"`
	Auto  int    `json:"fan_custom_auto"`
	Speed int    `json:"fan_custom_speed"`
	Index int    `json:"fan_index"`
}

type FanReading struct {
	Mode int       `json:"mode"`
	Fans []FanData `json:"custom_fan_data"`
}

func ReadFan(service Service) (FanReading, error) {
	answer, err := service.Get(agentservice.FunctionFanControl)
	if err != nil {
		return FanReading{}, err
	}

	var reading FanReading
	if err := decode(answer, &reading); err != nil {
		return FanReading{}, err
	}
	return reading, nil
}

func FanPayload(mode int, cpu, gpu FanData) map[string]any {
	return map[string]any{
		"mode": mode,
		"custom_fan_data": []any{
			map[string]any{
				"fan_custom_auto":  cpu.Auto,
				"fan_custom_speed": cpu.Speed,
				"fan_index":        cpu.Index,
				"fan_name":         cpu.Name,
			},
			map[string]any{
				"fan_custom_auto":  gpu.Auto,
				"fan_custom_speed": gpu.Speed,
				"fan_index":        gpu.Index,
				"fan_name":         gpu.Name,
			},
		},
	}
}

func WriteFan(service Service, mode int, cpu, gpu FanData) error {
	answer, err := service.Set(agentservice.FunctionFanControl, FanPayload(mode, cpu, gpu))
	if err != nil {
		return err
	}
	if err, failed := answer.Err(); failed {
		return err
	}
	return nil
}

func WriteFans(service Service, reading FanReading) error {
	fans := make([]any, 0, len(reading.Fans))
	for _, fan := range reading.Fans {
		fans = append(fans, map[string]any{
			"fan_custom_auto":  fan.Auto,
			"fan_custom_speed": fan.Speed,
			"fan_index":        fan.Index,
			"fan_name":         fan.Name,
		})
	}

	answer, err := service.Set(agentservice.FunctionFanControl, map[string]any{
		"mode":            reading.Mode,
		"custom_fan_data": fans,
	})
	if err != nil {
		return err
	}
	if err, failed := answer.Err(); failed {
		return err
	}
	return nil
}
