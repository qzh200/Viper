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
func (b *Backend) PowerMode() (PowerMode, error) {
	outputs, err := b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(MiscOperationMode)),
	})
	if err != nil {
		return PowerBalanced, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return PowerBalanced, err
	}
	return decodePowerMode((raw >> 8) & 0xFF), nil
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
// The provider encodes the state in the high byte of the setting.
func (b *Backend) DiscreteGPU() (bool, error) {
	outputs, err := b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(MiscDiscreteGPU)),
	})
	if err != nil {
		return false, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, err
	}
	return (raw & 0xFF00) == 0x0300, nil
}

// OverclockCapability reports which overclocking features the machine
// supports.
//
// The provider answers with a non zero value when the feature is present.
func (b *Backend) OverclockCapability() (gpu bool, cpu bool, err error) {
	outputs, err := b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(MiscCPUOverclock)),
	})
	if err != nil {
		return false, false, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, false, err
	}
	cpuValue := (raw >> 8) & 0xFF
	cpu = raw&0xFF == 0 && cpuValue > 0 && cpuValue != 0xFF

	outputs, err = b.callGaming("GetGamingMiscSetting", map[string]wmi.Value{
		"gmInput": wmi.Uint32(uint32(MiscDiscreteGPU)),
	})
	if err != nil {
		return false, cpu, err
	}
	raw, err = decodeUint(outputs["gmOutput"])
	if err != nil {
		return false, cpu, err
	}
	gpuValue := (raw >> 8) & 0xFF
	gpu = raw&0xFF == 0 && gpuValue > 0 && gpuValue != 0xFF

	return gpu, cpu, nil
}
