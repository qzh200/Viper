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
	GPUMode FanMode
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
func (b *Backend) SetFans(config FanConfig) error {
	cpuPercent := clampPercent(config.CPUPercent)
	gpuPercent := clampPercent(config.GPUPercent)

	behavior := uint64(9) |
		(config.CPUMode.providerCode() << 16) |
		(config.GPUMode.providerCode() << 22)

	if _, err := b.callGaming("SetGamingFanBehavior", map[string]wmi.Value{
		"gmInput": wmi.Uint64(behavior),
	}); err != nil {
		return fmt.Errorf("hardware: set fan behaviour: %w", err)
	}

	if config.CPUMode == FanCustom || config.CPUMode == FanMax {
		speed := uint64(cpuPercent)
		if config.CPUMode == FanMax {
			speed = 100
		}
		if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
			"gmInput": wmi.Uint64((speed << 8) | FanIndexCPU),
		}); err != nil {
			return fmt.Errorf("hardware: set CPU fan speed: %w", err)
		}
	} else {
		// Hand the duty cycle back to the controller, otherwise the last
		// requested value keeps running even in automatic mode.
		if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
			"gmInput": wmi.Uint64(FanIndexCPU),
		}); err != nil {
			return fmt.Errorf("hardware: reset CPU fan speed: %w", err)
		}
	}

	if config.GPUMode == FanCustom || config.GPUMode == FanMax {
		speed := uint64(gpuPercent)
		if config.GPUMode == FanMax {
			speed = 100
		}
		for _, index := range []uint64{FanIndexGPUPrimary, FanIndexGPUSecondary} {
			if _, err := b.callGaming("SetGamingFanSpeed", map[string]wmi.Value{
				"gmInput": wmi.Uint64((speed << 8) | index),
			}); err != nil {
				return fmt.Errorf("hardware: set GPU fan speed: %w", err)
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
		return fmt.Errorf("hardware: set fan table: %w", err)
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
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return state, err
	}
	// The behaviour is reported in the third byte.
	state.CPUMode = decodeFanMode((raw >> 8) & 0x3)

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
func (b *Backend) FanTable() (uint64, error) {
	outputs, err := b.callGaming("GetGamingFanTable", nil)
	if err != nil {
		return 0, err
	}
	raw, err := decodeUint(outputs["gmOutput"])
	if err != nil {
		return 0, err
	}
	return (raw >> 8) & 0xFF, nil
}

// fanSpeed reads the duty cycle of one fan.
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
