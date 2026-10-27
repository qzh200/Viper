package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// Fan identifiers used by SetGamingFanSpeed.
const (
	// FanIndexCPU addresses the CPU fan.
	FanIndexCPU = 1
	// FanIndexGPUPrimary addresses the first GPU fan.
	FanIndexGPUPrimary = 4
	// FanIndexGPUSecondary addresses the second GPU fan on dual fan
	// models.
	FanIndexGPUSecondary = 2
)

// Fan tables of the embedded controller.
const (
	// FanTableNormal is the regular performance table.
	FanTableNormal uint64 = 1
	// FanTableBoost is the high performance table used by CoolBoost and
	// by maximum fan mode.
	FanTableBoost uint64 = 3
)

// FanState describes the current fan configuration.
type FanState struct {
	// CPUMode is the behaviour of the CPU fan.
	CPUMode FanMode
	// GPUMode is the behaviour of the GPU fan.
	//
	// 固件只报告主风扇组（CPU）的行为，显卡风扇组的行为读不回来，所以这个
	// 字段只有在 GPUModeReadable 为真的时候才有意义。
	GPUMode FanMode
	// GPUModeReadable reports whether the behaviour of the graphics fan was
	// read back from the firmware. 这个 provider 只报告主风扇组，所以它是
	// 假的；上层必须自己区分"是自动"和"读不到"。
	GPUModeReadable bool
	// CPUPercent is the duty cycle the CPU fan was last configured with.
	//
	// In automatic mode the embedded controller runs its own curve, so the
	// value is only the last requested setting and not the current speed.
	CPUPercent int
	// GPUPercent is the duty cycle the GPU fan was last configured with.
	GPUPercent int
	// CoolBoost reports whether the boost table is selected.
	CoolBoost bool
}

// FanConfig describes a requested fan configuration.
type FanConfig struct {
	// CPUMode is the requested behaviour of the CPU fan.
	CPUMode FanMode
	// GPUMode is the requested behaviour of the GPU fan.
	GPUMode FanMode
	// CPUPercent is the requested duty cycle of the CPU fan.
	CPUPercent int
	// GPUPercent is the requested duty cycle of the GPU fan.
	GPUPercent int
	// CoolBoost requests the boost table.
	CoolBoost bool
}

// SetFans applies a fan configuration.
//
// The embedded controller wants the behaviour of both fans in one word, so
// the modes are combined first, then the duty cycles and the fan table are
// applied.
//
// 写完之后把行为、占空比和风扇表读回来核对，固件没有接受哪一个设置就在错误
// 信息里说清楚是哪一个。
func (b *Backend) SetFans(config FanConfig) error {
	cpuPercent := clampPercent(config.CPUPercent)
	gpuPercent := clampPercent(config.GPUPercent)

	behavior := uint64(9) |
		(config.CPUMode.providerCode() << 16) |
		(config.GPUMode.providerCode() << 22)

	if _, err := b.callGaming("SetGamingFanBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint64(behavior),
	}); err != nil {
		return fmt.Errorf("hardware: 设置风扇行为失败: %w", err)
	}

	// The duty cycle that is written for the CPU fan. A value of zero means
	// that the controller keeps its own curve.
	cpuSpeed := 0
	switch config.CPUMode {
	case FanMax:
		cpuSpeed = 100
	case FanCustom:
		cpuSpeed = cpuPercent
	}

	if cpuSpeed > 0 {
		if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
			"gmInput": wmi.Uint64(uint64(cpuSpeed)<<8 | FanIndexCPU),
		}); err != nil {
			return fmt.Errorf("hardware: 设置 CPU 风扇转速失败: %w", err)
		}
	} else {
		// Hand the duty cycle back to the controller, otherwise the last
		// requested value keeps running even in automatic mode.
		if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
			"gmInput": wmi.Uint64(FanIndexCPU),
		}); err != nil {
			return fmt.Errorf("hardware: 交还 CPU 风扇的占空比失败: %w", err)
		}
	}

	// The same for the fans of the graphics card.
	gpuSpeed := 0
	switch config.GPUMode {
	case FanMax:
		gpuSpeed = 100
	case FanCustom:
		gpuSpeed = gpuPercent
	}

	if gpuSpeed > 0 {
		for _, index := range []uint64{FanIndexGPUPrimary, FanIndexGPUSecondary} {
			if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
				"gmInput": wmi.Uint64(uint64(gpuSpeed)<<8 | index),
			}); err != nil {
				return fmt.Errorf("hardware: 设置显卡风扇 %d 的转速失败: %w", index, err)
			}
		}
	}

	table := FanTableNormal
	if config.CoolBoost || (config.CPUMode == FanMax && config.GPUMode == FanMax) {
		table = FanTableBoost
	}
	if _, err := b.callGaming("SetGamingFanTable", map[string]wmi.Value{
		"gmInput": wmi.Uint64(table),
	}); err != nil {
		return fmt.Errorf("hardware: 设置风扇表失败: %w", err)
	}

	return b.verifyFans(config, cpuSpeed, gpuSpeed, table)
}

// verifyFans reads the fan configuration back and compares it with the one
// that was written.
//
// 固件只报告主风扇组（CPU）的行为，显卡风扇组的行为读不回来，所以显卡风扇
// 的行为不做核对，只核对它的占空比。
//
// 读不回来的那一个（provider 没有实现这个读取方法）就跳过核对，因为没有
// 可以比较的值；能读回来却不一致的设置会报错，并且说清是哪一个设置。
func (b *Backend) verifyFans(config FanConfig, cpuSpeed, gpuSpeed int, table uint64) error {
	outputs, err := b.callGaming("GetGamingFanBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err == nil {
		value, ok := outputs["gmOutput"]
		if !ok {
			return fmt.Errorf("hardware: 读回风扇行为时固件没有给出 gmOutput")
		}
		raw, decodeErr := decodeUint(value)
		if decodeErr != nil {
			return fmt.Errorf("hardware: 读回风扇行为失败: %w", decodeErr)
		}
		// 最低字节是状态，状态不是零时这个答案说的不是风扇的行为，没有可以
		// 比较的值，跳过核对。
		if raw&0xFF == 0 {
			// 固件把行为编码乘以 256 放在第二个字节里。
			mode := decodeFanMode((raw >> 8) & 0x3)
			if mode != config.CPUMode {
				return fmt.Errorf("hardware: 固件把 CPU 风扇的行为保持在 %s，没有变成 %s", mode, config.CPUMode)
			}
		}
	}

	if cpuSpeed > 0 {
		if err := b.verifyFanSpeed(FanIndexCPU, cpuSpeed); err != nil {
			return err
		}
	}
	if gpuSpeed > 0 {
		for _, index := range []uint64{FanIndexGPUPrimary, FanIndexGPUSecondary} {
			if err := b.verifyFanSpeed(index, gpuSpeed); err != nil {
				return err
			}
		}
	}

	outputs, err = b.callGaming("GetGamingFanTable", nil)
	if err == nil {
		value, ok := outputs["gmOutput"]
		if !ok {
			return fmt.Errorf("hardware: 读回风扇表时固件没有给出 gmOutput")
		}
		raw, decodeErr := decodeUint(value)
		if decodeErr != nil {
			return fmt.Errorf("hardware: 读回风扇表失败: %w", decodeErr)
		}
		if raw&0xFF == 0 {
			current := (raw >> 8) & 0xFF
			if current != table {
				return fmt.Errorf("hardware: 固件把风扇表保持在 %d，没有变成 %d", current, table)
			}
		}
	}

	return nil
}

// verifyFanSpeed reads the duty cycle of one fan back and compares it with
// the one that was written.
func (b *Backend) verifyFanSpeed(index uint64, want int) error {
	speed, err := b.fanSpeed(index)
	if err != nil {
		// 没有可以比较的值，跳过核对。
		return nil
	}
	if speed != want {
		return fmt.Errorf("hardware: 固件把风扇 %d 的占空比保持在 %d%%，没有变成 %d%%", index, speed, want)
	}
	return nil
}

// Fans reads the current fan configuration.
//
// The provider only reports the behaviour of the primary group, so the
// group of the graphics fan cannot be read back. Its duty cycle can.
func (b *Backend) Fans() (FanState, error) {
	var state FanState

	outputs, err := b.callGaming("GetGamingFanBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint32(1),
	})
	if err != nil {
		return state, err
	}
	// 固件没有给出 gmOutput 时不能当成读到了零，那会让上层把"读不到"当成
	// "风扇在自动模式"。
	value, ok := outputs["gmOutput"]
	if !ok {
		return state, fmt.Errorf("hardware: 读取风扇行为时固件没有给出 gmOutput")
	}
	raw, err := decodeUint(value)
	if err != nil {
		return state, err
	}
	// 固件把行为编码乘以 256 放在答案的第二个字节里，也就是位 8 到位 15。
	state.CPUMode = decodeFanMode((raw >> 8) & 0x3)
	// 显卡风扇组的行为读不回来，这个字段没有值，所以它不算可读。
	state.GPUModeReadable = false

	if speed, err := b.fanSpeed(FanIndexCPU); err == nil {
		state.CPUPercent = speed
	}
	if speed, err := b.fanSpeed(FanIndexGPUPrimary); err == nil {
		state.GPUPercent = speed
	}

	state.CoolBoost, err = b.CoolBoostEnabled()
	if err != nil {
		return state, err
	}
	return state, nil
}

// FanTable reads the selected fan table.
//
// 这是最原始的读取，调用者拿到的就是固件里的编号。普通代码应该用
// CoolBoostEnabled，它先把动作类的答案和风扇表合起来看。
func (b *Backend) FanTable() (uint64, error) {
	outputs, err := b.callGaming("GetGamingFanTable", nil)
	if err != nil {
		return 0, err
	}
	// 固件没有给出 gmOutput 时不能当成读到了零，那会让上层把"读不到"当成
	// "用的是普通的表"。
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: 读取风扇表时固件没有给出 gmOutput")
	}
	raw, err := decodeUint(value)
	if err != nil {
		return 0, err
	}
	return (raw >> 8) & 0xFF, nil
}

// fanSpeed reads the duty cycle of one fan.
//
// The lowest byte of the answer is the status of the reading, a status that
// is not zero means that the provider did not answer this fan.
func (b *Backend) fanSpeed(index uint64) (int, error) {
	outputs, err := b.callGaming("GetGamingFanSpeed", map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(index)),
	})
	if err != nil {
		return 0, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return 0, err
	}
	if status := raw & 0xFF; status != 0 {
		return 0, fmt.Errorf("hardware: 固件没有回答风扇 %d 的转速，状态字节是 %d", index, status)
	}
	return int((raw >> 8) & 0xFF), nil
}

// clampPercent keeps a duty cycle inside the range the EC accepts.
func clampPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
