package quickaccess

import (
	"fmt"
	"strconv"
	"strings"
)

// QuickAccess 服务支持的功能命令常量。
const (
	CommandUSBCharge          = "USBCharge"
	CommandUSBChargeSwitch    = "USBChargeSwitch"
	CommandDisplayMode        = "DisplayModeSwitch"
	CommandDarkMode           = "DarkMode"
	CommandTransparentEffect  = "TransparentEffect"
	CommandAutoHideTaskbar    = "AutoHideTaskbar"
	CommandScreenSaver        = "ScreenSaver"
	CommandScreenSaverTimeout = "ScreenSaverTimeout"
)

// DisplayMode 表示显卡显示输出工作模式。
type DisplayMode int

const (
	// DisplayModeHybrid 混合输出模式。
	DisplayModeHybrid DisplayMode = 1
	// DisplayModeDiscrete 独立显卡直连模式。
	DisplayModeDiscrete DisplayMode = 2
	// DisplayModeAuto 自动显示切换模式。
	DisplayModeAuto DisplayMode = 3
)

// String 返回显示模式的文本表示。
func (d DisplayMode) String() string {
	switch d {
	case DisplayModeHybrid:
		return "hybrid"
	case DisplayModeDiscrete:
		return "discrete"
	case DisplayModeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// ParseDisplayMode 解析显示模式字符串。
func ParseDisplayMode(name string) (DisplayMode, bool) {
	switch name {
	case "hybrid", "optimus", "mshybrid":
		return DisplayModeHybrid, true
	case "discrete", "dgpu", "nvidia", "only":
		return DisplayModeDiscrete, true
	case "auto", "automatic", "dds":
		return DisplayModeAuto, true
	default:
		return DisplayModeHybrid, false
	}
}

// USBChargeLimit 查询关机 USB 供电电量下限阈值百分比。
func (c *Client) USBChargeLimit() (int, error) {
	result, err := c.Get(CommandUSBCharge)
	if err != nil {
		return 0, err
	}
	value, err := intValue(result)
	if err != nil {
		return 0, fmt.Errorf("quickaccess: %s: %w", CommandUSBCharge, err)
	}
	return value, nil
}

// SetUSBChargeLimit 设置关机 USB 供电终止的电池百分比下限（通常为 10, 20 或 30）。
func (c *Client) SetUSBChargeLimit(percent int) error {
	switch percent {
	case 10, 20, 30:
	default:
		return fmt.Errorf("quickaccess: the charging limit %d is not one of 10, 20 and 30", percent)
	}
	return c.Set(CommandUSBCharge, percent)
}

// USBChargeEnabled 查询关机 USB 供电功能是否开启。
func (c *Client) USBChargeEnabled() (bool, error) {
	result, err := c.Get(CommandUSBChargeSwitch)
	if err != nil {
		return false, err
	}
	value, ok := result.Value.(bool)
	if !ok {
		return false, fmt.Errorf("quickaccess: %s: the service answered with %v", CommandUSBChargeSwitch, result.Value)
	}
	return value, nil
}

// SetUSBChargeEnabled 启用或停用关机状态下的 USB 端口充电功能。
func (c *Client) SetUSBChargeEnabled(enabled bool) error {
	return c.Set(CommandUSBChargeSwitch, enabled)
}

// DisplayMode 查询当前的显卡显示模式。
func (c *Client) DisplayMode() (DisplayMode, error) {
	result, err := c.Get(CommandDisplayMode)
	if err != nil {
		return DisplayModeHybrid, err
	}
	value, err := intValue(result)
	if err != nil {
		return DisplayModeHybrid, fmt.Errorf("quickaccess: %s: %w", CommandDisplayMode, err)
	}
	return DisplayMode(value), nil
}

// SetDisplayMode 设置显卡显示模式。
func (c *Client) SetDisplayMode(mode DisplayMode) error {
	switch mode {
	case DisplayModeHybrid, DisplayModeDiscrete, DisplayModeAuto:
	default:
		return fmt.Errorf("quickaccess: the display mode %d is not known", int(mode))
	}
	return c.Set(CommandDisplayMode, int(mode))
}

// DarkMode 查询 Windows 深色模式开关。
func (c *Client) DarkMode() (bool, error) {
	result, err := c.Get(CommandDarkMode)
	if err != nil {
		return false, err
	}
	value, ok := result.Value.(bool)
	if !ok {
		return false, fmt.Errorf("quickaccess: %s: the service answered with %v", CommandDarkMode, result.Value)
	}
	return value, nil
}

// SetDarkMode 设置系统深色/浅色模式。
func (c *Client) SetDarkMode(enabled bool) error {
	return c.Set(CommandDarkMode, enabled)
}

// TransparentEffect 查询系统窗口背景透明视觉特效开关。
func (c *Client) TransparentEffect() (bool, error) {
	result, err := c.Get(CommandTransparentEffect)
	if err != nil {
		return false, err
	}
	value, err := switchValue(result)
	if err != nil {
		return false, fmt.Errorf("quickaccess: %s: %w", CommandTransparentEffect, err)
	}
	return value, nil
}

// SetTransparentEffect 启用或关闭系统窗口透明视觉特效。
func (c *Client) SetTransparentEffect(enabled bool) error {
	return c.Set(CommandTransparentEffect, switchNumber(enabled))
}

// AutoHideTaskbar 查询任务栏自动隐藏设置。
func (c *Client) AutoHideTaskbar() (bool, error) {
	result, err := c.Get(CommandAutoHideTaskbar)
	if err != nil {
		return false, err
	}
	value, err := switchValue(result)
	if err != nil {
		return false, fmt.Errorf("quickaccess: %s: %w", CommandAutoHideTaskbar, err)
	}
	return value, nil
}

// SetAutoHideTaskbar 启用或关闭任务栏自动隐藏功能。
func (c *Client) SetAutoHideTaskbar(enabled bool) error {
	return c.Set(CommandAutoHideTaskbar, switchNumber(enabled))
}

// ScreenSaver 查询屏幕保护程序是否开启。
func (c *Client) ScreenSaver() (bool, error) {
	result, err := c.Get(CommandScreenSaver)
	if err != nil {
		return false, err
	}
	value, err := switchValue(result)
	if err != nil {
		return false, fmt.Errorf("quickaccess: %s: %w", CommandScreenSaver, err)
	}
	return value, nil
}

// SetScreenSaver 启用或停用屏幕保护程序。
func (c *Client) SetScreenSaver(enabled bool) error {
	return c.Set(CommandScreenSaver, enabled)
}

// ScreenSaverTimeout 获取屏幕保护程序的超时等待秒数。
func (c *Client) ScreenSaverTimeout() (int, error) {
	result, err := c.Get(CommandScreenSaverTimeout)
	if err != nil {
		return 0, err
	}
	value, err := intValue(result)
	if err != nil {
		return 0, fmt.Errorf("quickaccess: %s: %w", CommandScreenSaverTimeout, err)
	}
	return value, nil
}

// SetScreenSaverTimeout 设置屏幕保护程序激活前的空闲超时等待秒数。
func (c *Client) SetScreenSaverTimeout(seconds int) error {
	if seconds < 0 {
		return fmt.Errorf("quickaccess: 屏保时间 %d 秒不是正数", seconds)
	}
	return c.Set(CommandScreenSaverTimeout, seconds)
}

func switchValue(result Result) (bool, error) {
	switch value := result.Value.(type) {
	case bool:
		return value, nil
	case float64:
		return value != 0, nil
	case int:
		return value != 0, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "on", "yes":
			return true, nil
		case "0", "false", "off", "no", "":
			return false, nil
		}
	}
	return false, fmt.Errorf("服务答话 %v 不是开关", result.Value)
}

func switchNumber(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

func intValue(result Result) (int, error) {
	switch value := result.Value.(type) {
	case float64:
		return int(value), nil
	case string:
		return strconv.Atoi(value)
	case int:
		return value, nil
	default:
		return 0, fmt.Errorf("the service answered with %v", result.Value)
	}
}
