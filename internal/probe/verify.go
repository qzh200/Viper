package probe

import (
	"encoding/json"
	"errors"

	"viper/internal/agentservice"
	"viper/internal/domain"
	"viper/internal/machine"
)

type Machine interface {
	Instance() string

	Sensors() (domain.Sensors, error)
	Lighting() (domain.RGBConfig, error)
	SetLighting(config domain.RGBConfig) error
	Backlight() (domain.BacklightState, error)
	SetBacklight(brightness int, timeout bool) error
	PowerMode() (domain.PowerMode, error)
	SetPowerMode(mode domain.PowerMode) error
	LCDOverdrive() (bool, error)
	SetLCDOverdrive(enabled bool) error
	BootSound() (bool, error)
	SetBootSound(enabled bool) error
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	Fans() (domain.FanState, error)
	SetFans(config domain.FanConfig) error
}

type StepResult struct {
	Name       string
	Skipped    bool
	ApplyErr   error
	RestoreErr error
}

func (r StepResult) OK() bool {
	return r.ApplyErr == nil && r.RestoreErr == nil && !r.Skipped
}

type Original struct {
	Lighting  domain.RGBConfig
	Backlight domain.BacklightState
	Power     domain.PowerMode
	Overdrive bool
	BootSound bool
	WinKey    bool
	Fans      domain.FanState
}

func Verify(device Machine, raw Service) (Original, []StepResult, error) {
	var original Original

	originalFans, err := raw.Get(agentservice.FunctionFanControl)
	if err != nil {
		return original, nil, err
	}
	var originalFanPayload map[string]any
	if err := json.Unmarshal(originalFans.Data, &originalFanPayload); err != nil {
		return original, nil, err
	}

	original.Lighting, err = device.Lighting()
	if err != nil {
		return original, nil, err
	}
	original.Backlight, err = device.Backlight()
	if err != nil {
		return original, nil, err
	}
	original.Power, err = device.PowerMode()
	if err != nil {
		return original, nil, err
	}
	original.Overdrive, err = device.LCDOverdrive()
	if err != nil {
		return original, nil, err
	}
	original.BootSound, err = device.BootSound()
	if err != nil {
		return original, nil, err
	}
	original.WinKey, err = device.WinKeyLocked()
	if err != nil {
		original.WinKey = false
	}
	original.Fans, err = device.Fans()
	if err != nil {
		return original, nil, err
	}

	lighting := original.Lighting
	backlight := original.Backlight
	power := original.Power
	overdrive := original.Overdrive
	sound := original.BootSound
	winKey := original.WinKey

	zone := lighting
	zone.PerZone = true
	zone.Effect = domain.EffectStatic
	zone.Speed = 3
	zone.Direction = 0
	zone.Brightness = 50
	for i := range zone.Zones {
		zone.Zones[i] = domain.RGBColor{B: 255}
	}

	global := lighting
	global.PerZone = false
	global.Effect = domain.EffectStatic
	global.Brightness = 50
	global.Zones[0] = domain.RGBColor{G: 255}

	steps := []struct {
		name    string
		apply   func() error
		restore func() error
	}{
		{
			name:    "分区颜色",
			apply:   func() error { return device.SetLighting(zone) },
			restore: func() error { return device.SetLighting(lighting) },
		},
		{
			name:    "全局颜色",
			apply:   func() error { return device.SetLighting(global) },
			restore: func() error { return device.SetLighting(lighting) },
		},
		{
			name:    "背光亮度",
			apply:   func() error { return device.SetBacklight(40, backlight.Timeout) },
			restore: func() error { return device.SetBacklight(backlight.Brightness, backlight.Timeout) },
		},
		{
			name:    "背光超时",
			apply:   func() error { return device.SetBacklight(backlight.Brightness, !backlight.Timeout) },
			restore: func() error { return device.SetBacklight(backlight.Brightness, backlight.Timeout) },
		},
		{
			name:    "屏幕超频",
			apply:   func() error { return device.SetLCDOverdrive(!overdrive) },
			restore: func() error { return device.SetLCDOverdrive(overdrive) },
		},
		{
			name:    "开机音效",
			apply:   func() error { return device.SetBootSound(!sound) },
			restore: func() error { return device.SetBootSound(sound) },
		},
		{
			name:    "性能档位",
			apply:   func() error { return device.SetPowerMode(domain.PowerPerformance) },
			restore: func() error { return device.SetPowerMode(power) },
		},
		{
			name:    "Windows 键",
			apply:   func() error { return device.SetWinKeyLocked(!winKey) },
			restore: func() error { return device.SetWinKeyLocked(winKey) },
		},
		{
			name: "风扇全速",
			apply: func() error {
				return device.SetFans(domain.FanConfig{
					CPUMode: domain.FanMax,
					GPUMode: domain.FanMax,
				})
			},
			restore: func() error {
				answer, err := raw.Set(agentservice.FunctionFanControl, originalFanPayload)
				if err != nil {
					return err
				}
				if err, failed := answer.Err(); failed {
					return err
				}
				return nil
			},
		},
	}

	results := make([]StepResult, 0, len(steps))
	for _, item := range steps {
		applyErr := item.apply()
		restoreErr := item.restore()

		result := StepResult{Name: item.name, ApplyErr: applyErr, RestoreErr: restoreErr}
		if errors.Is(applyErr, machine.ErrNoProvider) {
			result.Skipped = true
		}
		results = append(results, result)
	}
	return original, results, nil
}
