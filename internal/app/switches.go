package app

import (
	"fmt"

	"viper/internal/domain"
)

func (a *App) LCDOverdrive() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.LCDOverdrive()
}

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

func (a *App) WinKeyLocked() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.WinKeyLocked()
}

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

func (a *App) DiscreteGPU() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.DiscreteGPU()
}

func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	backend, err := a.hardware()
	if err != nil {
		return false, false, err
	}
	return backend.OverclockCapability()
}

func (a *App) GPUMode() (domain.GPUMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return domain.GPUModeHybrid, err
	}
	return backend.GPUMode()
}

func (a *App) SetGPUMode(mode domain.GPUMode) error {
	backend, err := a.hardware()
	if err != nil {
		return err
	}
	if err := backend.SetGPUMode(mode); err != nil {
		return fmt.Errorf("app: 设置显卡模式失败: %w", err)
	}
	return nil
}

func (a *App) BootSound() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.BootSound()
}

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
