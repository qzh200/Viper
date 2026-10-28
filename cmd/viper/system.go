package main

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/quickaccess"
)

// The settings of this file belong to the Quick Access service of the
// original software. They are not part of the gaming hardware provider,
// so they need the service of the original stack to be running.
//
// 这几个设置属于原厂软件的 Quick Access 服务，不属于游戏硬件的固件接口，
// 所以它们需要原厂服务在跑。

// runDisplayMode shows or changes the way the machine drives the graphics
// chip.
//
// The setting is the one of the Quick Access service. The hardware
// provider has a second way to the same switch, that one is the command
// "viper gpu".
func runDisplayMode(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		mode, err := application.DisplayMode()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  display mode:", mode)
		return nil
	}

	mode, ok := quickaccess.ParseDisplayMode(strings.ToLower(args[0]))
	if !ok {
		return fmt.Errorf("unknown display mode %q, use hybrid, discrete or auto", args[0])
	}
	if err := application.SetDisplayMode(mode); err != nil {
		return err
	}
	current, err := application.DisplayMode()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  display mode:", current)
	return nil
}

// runDarkMode shows or changes the dark mode of the system.
func runDarkMode(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		enabled, err := application.DarkMode()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  dark mode:", onOff(enabled))
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetDarkMode(enabled); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  dark mode:", onOff(enabled))
	return nil
}

// runTransparent shows or changes the transparency of the windows.
func runTransparent(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		enabled, err := application.TransparentEffect()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  transparent effect:", onOff(enabled))
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetTransparentEffect(enabled); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  transparent effect:", onOff(enabled))
	return nil
}

// runTaskbar shows or changes the automatic hiding of the taskbar.
//
// The setting is switched on when the taskbar hides itself.
func runTaskbar(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) == 0 {
		hidden, err := application.AutoHideTaskbar()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  auto hide taskbar:", onOff(hidden))
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetAutoHideTaskbar(enabled); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  auto hide taskbar:", onOff(enabled))
	return nil
}

// runScreenSaver shows or changes the screen saver.
//
// The second form sets the idle time in seconds before the screen saver
// starts.
func runScreenSaver(args []string) error {
	application, err := newApp()
	if err != nil {
		return err
	}
	defer application.Close()

	if len(args) > 0 && (args[0] == "timeout" || args[0] == "delay") {
		if len(args) < 2 {
			return fmt.Errorf("usage: viper screensaver timeout <seconds>")
		}
		seconds, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		if seconds < 0 {
			return fmt.Errorf("the timeout cannot be negative")
		}
		if err := application.SetScreenSaverTimeout(seconds); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  screen saver timeout:", seconds, "seconds")
		return nil
	}

	if len(args) == 0 {
		enabled, err := application.ScreenSaver()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, "  screen saver:", onOff(enabled))
		timeout, err := application.ScreenSaverTimeout()
		if err == nil {
			fmt.Fprintln(stdout, "  timeout     :", timeout, "seconds")
		}
		return nil
	}

	enabled, err := parseOnOff(args[0])
	if err != nil {
		return err
	}
	if err := application.SetScreenSaver(enabled); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "  screen saver:", onOff(enabled))
	return nil
}
