package app

import (
	"fmt"

	"viper/internal/hardware"
)

// LCDOverdrive 查询屏幕响应时间超速加速功能是否开启。
func (a *App) LCDOverdrive() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.LCDOverdrive()
}

// SetLCDOverdrive 设置屏幕 LCD Overdrive 响应时间加速开关。
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

// WinKeyLocked 查询 Windows 徽标键是否处于锁定禁用状态。
func (a *App) WinKeyLocked() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.WinKeyLocked()
}

// SetWinKeyLocked 启用或禁用 Windows 徽标键锁定。
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

// DiscreteGPU 查询当前是否工作在纯独显直连模式。
func (a *App) DiscreteGPU() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.DiscreteGPU()
}

// OverclockCapability 检测 GPU 与 CPU 的超频支持能力。
func (a *App) OverclockCapability() (gpu bool, cpu bool, err error) {
	backend, err := a.hardware()
	if err != nil {
		return false, false, err
	}
	return backend.OverclockCapability()
}

// GPUMode 查询当前的显卡工作模式。
func (a *App) GPUMode() (hardware.GPUMode, error) {
	backend, err := a.hardware()
	if err != nil {
		return hardware.GPUModeHybrid, err
	}
	return backend.GPUMode()
}

// SetGPUMode 切换显卡模式（混合/独显直连/自动）。
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

// BootSound 查询开机启动音效是否开启。
func (a *App) BootSound() (bool, error) {
	backend, err := a.hardware()
	if err != nil {
		return false, err
	}
	return backend.BootSound()
}

// SetBootSound 设置开机启动音效开关。
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
