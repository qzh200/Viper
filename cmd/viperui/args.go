package main

import (
	"fmt"
	"io"

	"viper/internal/gui/theme"
)

// options 是 viperui 认得的命令行选项。
type options struct {
	// screenshot 是要存 PNG 的路径；空串表示不截屏。
	screenshot string
	// theme 是这一次运行要用的外观模式；空串表示用存下来的那个。
	theme string
	// page 是这一次运行打开的初始页面；空串表示默认总览页。
	page string
	// help 表示只打印用法。
	help bool
}

// parseArgs 解析命令行参数。viperui 没有别的子命令，只有这两个选项。
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

// usage 打印用法。
func usage(out io.Writer) {
	fmt.Fprintln(out, `viperui - the native control centre of Acer gaming laptops

Usage:
  viperui                       open the control centre
  viperui --screenshot <path>   draw one frame into a PNG, then quit
  viperui --theme <mode>        use auto, light or dark for this run

The window is drawn by MyGo itself, not by a web page. The appearance can be
light, dark, or the one of the system; it is remembered in the settings file.`)
}
