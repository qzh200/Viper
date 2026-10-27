package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// MiscSettingValue reads one of the provider's miscellaneous settings.
//
// The provider answers with a single word. The upper byte carries the
// value of the setting, the lower byte is a status that is zero when the
// setting is defined.
//
// 固件没有给出 gmOutput 时不能当成读到了零：那会让上层把"读不到"当成
// "值是零"，性能档位会变成静音模式，开机音效会变成关着的，而且都不报错。
func (b *Backend) MiscSettingValue(id uint32) (uint64, error) {
	outputs, err := b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(id),
	})
	if err != nil {
		return 0, err
	}
	value, ok := outputs["gmOutput"]
	if !ok {
		return 0, fmt.Errorf("hardware: 读取杂项设置 0x%02X 时固件没有给出 gmOutput", id)
	}
	raw, err := decodeUint(value)
	if err != nil {
		return 0, err
	}
	return raw, nil
}

// SetMiscSettingValue writes one of the provider's miscellaneous settings.
//
// The value and the identifier travel in one word, the value in the upper
// byte and the identifier in the lower one. The provider answers with the
// new value rather than with a status, so the result is confirmed by
// reading the setting back.
//
// 读回的时候要先看状态字节：固件不认识这个设置时会答一个非零的状态，
// 而值恰好是零的时候，只看值就会把"固件拒绝了"当成"写成功了"。
func (b *Backend) SetMiscSettingValue(id uint32, value uint64) error {
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.applyMiscSettingValue(id, value)
	})
	return err
}

// applyMiscSettingValue writes the setting without holding the lock.
//
// 锁加在公开的那一层，内部的辅助方法不再加锁，所以一个操作里可以调用另一个
// 操作而不会自己等自己。
func (b *Backend) applyMiscSettingValue(id uint32, value uint64) error {
	input := (value << 8) | uint64(id)
	if _, err := b.callGaming("SetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint64(input),
	}); err != nil {
		return fmt.Errorf("hardware: set the miscellaneous setting 0x%02X: %w", id, err)
	}

	raw, err := b.MiscSettingValue(id)
	if err != nil {
		return err
	}
	if status := raw & 0xFF; status != 0 {
		return fmt.Errorf("hardware: 固件不认识杂项设置 0x%02X，状态字节是 %d", id, status)
	}
	if raw>>8 != value {
		return fmt.Errorf("hardware: the provider kept the setting 0x%02X at %d instead of %d", id, raw>>8, value)
	}
	return nil
}

// miscValue reads the value of a miscellaneous setting.
//
// A setting that the firmware does not know answers with an error code in
// the lower byte, such a setting is reported as not present.
func (b *Backend) miscValue(id MiscSetting) (uint64, bool, error) {
	raw, err := b.MiscSettingValue(uint32(id))
	if err != nil {
		return 0, false, err
	}
	if raw&0xFF != 0 {
		// The lower byte carries a status, a value of two means that the
		// firmware does not know the setting.
		return 0, false, nil
	}
	return raw >> 8, true, nil
}

// GPUMode reads the graphics switch of the machine.
//
// The value is the one of the graphics mode, the same word that the
// original service reports as the display mode.
func (b *Backend) GPUMode() (GPUMode, error) {
	value, ok, err := b.miscValue(MiscGPUMode)
	if err != nil {
		return GPUModeHybrid, err
	}
	if !ok {
		return GPUModeHybrid, fmt.Errorf("hardware: the firmware does not know the graphics mode")
	}
	return GPUMode(value), nil
}

// SetGPUMode selects the graphics switch of the machine.
//
// The change is taken over when the machine restarts, the official
// application asks for the same restart.
func (b *Backend) SetGPUMode(mode GPUMode) error {
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.applyMiscSettingValue(uint32(MiscGPUMode), uint64(mode))
	})
	return err
}

// BootSound reports whether the sound at the start of the machine is on.
//
// The setting is a plain switch in the miscellaneous table of the
// provider, the original service reads it under the name of the boot
// sound.
func (b *Backend) BootSound() (bool, error) {
	value, ok, err := b.miscValue(MiscBootSound)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("hardware: the firmware does not know the boot sound")
	}
	return value != 0, nil
}

// SetBootSound switches the sound at the start of the machine on or off.
func (b *Backend) SetBootSound(enabled bool) error {
	value := uint64(0)
	if enabled {
		value = 1
	}
	_, err := operate(b, func() (struct{}, error) {
		return struct{}{}, b.applyMiscSettingValue(uint32(MiscBootSound), value)
	})
	return err
}
