package machine

import (
	"fmt"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

const (
	fanModeAuto   = 0
	fanModeMax    = 1
	fanModeCustom = 2
)

const (
	fanNameCPU = "CPU"
	fanNameGPU = "GPU"
)

type fanData struct {
	Mode          int `json:"mode"`
	CustomFanData []struct {
		Auto  int    `json:"fan_custom_auto"`
		Speed int    `json:"fan_custom_speed"`
		Index int    `json:"fan_index"`
		Name  string `json:"fan_name"`
	} `json:"custom_fan_data"`
}

// Fans 读取当前 CPU 与 GPU 风扇的运转模式与转速百分比。
func (b *Backend) Fans() (hardware.FanState, error) {
	data, err := b.fans()
	if err != nil {
		return hardware.FanState{}, err
	}

	state := hardware.FanState{
		GPUModeReadable: true,
	}

	switch data.Mode {
	case fanModeMax:
		state.CPUMode = hardware.FanMax
		state.GPUMode = hardware.FanMax
	case fanModeCustom, fanModeAuto:
		state.CPUMode = hardware.FanAuto
		state.GPUMode = hardware.FanAuto
		for _, fan := range data.CustomFanData {
			mode := hardware.FanAuto
			if fan.Auto == 0 {
				mode = hardware.FanCustom
			}
			switch strings.ToUpper(fan.Name) {
			case fanNameCPU:
				state.CPUMode = mode
				state.CPUPercent = fan.Speed
			case fanNameGPU:
				state.GPUMode = mode
				state.GPUPercent = fan.Speed
			}
		}
	default:
		return hardware.FanState{}, fmt.Errorf("machine: 原厂服务报了不认识的风扇模式 %d", data.Mode)
	}

	state.CoolBoost = state.CPUMode == hardware.FanMax && state.GPUMode == hardware.FanMax

	return state, nil
}

// SetFans 配置并下发风扇的目标运转模式及转速参数。
func (b *Backend) SetFans(config hardware.FanConfig) error {
	if config.CoolBoost {
		config.CPUMode = hardware.FanMax
		config.GPUMode = hardware.FanMax
	}

	mode := fanModeCustom
	cpuAuto, cpuSpeed := fanRequest(config.CPUMode, config.CPUPercent)
	gpuAuto, gpuSpeed := fanRequest(config.GPUMode, config.GPUPercent)
	switch {
	case config.CPUMode == hardware.FanAuto && config.GPUMode == hardware.FanAuto:
		mode = fanModeAuto
	case config.CPUMode == hardware.FanMax && config.GPUMode == hardware.FanMax:
		mode = fanModeMax
	}

	payload := map[string]any{
		"mode": mode,
		"custom_fan_data": []any{
			map[string]any{
				"fan_custom_auto":  cpuAuto,
				"fan_custom_speed": cpuSpeed,
				"fan_index":        0,
				"fan_name":         fanNameCPU,
			},
			map[string]any{
				"fan_custom_auto":  gpuAuto,
				"fan_custom_speed": gpuSpeed,
				"fan_index":        0,
				"fan_name":         fanNameGPU,
			},
		},
	}

	if err := b.set(agentservice.FunctionFanControl, payload); err != nil {
		return err
	}

	return b.verifyFans(mode, cpuAuto, cpuSpeed, gpuAuto, gpuSpeed)
}

func (b *Backend) verifyFans(mode int, cpuAuto int, cpuSpeed int, gpuAuto int, gpuSpeed int) error {
	data, err := b.fans()
	if err != nil {
		return err
	}
	if data.Mode != mode {
		return fmt.Errorf("machine: 风扇模式没有生效，服务报的是 %d，不是 %d", data.Mode, mode)
	}
	if mode == fanModeMax {
		return nil
	}

	for _, fan := range data.CustomFanData {
		auto, speed := cpuAuto, cpuSpeed
		switch strings.ToUpper(fan.Name) {
		case fanNameGPU:
			auto, speed = gpuAuto, gpuSpeed
		case fanNameCPU:
		default:
			continue
		}

		if fan.Auto != auto {
			return fmt.Errorf("machine: %s 风扇的行为没有生效，服务报的是 %d，不是 %d",
				fan.Name, fan.Auto, auto)
		}
		if auto == 0 && fan.Speed != speed {
			return fmt.Errorf("machine: %s 风扇的占空比没有生效，服务报的是 %d，不是 %d",
				fan.Name, fan.Speed, speed)
		}
	}

	return nil
}

func fanRequest(mode hardware.FanMode, percent int) (auto int, speed int) {
	switch mode {
	case hardware.FanMax:
		return 0, 100
	case hardware.FanCustom:
		return 0, int(clamp(uint64(percent), 0, 100))
	default:
		return 1, 0
	}
}

// CoolBoostEnabled 查询 CoolBoost 强冷是否开启（两风扇处于最大转速）。
func (b *Backend) CoolBoostEnabled() (bool, error) {
	state, err := b.Fans()
	if err != nil {
		return false, err
	}
	return state.CPUMode == hardware.FanMax && state.GPUMode == hardware.FanMax, nil
}

// CoolBoost 启用或禁用 CoolBoost 强冷功能。
func (b *Backend) CoolBoost(enabled bool) error {
	if enabled {
		return b.SetFans(hardware.FanConfig{
			CPUMode: hardware.FanMax,
			GPUMode: hardware.FanMax,
		})
	}
	return b.SetFans(hardware.FanConfig{
		CPUMode: hardware.FanAuto,
		GPUMode: hardware.FanAuto,
	})
}

func (b *Backend) fans() (fanData, error) {
	var data fanData

	answer, err := b.get(agentservice.FunctionFanControl)
	if err != nil {
		return data, err
	}
	if err := answer.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}
