package guitest

import (
	"errors"

	"viper/internal/domain"
	"viper/internal/gui/controller"
)

var ErrHardware = errors.New("guitest: 机器没有回应")

type Hardware struct {
	MonitorData           domain.MonitorData
	Power                 domain.PowerMode
	FansState             domain.FanState
	LightingConfig        domain.RGBConfig
	BacklightTimeoutState bool
	GPUModeState          domain.GPUMode
	DiscreteGPUState      bool
	LCDOverdriveState     bool
	WinKeyLockedState     bool
	BootSoundState        bool
	InstanceName          string
	RouteName             string
	Err                   error
	SetErr                error

	Writes             []domain.PowerMode
	FanWrites          []domain.FanConfig
	CoolBoostWrites    []bool
	LightWrites        []domain.RGBConfig
	ResetLightCalls    int
	TimeoutWrites      []bool
	GPUModeWrites      []domain.GPUMode
	LCDOverdriveWrites []bool
	WinKeyLockedWrites []bool
	BootSoundWrites    []bool
}

func (h *Hardware) Interface() controller.Hardware { return h }

func (h *Hardware) Monitor() (domain.MonitorData, error) {
	if h.Err != nil {
		return domain.MonitorData{}, h.Err
	}
	return h.MonitorData, nil
}

func (h *Hardware) PowerMode() (domain.PowerMode, error) {
	if h.Err != nil {
		return domain.PowerBalanced, h.Err
	}
	return h.Power, nil
}

func (h *Hardware) SetPowerMode(mode domain.PowerMode) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.Writes = append(h.Writes, mode)
	h.Power = mode
	return nil
}

func (h *Hardware) Route() string { return h.RouteName }

func (h *Hardware) FanState() (domain.FanState, error) {
	if h.Err != nil {
		return domain.FanState{}, h.Err
	}
	return h.FansState, nil
}

func (h *Hardware) SetFans(config domain.FanConfig) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.FanWrites = append(h.FanWrites, config)
	h.FansState.CPUMode = config.CPUMode
	h.FansState.GPUMode = config.GPUMode
	h.FansState.CPUPercent = config.CPUPercent
	h.FansState.GPUPercent = config.GPUPercent
	h.FansState.CoolBoost = config.CoolBoost
	return nil
}

func (h *Hardware) SetCoolBoost(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.CoolBoostWrites = append(h.CoolBoostWrites, enabled)
	h.FansState.CoolBoost = enabled
	return nil
}

func (h *Hardware) Lighting() (domain.RGBConfig, error) {
	if h.Err != nil {
		return domain.RGBConfig{}, h.Err
	}
	return h.LightingConfig, nil
}

func (h *Hardware) SetLighting(config domain.RGBConfig) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.LightWrites = append(h.LightWrites, config)
	h.LightingConfig = config
	return nil
}

func (h *Hardware) ResetLighting() error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.ResetLightCalls++
	h.LightingConfig = domain.RGBConfig{
		Effect:     domain.EffectStatic,
		Speed:      3,
		Brightness: 100,
		PerZone:    true,
	}
	for i := range h.LightingConfig.Zones {
		h.LightingConfig.Zones[i] = domain.RGBColor{R: 255, G: 255, B: 255}
	}
	return nil
}

func (h *Hardware) BacklightTimeout() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.BacklightTimeoutState, nil
}

func (h *Hardware) SetBacklightTimeout(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.TimeoutWrites = append(h.TimeoutWrites, enabled)
	h.BacklightTimeoutState = enabled
	return nil
}

func (h *Hardware) GPUMode() (domain.GPUMode, error) {
	if h.Err != nil {
		return domain.GPUModeHybrid, h.Err
	}
	return h.GPUModeState, nil
}

func (h *Hardware) SetGPUMode(mode domain.GPUMode) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.GPUModeWrites = append(h.GPUModeWrites, mode)
	h.GPUModeState = mode
	return nil
}

func (h *Hardware) DiscreteGPU() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.DiscreteGPUState, nil
}

func (h *Hardware) LCDOverdrive() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.LCDOverdriveState, nil
}

func (h *Hardware) SetLCDOverdrive(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.LCDOverdriveWrites = append(h.LCDOverdriveWrites, enabled)
	h.LCDOverdriveState = enabled
	return nil
}

func (h *Hardware) WinKeyLocked() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.WinKeyLockedState, nil
}

func (h *Hardware) SetWinKeyLocked(locked bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.WinKeyLockedWrites = append(h.WinKeyLockedWrites, locked)
	h.WinKeyLockedState = locked
	return nil
}

func (h *Hardware) BootSound() (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.BootSoundState, nil
}

func (h *Hardware) SetBootSound(enabled bool) error {
	if h.SetErr != nil {
		return h.SetErr
	}
	h.BootSoundWrites = append(h.BootSoundWrites, enabled)
	h.BootSoundState = enabled
	return nil
}

func (h *Hardware) Instance() (string, error) {
	if h.Err != nil {
		return "", h.Err
	}
	return h.InstanceName, nil
}

type Store struct {
	Mode  string
	Err   error
	Saved []string
}

func (s *Store) Interface() controller.AppearanceStore { return s }

func (s *Store) Theme() string { return s.Mode }

func (s *Store) SetTheme(mode string) error {
	if s.Err != nil {
		return s.Err
	}
	s.Mode = mode
	s.Saved = append(s.Saved, mode)
	return nil
}
