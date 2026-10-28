// 原厂 Quick Access 服务管的那几项设置：USB 充电、显示模式等。
package app

import (
	"fmt"

	"viper/internal/quickaccess"
)

// QuickAccess is the set of abilities of the Quick Access service that
// the program uses.
//
// The interface keeps the program testable: a test can hand the program
// a stand for the service, so that the behaviour can be checked without
// the service of the original software.
type QuickAccess interface {
	// USBChargeLimit reads the charging limit in percent.
	USBChargeLimit() (int, error)
	// SetUSBChargeLimit sets the charging limit in percent.
	SetUSBChargeLimit(percent int) error
	// USBChargeEnabled reports whether the charging over USB is allowed.
	USBChargeEnabled() (bool, error)
	// SetUSBChargeEnabled allows or forbids the charging over USB.
	SetUSBChargeEnabled(enabled bool) error
	// DisplayMode 读取机器的显卡切换方式。
	DisplayMode() (quickaccess.DisplayMode, error)
	// SetDisplayMode 选择机器的显卡切换方式。
	SetDisplayMode(mode quickaccess.DisplayMode) error
	// DarkMode 报告系统的深色模式是否打开。
	DarkMode() (bool, error)
	// SetDarkMode 打开或关闭系统的深色模式。
	SetDarkMode(enabled bool) error
	// TransparentEffect 报告系统的窗口是不是透明的。
	TransparentEffect() (bool, error)
	// SetTransparentEffect 打开或关闭窗口的透明效果。
	SetTransparentEffect(enabled bool) error
	// AutoHideTaskbar 报告任务栏是不是自己藏起来。
	AutoHideTaskbar() (bool, error)
	// SetAutoHideTaskbar 打开或关闭任务栏的自动隐藏。
	SetAutoHideTaskbar(enabled bool) error
	// ScreenSaver 报告屏幕保护程序是不是打开的。
	ScreenSaver() (bool, error)
	// SetScreenSaver 打开或关闭屏幕保护程序。
	SetScreenSaver(enabled bool) error
	// ScreenSaverTimeout 读取屏幕保护程序启动前的空闲时间，单位是秒。
	ScreenSaverTimeout() (int, error)
	// SetScreenSaverTimeout 设置屏幕保护程序启动前的空闲时间，单位是秒。
	SetScreenSaverTimeout(seconds int) error
	// Close ends the session.
	Close() error
}

// USBChargeLimit reads the charging limit in percent.
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

// SetUSBChargeLimit sets the charging limit in percent.
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

// USBChargeEnabled reports whether the charging over USB is allowed.
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

// SetUSBChargeEnabled allows or forbids the charging over USB.
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

// DisplayMode reads the graphics switch of the machine.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
func (a *App) DisplayMode() (quickaccess.DisplayMode, error) {
	client, release, err := a.quick()
	if err != nil {
		return quickaccess.DisplayModeHybrid, err
	}
	defer release()
	mode, err := client.DisplayMode()
	if err != nil {
		return quickaccess.DisplayModeHybrid, fmt.Errorf("app: 读取显卡切换方式失败: %w", err)
	}
	return mode, nil
}

// SetDisplayMode selects the graphics switch of the machine.
func (a *App) SetDisplayMode(mode quickaccess.DisplayMode) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetDisplayMode(mode); err != nil {
		return fmt.Errorf("app: 设置显卡切换方式失败: %w", err)
	}
	return nil
}

// DarkMode reports whether the dark mode of the system is switched on.
//
// The setting belongs to the Quick Access service of the original stack,
// it is not part of the gaming hardware provider.
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

// SetDarkMode switches the dark mode of the system on or off.
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

// TransparentEffect reports whether the windows of the system are
// transparent.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the effect is switched off.
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

// SetTransparentEffect switches the transparency of the windows.
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

// AutoHideTaskbar reports whether the taskbar hides itself.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the taskbar does not hide itself.
func (a *App) AutoHideTaskbar() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	hidden, err := client.AutoHideTaskbar()
	if err != nil {
		return false, fmt.Errorf("app: 读取任务栏自动隐藏失败: %w", err)
	}
	return hidden, nil
}

// SetAutoHideTaskbar switches the automatic hiding of the taskbar.
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

// ScreenSaver reports whether the screen saver is switched on.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error, the
// program never answers that the screen saver is switched off.
func (a *App) ScreenSaver() (bool, error) {
	client, release, err := a.quick()
	if err != nil {
		return false, err
	}
	defer release()
	enabled, err := client.ScreenSaver()
	if err != nil {
		return false, fmt.Errorf("app: 读取屏幕保护程序失败: %w", err)
	}
	return enabled, nil
}

// SetScreenSaver switches the screen saver on or off.
func (a *App) SetScreenSaver(enabled bool) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaver(enabled); err != nil {
		return fmt.Errorf("app: 设置屏幕保护程序失败: %w", err)
	}
	return nil
}

// ScreenSaverTimeout reads the idle time in seconds before the screen
// saver starts.
//
// The ability belongs to the Quick Access service of the original stack.
// A service that does not know the command answers with an error.
func (a *App) ScreenSaverTimeout() (int, error) {
	client, release, err := a.quick()
	if err != nil {
		return 0, err
	}
	defer release()
	timeout, err := client.ScreenSaverTimeout()
	if err != nil {
		return 0, fmt.Errorf("app: 读取屏幕保护程序超时失败: %w", err)
	}
	return timeout, nil
}

// SetScreenSaverTimeout sets the idle time in seconds before the screen
// saver starts.
func (a *App) SetScreenSaverTimeout(seconds int) error {
	client, release, err := a.quick()
	if err != nil {
		return err
	}
	defer release()
	if err := client.SetScreenSaverTimeout(seconds); err != nil {
		return fmt.Errorf("app: 设置屏幕保护程序超时失败: %w", err)
	}
	return nil
}

// quick opens a session with the Quick Access service of the original
// stack.
//
// The service owns the settings that do not belong to the gaming
// hardware, such as the charging over USB. It is not needed for the other
// settings, so the session is only opened when one of them is used.
//
// 注入给程序的会话属于注入的人，程序不会关掉它。程序自己开的会话每次
// 调用都用完就关，因为关掉的会话不能再用第二次。
//
// 返回的函数负责结束会话，用完之后必须调用。
func (a *App) quick() (QuickAccess, func(), error) {
	a.mu.Lock()
	client := a.quickClient
	options := a.quickOptions
	a.mu.Unlock()

	if client != nil {
		return client, func() {}, nil
	}

	owned, err := quickaccess.Connect(options)
	if err != nil {
		return nil, nil, fmt.Errorf("app: 连不上原厂软件的 Quick Access 服务: %w", err)
	}
	return owned, func() { _ = owned.Close() }, nil
}
