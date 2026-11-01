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

type desktop interface {
	WhenReady(fn func())
	OpenWindow(options mygo.WindowOptions) *mygo.Window
	SetAppearance(mode theme.Mode)
	Run() error
	Quit()
}

type mygoDesktop struct{}

func (mygoDesktop) WhenReady(fn func()) { mygo.App.WhenReady(fn) }
func (mygoDesktop) OpenWindow(options mygo.WindowOptions) *mygo.Window {
	return mygo.NewWindow(options)
}

func (mygoDesktop) SetAppearance(mode theme.Mode) {
	mygo.Theme.SetSource(shell.Source(mode))
}

func (mygoDesktop) Run() error { return mygo.App.Run() }
func (mygoDesktop) Quit()      { mygo.App.Quit() }

type parts struct {
	store       *config.Store
	application *app.App
	ctrl        *controller.Controller
	window      *shell.Shell
	appearance  theme.Mode
}

func build(store *config.Store, options options) *parts {
	application := app.NewWithStore(store)
	ctrl := controller.New(application, store)

	appearance := ctrl.ThemeMode()
	if options.theme != "" {
		appearance = theme.Mode(options.theme)
	}

	opts := []shell.Option{shell.WithAppearance(appearance)}
	if options.page != "" {
		opts = append(opts, shell.WithPage(options.page))
	}

	return &parts{
		store:       store,
		application: application,
		ctrl:        ctrl,
		window:      shell.New(ctrl, opts...),
		appearance:  appearance,
	}
}

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

func run(options options, desk desktop) error {
	store, err := config.Open()
	if err != nil {
		return err
	}
	return runWith(store, options, desk)
}

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

const screenshotSamples = 10

func (p *parts) waitForSamples(want int) {
	deadline := time.Now().Add(45 * time.Second)
	for p.ctrl.Samples() < want && time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
	}
}
