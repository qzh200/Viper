package main

import (
	"fmt"
	"io"

	"viper/internal/gui/theme"
)

type options struct {
	screenshot string
	theme      string
	page       string
	help       bool
}

func parseArgs(args []string) (options, error) {
	var result options
	for len(args) > 0 {
		switch args[0] {
		case "--screenshot":
			if len(args) < 2 {
				return result, fmt.Errorf("--screenshot needs a path, for example --screenshot viper.png")
			}
			result.screenshot = args[1]
			args = args[2:]
		case "--theme":
			if len(args) < 2 {
				return result, fmt.Errorf("--theme needs a mode, for example --theme dark")
			}
			mode, ok := theme.Parse(args[1])
			if !ok {
				return result, fmt.Errorf("unknown appearance %q, want auto, light or dark", args[1])
			}
			result.theme = string(mode)
			args = args[2:]
		case "--page":
			if len(args) < 2 {
				return result, fmt.Errorf("--page needs a page id, for example --page fans")
			}
			result.page = args[1]
			args = args[2:]
		case "--help", "-h":
			result.help = true
			return result, nil
		default:
			return result, fmt.Errorf("unknown option %q", args[0])
		}
	}
	return result, nil
}

func usage(out io.Writer) {
	fmt.Fprintln(out, `viperui - the native control centre of Acer gaming laptops

Usage:
  viperui                       open the control centre
  viperui --screenshot <path>   draw one frame into a PNG, then quit
  viperui --theme <mode>        use auto, light or dark for this run

The window is drawn by MyGo itself, not by a web page. The appearance can be
light, dark, or the one of the system; it is remembered in the settings file.`)
}
