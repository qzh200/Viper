package quickaccess

import (
	"fmt"
	"strconv"
	"strings"
)

// The commands of the service that Viper uses.
const (
	// CommandUSBCharge holds the charging limit in percent.
	CommandUSBCharge = "USBCharge"
	// CommandUSBChargeSwitch allows the charging over USB.
	CommandUSBChargeSwitch = "USBChargeSwitch"
	// CommandDisplayMode selects the graphics switch of the machine.
	CommandDisplayMode = "DisplayModeSwitch"
	// CommandDarkMode switches the dark mode of the system.
	CommandDarkMode = "DarkMode"
	// CommandTransparentEffect switches the transparent effect of the
	// system.
	CommandTransparentEffect = "TransparentEffect"
	// CommandAutoHideTaskbar switches the automatic hiding of the taskbar.
	CommandAutoHideTaskbar = "AutoHideTaskbar"
	// CommandScreenSaver switches the screen saver of the system.
	CommandScreenSaver = "ScreenSaver"
	// CommandScreenSaverTimeout holds the idle time before the screen
	// saver starts.
	CommandScreenSaverTimeout = "ScreenSaverTimeout"
)

// DisplayMode is the way the machine drives the graphics chip.
type DisplayMode int

// The display modes the service offers.
const (
	// DisplayModeHybrid keeps the integrated graphics in charge.
	DisplayModeHybrid DisplayMode = 1
	// DisplayModeDiscrete hands the display over to the dedicated
	// graphics.
	DisplayModeDiscrete DisplayMode = 2
	// DisplayModeAuto lets the system pick the mode on its own.
	DisplayModeAuto DisplayMode = 3
)

// String renders the mode for display.
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

// ParseDisplayMode maps a command line word onto a display mode.
//
// The words are the ones that the hardware layer uses for the same
// setting, so that a user does not have to learn two vocabularies.
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

// USBChargeLimit reads the charging limit in percent.
//
// The service offers the limits ten, twenty and thirty percent.
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

// SetUSBChargeLimit sets the charging limit in percent.
func (c *Client) SetUSBChargeLimit(percent int) error {
	switch percent {
	case 10, 20, 30:
	default:
		return fmt.Errorf("quickaccess: the charging limit %d is not one of 10, 20 and 30", percent)
	}
	return c.Set(CommandUSBCharge, percent)
}

// USBChargeEnabled reports whether the charging over USB is allowed.
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

// SetUSBChargeEnabled allows or forbids the charging over USB.
func (c *Client) SetUSBChargeEnabled(enabled bool) error {
	return c.Set(CommandUSBChargeSwitch, enabled)
}

// DisplayMode reads the graphics switch of the machine.
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

// SetDisplayMode selects the graphics switch of the machine.
func (c *Client) SetDisplayMode(mode DisplayMode) error {
	switch mode {
	case DisplayModeHybrid, DisplayModeDiscrete, DisplayModeAuto:
	default:
		return fmt.Errorf("quickaccess: the display mode %d is not known", int(mode))
	}
	return c.Set(CommandDisplayMode, int(mode))
}

// DarkMode reports whether the dark mode of the system is switched on.
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

// SetDarkMode switches the dark mode of the system on or off.
func (c *Client) SetDarkMode(enabled bool) error {
	return c.Set(CommandDarkMode, enabled)
}

// TransparentEffect reports whether the transparent effect of the system is
// switched on.
//
// The service reports this state as a number, one for on and zero for off.
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

// SetTransparentEffect switches the transparent effect of the system on or
// off.
//
// The service keeps this state as the number that it reports itself, one
// for on and zero for off, so that is what is written.
func (c *Client) SetTransparentEffect(enabled bool) error {
	return c.Set(CommandTransparentEffect, switchNumber(enabled))
}

// AutoHideTaskbar reports whether the taskbar of the system hides itself.
//
// The service reports this state as a number, one for on and zero for off.
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

// SetAutoHideTaskbar hides the taskbar of the system or shows it again.
//
// The service keeps this state as the number that it reports itself, one
// for hidden and zero for shown, so that is what is written.
func (c *Client) SetAutoHideTaskbar(enabled bool) error {
	return c.Set(CommandAutoHideTaskbar, switchNumber(enabled))
}

// ScreenSaver reports whether the screen saver of the system is switched
// on.
//
// The service reports this state as a switch, not as a number.
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

// SetScreenSaver switches the screen saver of the system on or off.
func (c *Client) SetScreenSaver(enabled bool) error {
	return c.Set(CommandScreenSaver, enabled)
}

// ScreenSaverTimeout reads the idle time in seconds before the screen saver
// of the system starts.
//
// The service answers with zero while the screen saver is switched off.
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

// SetScreenSaverTimeout sets the idle time in seconds before the screen
// saver of the system starts.
//
// The official application always writes six hundred seconds, the service
// takes any number of seconds.
func (c *Client) SetScreenSaverTimeout(seconds int) error {
	if seconds < 0 {
		return fmt.Errorf("quickaccess: 屏保时间 %d 秒不是正数", seconds)
	}
	return c.Set(CommandScreenSaverTimeout, seconds)
}

// switchValue reads a switch of an answer.
//
// The service uses both shapes: some commands answer with a switch and some
// answer with a number, one for on and zero for off. Both are taken for a
// switch.
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

// switchNumber turns a switch into the number that the service uses.
func switchNumber(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

// intValue reads the number of an answer.
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
