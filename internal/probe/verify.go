package probe

import (
	"encoding/json"
	"errors"

	"viper/internal/agentservice"
	"viper/internal/hardware"
	"viper/internal/machine"
)

// Machine 定义硬件自动化验证所依赖的设备控制接口集合。
type Machine interface {
	Instance() string

	Sensors() (hardware.Sensors, error)
	Lighting() (hardware.RGBConfig, error)
	SetLighting(config hardware.RGBConfig) error
	Backlight() (hardware.BacklightState, error)
	SetBacklight(brightness int, timeout bool) error
	PowerMode() (hardware.PowerMode, error)
	SetPowerMode(mode hardware.PowerMode) error
	LCDOverdrive() (bool, error)
	SetLCDOverdrive(enabled bool) error
	BootSound() (bool, error)
	SetBootSound(enabled bool) error
	WinKeyLocked() (bool, error)
	SetWinKeyLocked(locked bool) error
	Fans() (hardware.FanState, error)
	SetFans(config hardware.FanConfig) error
}

// StepResult 记录单项功能测试与状态复原的执行结果。
type StepResult struct {
	Name       string
	Skipped    bool
	ApplyErr   error
	RestoreErr error
}

// OK 检查测试步骤是否成功应用并成功恢复初始状态。
func (r StepResult) OK() bool {
	return r.ApplyErr == nil && r.RestoreErr == nil && !r.Skipped
}

// Original 保存各项硬件控制功能在测试前的原始状态快照。
type Original struct {
	Lighting  hardware.RGBConfig
	Backlight hardware.BacklightState
	Power     hardware.PowerMode
	Overdrive bool
	BootSound bool
	WinKey    bool
	Fans      hardware.FanState
}

// Verify 执行全套硬件功能的读写校验流程：先备份现有状态，依次触发改动并校验，最后还原初始设定。
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
	zone.Effect = hardware.EffectStatic
	zone.Speed = 3
	zone.Direction = 0
	zone.Brightness = 50
	for i := range zone.Zones {
		zone.Zones[i] = hardware.RGBColor{B: 255}
	}

	global := lighting
	global.PerZone = false
	global.Effect = hardware.EffectStatic
	global.Brightness = 50
	global.Zones[0] = hardware.RGBColor{G: 255}

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
			apply:   func() error { return device.SetPowerMode(hardware.PowerPerformance) },
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
				return device.SetFans(hardware.FanConfig{
					CPUMode: hardware.FanMax,
					GPUMode: hardware.FanMax,
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
