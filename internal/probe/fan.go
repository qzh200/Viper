package probe

import "viper/internal/agentservice"

// FanData is the behaviour of one fan.
type FanData struct {
	// Name is the name of the fan, "CPU" or "GPU".
	Name string `json:"fan_name"`
	// Auto says whether the embedded controller runs its own curve. One
	// means it does, zero means the duty cycle below is used.
	Auto int `json:"fan_custom_auto"`
	// Speed is the duty cycle for the case where the controller does not
	// run its own curve.
	Speed int `json:"fan_custom_speed"`
	// Index is the number of the fan.
	Index int `json:"fan_index"`
}

// FanReading is what the service reports about the fans.
type FanReading struct {
	// Mode is the behaviour of the fans as a whole.
	//
	// 顶层 mode 是开关：0 是自动，1 是全速，2 是固定占空比。
	Mode int `json:"mode"`
	// Fans are the fans themselves.
	Fans []FanData `json:"custom_fan_data"`
}

// ReadFan asks the service how the fans are set.
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

// FanPayload builds the write of the fan configuration.
//
// 两个风扇各带一份行为与占空比，顶层的 mode 是它们的总开关。
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

// WriteFan writes the fan configuration.
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

// WriteFans writes the fan configuration back the way the service reported it.
//
// 每一个风扇都按服务记着的样子写回去，所以机器的状态不变。
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
