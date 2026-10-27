// 开关类的设置：屏幕超频、Windows 键、开机音效、显卡模式。
package app

import (
	"fmt"

	"viper/internal/hardware"
)

// LCDOverdrive reports whether the display overdrive is active.
func (a *App) LCDOverdrive() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.LCDOverdrive()
}

// SetLCDOverdrive enables or disables the display overdrive.
func (a *App) SetLCDOverdrive(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetLCDOverdrive(enabled); err != nil {
		return fmt.Errorf("app: 设置屏幕超频失败: %w", err)
	}
	return nil
}

// WinKeyLocked reports whether the Windows key is locked.
func (a *App) WinKeyLocked() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.WinKeyLocked()
}

// SetWinKeyLocked locks or unlocks the Windows key.
func (a *App) SetWinKeyLocked(locked bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetWinKeyLocked(locked); err != nil {
		return fmt.Errorf("app: 设置 Windows 键锁定失败: %w", err)
	}
	return nil
}

// StickyKeys reports whether the sticky keys are switched on.
//
// The sticky keys belong to the accessibility settings of the current
// user, they are not part of the gaming hardware.
func (a *App) StickyKeys() (bool, error) { return hardware.StickyKeys() }

// SetStickyKeys switches the sticky keys on or off.
func (a *App) SetStickyKeys(enabled bool) error { return hardware.SetStickyKeys(enabled) }

// DiscreteGPU reports the state of the discrete GPU switch.
func (a *App) DiscreteGPU() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.DiscreteGPU()
}

// OverclockCapability reports which overclocking features the machine has.
func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	backend, err := a.hardware()
	if err != nil {
		return false, false, err
	}
	return backend.OverclockCapability()
}

// GPUMode reads the graphics mode of the machine.
func (a *App) GPUMode() (hardware.GPUMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.GPUModeHybrid, err
	}
	return backend.GPUMode()
}

// SetGPUMode selects the graphics mode of the machine.
func (a *App) SetGPUMode(mode hardware.GPUMode) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetGPUMode(mode); err != nil {
		return fmt.Errorf("app: 设置显卡模式失败: %w", err)
	}
	return nil
}

// BootSound reports whether the sound at the start of the machine is on.
func (a *App) BootSound() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.BootSound()
}

// SetBootSound switches the sound at the start of the machine on or off.
func (a *App) SetBootSound(enabled bool) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetBootSound(enabled); err != nil {
		return fmt.Errorf("app: 设置开机声音失败: %w", err)
	}
	return nil
}
