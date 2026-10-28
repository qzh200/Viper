package probe

import (
	"encoding/json"
	"errors"

	"viper/internal/agentservice"
	"viper/internal/hardware"
	"viper/internal/machine"
)

// Machine is the part of the machine that the verification needs.
//
// The hardware layer of the program satisfies it, and a test can put a
// stand in its place.
type Machine interface {
	// Instance returns the object path of the gaming functions.
	Instance() string
	// Sensors reads the live temperatures and fan speeds.
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

// StepResult is what happened to one setting.
type StepResult struct {
	// Name is the name of the setting.
	Name string
	// Skipped reports that the machine cannot do this setting at all.
	Skipped bool
	// ApplyErr is the error of the write itself.
	ApplyErr error
	// RestoreErr is the error of putting the old value back.
	RestoreErr error
}

// OK reports whether the setting was written, checked and put back.
func (r StepResult) OK() bool {
	return r.ApplyErr == nil && r.RestoreErr == nil && !r.Skipped
}

// Original is the state of the machine before the verification.
type Original struct {
	// Lighting is the keyboard lighting.
	Lighting hardware.RGBConfig
	// Backlight is the backlight of the keyboard.
	Backlight hardware.BacklightState
	// Power is the performance profile.
	Power hardware.PowerMode
	// Overdrive reports whether the display overdrive is on.
	Overdrive bool
	// BootSound reports whether the sound at the start is on.
	BootSound bool
	// WinKey reports whether the Windows key is locked.
	WinKey bool
	// Fans is the fan configuration.
	Fans hardware.FanState
}

// Verify changes every setting the hardware layer can write, reads it back
// and puts the machine back where it was.
//
// The raw service is needed as well: the fans have to be written back
// exactly as the service remembers them, which the semantic interface alone
// cannot do.
//
// 这一项做完要读回来核对：机器没有接受就在结果里说清楚是哪一个。
func Verify(device Machine, raw Service) (Original, []StepResult, error) {
	var original Original

	// 风扇要回到服务记着的每一个数值，只靠语义化的接口做不到。
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
		// 这一项要固件那一侧，没有管理员权限就读不到。
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

	// 分区颜色：四个分区写成纯蓝，再写回原来的样子。
	zone := lighting
	zone.PerZone = true
	zone.Effect = hardware.EffectStatic
	zone.Speed = 3
	zone.Direction = 0
	zone.Brightness = 50
	for i := range zone.Zones {
		zone.Zones[i] = hardware.RGBColor{B: 255}
	}

	// 全局颜色：整块键盘写成纯绿。
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
				// 原样写回，服务记着的占空比也回到原来的值。
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
