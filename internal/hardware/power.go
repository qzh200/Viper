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
	if _, err := b.callGaming("SetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint64((mode.providerCode() << 8) | uint64(MiscOperationMode)),
	}); err != nil {
		return fmt.Errorf("hardware: set power mode: %w", err)
	}

	current, err := b.PowerMode()
	if err != nil {
		return err
	}
	if current != mode {
		return fmt.Errorf("hardware: the provider kept power mode %s instead of %s", current, mode)
	}
	return nil
}

// DiscreteGPU reports the state of the discrete GPU switch.
//
// The provider encodes the state in the high byte of the setting, the low
// byte is the status of the setting and has to be zero.
//
// 状态字节也一起检查：读取失败时错误码不能被当成"独显已连接"。
func (b *Backend) DiscreteGPU() (bool, error) {
	value, ok, err := b.miscValue(MiscDiscreteGPU)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("hardware: 固件不认识独显状态这个设置，状态字节不是零")
	}
	return value == 3, nil
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
