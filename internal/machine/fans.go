package machine

import (
	"fmt"
	"strings"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// The modes of the fan configuration of the service.
//
// 实测：模式 0 是自动，模式 1 是全速，模式 2 是固定占空比。写模式 1 之后
// 风扇从 2722 RPM 升到 7092 RPM，写模式 2 并把占空比写成 50 之后风扇落在
// 4400 RPM 左右。模式 0 与模式 2 的区别是每个风扇的 fan_custom_auto 标记，
// 它是零的时候那个风扇按 fan_custom_speed 跑。
const (
	fanModeAuto   = 0
	fanModeMax    = 1
	fanModeCustom = 2
)

// The names of the fan groups of the service.
const (
	fanNameCPU = "CPU"
	fanNameGPU = "GPU"
)

// fanData is the shape of the fan configuration of the service.
type fanData struct {
	Mode          int `json:"mode"`
	CustomFanData []struct {
		Auto  int    `json:"fan_custom_auto"`
		Speed int    `json:"fan_custom_speed"`
		Index int    `json:"fan_index"`
		Name  string `json:"fan_name"`
	} `json:"custom_fan_data"`
}

// Fans reads the fan configuration.
//
// 原厂服务报的是模式与每个风扇的占空比，显卡风扇组的行为它也报得回来，
// 所以这一项比固件那一侧多一样东西。
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

	// 加速标记在原厂服务那边：它的风扇模式 1 就是全速。
	state.CoolBoost = state.CPUMode == hardware.FanMax && state.GPUMode == hardware.FanMax

	return state, nil
}

// SetFans applies a fan configuration.
//
// 两个风扇都是全速的时候写模式 1，其余情况写模式 2：每个风扇的
// fan_custom_auto 标记说它是自动还是按占空比跑，fan_custom_speed 是占空比。
//
// 写完之后把配置读回来核对，固件没有接受哪一个设置就在错误信息里说清楚是
// 哪一个。
func (b *Backend) SetFans(config hardware.FanConfig) error {
	// 加速就是全速：原厂服务的风扇模式 1（实测 2722 RPM 升到 7092 RPM）。固件
	// 那一侧的加速表在本机是死的通道，所以这一项不碰固件。
	if config.CoolBoost {
		config.CPUMode = hardware.FanMax
		config.GPUMode = hardware.FanMax
	}

	mode := fanModeCustom
	cpuAuto, cpuSpeed := fanRequest(config.CPUMode, config.CPUPercent)
	gpuAuto, gpuSpeed := fanRequest(config.GPUMode, config.GPUPercent)
	switch {
	case config.CPUMode == hardware.FanAuto && config.GPUMode == hardware.FanAuto:
		// 两个风扇都是自动：写服务的自动模式（模式 0）。写模式 2 的话，
		// 服务自己的档案里记的是"自定义"（HKLM\SOFTWARE\OEM\
		// AcerAgentService\FanControl\CurrentFanMode），官方应用看到的
		// 也就不是自动了。
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

// verifyFans reads the fan configuration back.
//
// 原厂服务把写下的值原样报回来，所以读回来的配置和写下去的不一样，就说明
// 机器没有接受这个设置。
//
// 全速（模式 1）的时候服务**不看**每个风扇的数据，它只记模式，所以这一项
// 只核对模式。实测写模式 1 之后风扇从 2722 RPM 升到 7092 RPM，而分区数据
// 仍然是原来的自动。
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

// fanRequest turns a behaviour of a fan into the two numbers of the service.
//
// 返回的第一个数字是自动标记，零是按占空比跑，一是让控制器跑自己的曲线。
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

// CoolBoostEnabled reports whether the fans run at full speed.
//
// 加速走**原厂服务**：它的风扇模式 1 就是全速（实测 2722 RPM 升到 7092 RPM）。
// 固件那一侧也有一个加速表，但实测在本机是死的通道——动作类读回来是 0，写了
// 状态也不变，风扇表恒是 1（普通的表）。原厂程序能做的事优先用它。
func (b *Backend) CoolBoostEnabled() (bool, error) {
	state, err := b.Fans()
	if err != nil {
		return false, err
	}
	return state.CPUMode == hardware.FanMax && state.GPUMode == hardware.FanMax, nil
}

// CoolBoost turns the fans to full speed, and hands them back to the
// controller.
//
// 它走原厂服务的风扇模式，理由见 CoolBoostEnabled。
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

// FanTable reads the selected fan table.
//
// 原厂服务不报风扇表，所以这一项走固件那一侧。实测本机的原始答案是 1（普通
// 的表），写不进去。
func (b *Backend) FanTable() (uint64, error) {
	firmware, err := b.provider()
	if err != nil {
		return 0, err
	}
	return firmware.FanTable()
}

// fans reads the fan configuration of the service.
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
