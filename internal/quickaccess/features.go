package quickaccess

import (
	"fmt"
	"strconv"
)

// The commands of the service that Viper uses.
const (
	// CommandBluelightShield selects the blue light shield.
	CommandBluelightShield = "BluelightShield"
	// CommandUSBCharge holds the charging limit in percent.
	CommandUSBCharge = "USBCharge"
	// CommandUSBChargeSwitch allows the charging over USB.
	CommandUSBChargeSwitch = "USBChargeSwitch"
	// CommandDisplayMode selects the graphics switch of the machine.
	CommandDisplayMode = "DisplayModeSwitch"
	// CommandDarkMode switches the dark mode of the system.
	CommandDarkMode = "DarkMode"
)

// BlueLight is a level of the blue light shield.
//
// The names are the ones the service uses in its own description.
type BlueLight int

// The levels of the blue light shield.
const (
	// BlueLightDisabled switches the shield off.
	BlueLightDisabled BlueLight = 0
	// BlueLightLow reduces the blue light a little.
	BlueLightLow BlueLight = 1
	// BlueLightMedium reduces the blue light halfway.
	BlueLightMedium BlueLight = 2
	// BlueLightHigh reduces the blue light strongly.
	BlueLightHigh BlueLight = 3
	// BlueLightLongHour is the level for long working days.
	BlueLightLongHour BlueLight = 4
	// BlueLightCustomised is the level that the service reports when the
	// shield is not set to one of the presets. The service cannot write
	// this level.
	BlueLightCustomised BlueLight = 5
)

// String renders the level for display.
func (b BlueLight) String() string {
	switch b {
	case BlueLightDisabled:
		return "disabled"
	case BlueLightLow:
		return "low"
	case BlueLightMedium:
		return "medium"
	case BlueLightHigh:
		return "high"
	case BlueLightLongHour:
		return "long hour"
	case BlueLightCustomised:
		return "customised"
	default:
		return "unknown"
	}
}

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

// BlueLightShield reads the level of the blue light shield.
func (c *Client) BlueLightShield() (BlueLight, error) {
	result, err := c.Get(CommandBluelightShield)
	if err != nil {
		return BlueLightDisabled, err
	}
	value, err := intValue(result)
	if err != nil {
		return BlueLightDisabled, fmt.Errorf("quickaccess: %s: %w", CommandBluelightShield, err)
	}
	return BlueLight(value), nil
}

// SetBlueLightShield selects a level of the blue light shield.
func (c *Client) SetBlueLightShield(level BlueLight) error {
	if err := c.Set(CommandBluelightShield, int(level)); err != nil {
		return err
	}
	return nil
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
