// viperui 是 Viper 的原生图形界面。
//
// 窗口由 MyGo 自己画出，不经过网页：Windows 上是真正的原生控件，
// 外观可以选亮色、暗色，或者跟着系统走。
//
// 用法：
//
//	viperui                    打开控制中心
//	viperui --screenshot 路径   画一帧存成 PNG，然后退出
//	viperui --theme dark       这一回用暗色
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"viper/internal/app"
	"viper/internal/config"
	"viper/internal/gui/capture"
	"viper/internal/gui/controller"
	"viper/internal/gui/shell"
	"viper/internal/gui/theme"
)

// desktop 是「把界面跑起来」这件事：真的实现是 MyGo，测试里可以换掉。
type desktop interface {
	// WhenReady 在程序准备好之后调用 fn。
	WhenReady(fn func())
	// OpenWindow 开一扇窗口。
	OpenWindow(options mygo.WindowOptions) *mygo.Window
	// SetAppearance 告诉桌面这一次用哪种外观：亮色、暗色，或者跟着系统。
	SetAppearance(mode theme.Mode)
	// Run 跑起主循环。
	Run() error
	// Quit 退出。
	Quit()
}

// mygoDesktop 是真的那一个。
type mygoDesktop struct{}

func (mygoDesktop) WhenReady(fn func()) { mygo.App.WhenReady(fn) }
func (mygoDesktop) OpenWindow(options mygo.WindowOptions) *mygo.Window {
	return mygo.NewWindow(options)
}

// SetAppearance 把外观强制到 MyGo 上：自动模式交还给系统。
func (mygoDesktop) SetAppearance(mode theme.Mode) {
	mygo.Theme.SetSource(shell.Source(mode))
}

func (mygoDesktop) Run() error { return mygo.App.Run() }
func (mygoDesktop) Quit()      { mygo.App.Quit() }

// parts 是一次运行装配起来的部件。
type parts struct {
	// store 是配置存储，界面的外观模式记在里面。
	store *config.Store
	// application 是业务门面，界面通过它访问机器。
	application *app.App
	// controller 是界面与硬件之间唯一的关口。
	ctrl *controller.Controller
	// window 是这扇窗口。
	window *shell.Shell
	// appearance 是这一次运行要用的外观。
	appearance theme.Mode
}

// build 装配一次运行所需的部件。
//
// 外观由命令行说了算，否则用存下来的那个；命令行定的那个不写回配置。
func build(store *config.Store, options options) *parts {
	application := app.NewWithStore(store)
	ctrl := controller.New(application, store)

	appearance := ctrl.ThemeMode()
	if options.theme != "" {
		appearance = theme.Mode(options.theme)
	}

	return &parts{
		store:       store,
		application: application,
		ctrl:        ctrl,
		window:      shell.New(ctrl, shell.WithAppearance(appearance)),
		appearance:  appearance,
	}
}

// close 收掉这次运行的部件。
func (p *parts) close() {
	p.application.Close()
	p.ctrl.Stop()
}

func main() {
	options, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "viperui:", err)
		os.Exit(2)
	}
	if options.help {
		usage(os.Stdout)
		return
	}
	if err := run(options, mygoDesktop{}); err != nil {
		log.Fatal(err)
	}
}

// run 打开窗口，跑起来。
func run(options options, desk desktop) error {
	store, err := config.Open()
	if err != nil {
		return err
	}
	return runWith(store, options, desk)
}

// runWith 用给定的配置存储打开窗口，跑起来。
func runWith(store *config.Store, options options, desk desktop) error {
	kit := build(store, options)
	defer kit.close()

	desk.SetAppearance(kit.appearance)

	desk.WhenReady(func() {
		win := desk.OpenWindow(mygo.WindowOptions{
			Title:     "Viper",
			Width:     1320,
			Height:    860,
			MinWidth:  980,
			MinHeight: 640,
			Content:   ui.View(kit.window.View),
		})

		if options.screenshot == "" {
			return
		}
		// 截屏：等曲线攒够点，再等第一帧画好，存成 PNG，然后退出。
		go func() {
			kit.waitForSamples(screenshotSamples)
			if err := capture.WaitAndSave(win, options.screenshot); err != nil {
				fmt.Fprintln(os.Stderr, "viperui:", err)
			}
			desk.Quit()
		}()
	})

	return desk.Run()
}

// screenshotSamples 是截屏前要等到的采样个数：曲线得填满一个窗口。
const screenshotSamples = 10

// waitForSamples 等后台攒够采样，最多等一段时间。
func (p *parts) waitForSamples(want int) {
	deadline := time.Now().Add(45 * time.Second)
	for p.ctrl.Samples() < want && time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
	}
}
