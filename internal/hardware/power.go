package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// PowerMode reads the current performance profile.
//
// The provider packs the profile into the second byte of the setting, so
// the full byte has to be read: six and seven are the two performance
// profiles and a narrow mask would collapse them.
//
// 设置的最低字节是状态，只有状态为零时里面的值才是有效的。以前这里不看
// 状态，读取失败时错误码 0x0002 会被解成 (2>>8)=0，也就是"静音模式"，
// 而且还不报错。
func (b *Backend) PowerMode() (PowerMode, error) {
	value, ok, err := b.miscValue(MiscOperationMode)
	if err != nil {
		return PowerBalanced, err
	}
	if !ok {
		return PowerBalanced, fmt.Errorf("hardware: 固件不认识性能档位这个设置，状态字节不是零")
	}
	return decodePowerMode(value), nil
}

// SetPowerMode applies a performance profile.
//
// Turbo is only available on Predator models, the provider rejects it
// elsewhere. The output parameter of the setter echoes the new value rather
// than a status code, so the result is confirmed by reading it back.
func (b *Backend) SetPowerMode(mode PowerMode) error {
	_, err := operate(b, func() (struct{}, error) {
		if _, err := b.callGaming("SetGamingMiscSetting", map[string]wmi.Value{
			"gmInput": wmi.Uint64((mode.providerCode() << 8) | uint64(MiscOperationMode)),
		}); err != nil {
			return struct{}{}, fmt.Errorf("hardware: set power mode: %w", err)
		}

		current, err := b.PowerMode()
		if err != nil {
			return struct{}{}, err
		}
		if current != mode {
			return struct{}{}, fmt.Errorf("hardware: the provider kept power mode %s instead of %s", current, mode)
		}
		return struct{}{}, nil
	})
	return err
}

// DiscreteGPU reports the state of the discrete GPU switch.
//
// 开关的状态在杂项表的 0x02（显卡模式）里：混合、独显、自动。0x09 那一项
// 不是开关的状态，它是这台机器支持显卡模式的能力值，真机上读到的是 7，
// 与原厂服务能力表 SUPPORT_GPU_MODE_CAPABILITY 里的 7 相同。
//
// 只有模式明确是独显时才报开。自动模式下由系统决定，不能从设置里读出来。
func (b *Backend) DiscreteGPU() (bool, error) {
	mode, err := b.GPUMode()
	if err != nil {
		return false, err
	}
	return mode == GPUModeDiscrete, nil
}

// OverclockCapability reports which overclocking features the machine
// supports.
//
// 杂项设置表里只有 CPU 的超频能力（0x0A），固件没有给出显卡的超频能力，
// 所以显卡那一项只能报成不支持。
//
// 以前这里读的是独显状态（0x09），那是"这台机器上有没有独显"，和"显卡
// 能不能超频"是两回事：只要机器上有独显，就恒报支持显卡超频。
func (b *Backend) OverclockCapability() (gpu bool, cpu bool, err error) {
	value, ok, err := b.miscValue(MiscCPUOverclock)
	if err != nil {
		return false, false, err
	}
	if !ok {
		// 固件不认识这个设置，那就不能说这台机器能超频。
		return false, false, nil
	}
	// 固件用 0xFF 表示"不知道"。
	cpu = value > 0 && value != 0xFF
	return false, cpu, nil
}
