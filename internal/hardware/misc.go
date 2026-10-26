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
func (b *Backend) MiscSettingValue(id uint32) (uint64, error) {
	outputs, err := b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(id),
	})
	if err != nil {
		return 0, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
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
func (b *Backend) SetMiscSettingValue(id uint32, value uint64) error {
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

// GPUMode reads the graphics mode of the machine.
func (b *Backend) GPUMode() (GPUMode, error) {
	value, ok, err := b.miscValue(MiscGPUMode)
	if err != nil {
		return GPUModeOptimus, err
	}
	if !ok {
		return GPUModeOptimus, fmt.Errorf("hardware: the firmware does not know the graphics mode")
	}
	return GPUMode(value), nil
}

// SetGPUMode selects the graphics mode of the machine.
//
// The change is taken over when the machine restarts, the official
// application asks for the same restart.
func (b *Backend) SetGPUMode(mode GPUMode) error {
	return b.SetMiscSettingValue(uint32(MiscGPUMode), uint64(mode))
}

// BluelightShield reads the blue light shield.
//
// The shield is not a plain switch, the official application offers six
// levels from disabled to the long hour mode.
func (b *Backend) BluelightShield() (uint64, bool, error) {
	return b.miscValue(MiscBluelightShield)
}

// SetBluelightShield selects a level of the blue light shield.
func (b *Backend) SetBluelightShield(level uint64) error {
	return b.SetMiscSettingValue(uint32(MiscBluelightShield), level)
}

// USBChargeLimit reads the charging limit in percent.
func (b *Backend) USBChargeLimit() (uint64, bool, error) {
	return b.miscValue(MiscUSBCharge)
}

// SetUSBChargeLimit sets the charging limit in percent.
func (b *Backend) SetUSBChargeLimit(percent uint64) error {
	return b.SetMiscSettingValue(uint32(MiscUSBCharge), percent)
}

// USBChargeEnabled reports whether the charging over USB is allowed.
func (b *Backend) USBChargeEnabled() (bool, bool, error) {
	value, ok, err := b.miscValue(MiscUSBChargeSwitch)
	if err != nil {
		return false, false, err
	}
	return value != 0, ok, nil
}

// SetUSBChargeEnabled allows or forbids the charging over USB.
func (b *Backend) SetUSBChargeEnabled(enabled bool) error {
	value := uint64(0)
	if enabled {
		value = 1
	}
	return b.SetMiscSettingValue(uint32(MiscUSBChargeSwitch), value)
}
