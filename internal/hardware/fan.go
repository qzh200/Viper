package hardware

import (
	"fmt"
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

// applyFans writes the fan configuration without holding the lock.
//
// 锁加在公开的那一层，内部的辅助方法不再加锁，所以一个操作里可以调用另一个
// 操作而不会自己等自己。

// verifyFans reads the fan configuration back and compares it with the one
// that was written.
//
// 固件只报告主风扇组（CPU）的行为，显卡风扇组的行为读不回来，所以显卡风扇
// 的行为不做核对，只核对它的占空比。
//
// 读不回来的那一个（provider 没有实现这个读取方法）就跳过核对，因为没有
// 可以比较的值；能读回来却不一致的设置会报错，并且说清是哪一个设置。

// verifyFanSpeed reads the duty cycle of one fan back and compares it with
// the one that was written.

// Fans reads the current fan configuration.
//
// The provider only reports the behaviour of the primary group, so the
// group of the graphics fan cannot be read back. Its duty cycle can.

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
