package app

import (
	"fmt"

	"viper/internal/quickaccess"
)

type QuickAccess interface {
	USBChargeLimit() (int, error)
	SetUSBChargeLimit(percent int) error
	USBChargeEnabled() (bool, error)
	SetUSBChargeEnabled(enabled bool) error
	DisplayMode() (quickaccess.DisplayMode, error)
	SetDisplayMode(mode quickaccess.DisplayMode) error
	DarkMode() (bool, error)
	SetDarkMode(enabled bool) error
	TransparentEffect() (bool, error)
	SetTransparentEffect(enabled bool) error
	AutoHideTaskbar() (bool, error)
	SetAutoHideTaskbar(enabled bool) error
	ScreenSaver() (bool, error)
	SetScreenSaver(enabled bool) error
	ScreenSaverTimeout() (int, error)
	SetScreenSaverTimeout(seconds int) error
	Close() error
}

func (a *App) USBChargeLimit() (int, error) {
	client, release, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer release()
	limit, err := client.USBChargeLimit()
	if err != nil {
		return 0, fmt.Errorf("app: 读取充电上限失败: %w", err)
	}
	return limit, nil
}

func (a *App) SetUSBChargeLimit(percent int) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetUSBChargeLimit(percent); err != nil {
		return fmt.Errorf("app: 设置充电上限失败: %w", err)
	}
	return nil
}

func (a *App) USBChargeEnabled() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.USBChargeEnabled()
	if err != nil {
		return false, fmt.Errorf("app: 读取 USB 充电开关失败: %w", err)
	}
	return enabled, nil
}

func (a *App) SetUSBChargeEnabled(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetUSBChargeEnabled(enabled); err != nil {
		return fmt.Errorf("app: 设置 USB 充电开关失败: %w", err)
	}
	return nil
}

func (a *App) DisplayMode() (quickaccess.DisplayMode, error) {
	client, release, err := a.quick()
	if err != nil {
		return quickaccess.DisplayModeHybrid, err
	}
	defer release()
	mode, err := client.DisplayMode()
	if err != nil {
		return quickaccess.DisplayModeHybrid, fmt.Errorf("app: 读取显示模式失败: %w", err)
	}
	return mode, nil
}

func (a *App) SetDisplayMode(mode quickaccess.DisplayMode) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetDisplayMode(mode); err != nil {
		return fmt.Errorf("app: 设置显示模式失败: %w", err)
	}
	return nil
}

func (a *App) DarkMode() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.DarkMode()
	if err != nil {
		return false, fmt.Errorf("app: 读取深色模式失败: %w", err)
	}
	return enabled, nil
}

func (a *App) SetDarkMode(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetDarkMode(enabled); err != nil {
		return fmt.Errorf("app: 设置深色模式失败: %w", err)
	}
	return nil
}

func (a *App) TransparentEffect() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.TransparentEffect()
	if err != nil {
		return false, fmt.Errorf("app: 读取透明效果失败: %w", err)
	}
	return enabled, nil
}

func (a *App) SetTransparentEffect(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetTransparentEffect(enabled); err != nil {
		return fmt.Errorf("app: 设置透明效果失败: %w", err)
	}
	return nil
}

func (a *App) AutoHideTaskbar() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.AutoHideTaskbar()
	if err != nil {
		return false, fmt.Errorf("app: 读取任务栏自动隐藏失败: %w", err)
	}
	return enabled, nil
}

func (a *App) SetAutoHideTaskbar(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetAutoHideTaskbar(enabled); err != nil {
		return fmt.Errorf("app: 设置任务栏自动隐藏失败: %w", err)
	}
	return nil
}

func (a *App) ScreenSaver() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.ScreenSaver()
	if err != nil {
		return false, fmt.Errorf("app: 读取屏幕保护开关失败: %w", err)
	}
	return enabled, nil
}

func (a *App) SetScreenSaver(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaver(enabled); err != nil {
		return fmt.Errorf("app: 设置屏幕保护开关失败: %w", err)
	}
	return nil
}

func (a *App) ScreenSaverTimeout() (int, error) {
	client, release, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer release()
	seconds, err := client.ScreenSaverTimeout()
	if err != nil {
		return 0, fmt.Errorf("app: 读取屏幕保护时间失败: %w", err)
	}
	return seconds, nil
}

func (a *App) SetScreenSaverTimeout(seconds int) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaverTimeout(seconds); err != nil {
		return fmt.Errorf("app: 设置屏幕保护时间失败: %w", err)
	}
	return nil
}

func (a *App) quick() (QuickAccess, func(), error) {
	a.mu.Lock()
	if a.quickClient != nil {
		client := a.quickClient
		a.mu.Unlock()
		return client, func() {}, nil
	}
	options := a.quickOptions
	a.mu.Unlock()

	client, err := quickaccess.Connect(options)
	if err != nil {
		return nil, nil, fmt.Errorf("app: 连接 Quick Access 服务失败: %w", err)
	}
	return client, func() { client.Close() }, nil
}
