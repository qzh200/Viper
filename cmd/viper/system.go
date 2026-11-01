package main

import (
	"fmt"
	"strconv"
	"strings"

	"viper/internal/quickaccess"
)

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
