package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"

	"viper/internal/config"
	"viper/internal/gui/theme"
)

// fakeDesktop 是一台不会真开窗口的桌面。
type fakeDesktop struct {
	// windows 记下开过哪几扇窗口。
	windows []mygo.WindowOptions
	// appearance 记着这一次用的是哪种外观。
	appearance theme.Mode
	// ran 记着主循环起没起。
	ran bool
	// quit 记着有没有让人退出。
	quit bool
}

func (d *fakeDesktop) WhenReady(fn func()) { fn() }

func (d *fakeDesktop) OpenWindow(options mygo.WindowOptions) *mygo.Window {
	d.windows = append(d.windows, options)
	return nil
}

func (d *fakeDesktop) SetAppearance(mode theme.Mode) { d.appearance = mode }

func (d *fakeDesktop) Run() error { d.ran = true; return nil }

func (d *fakeDesktop) Quit() { d.quit = true }

func TestThePartsAreWired(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}

	kit := build(store, options{})
	defer kit.close()

	if kit.application == nil {
		t.Error("there is no application")
	}
	if kit.ctrl == nil {
		t.Error("there is no controller")
	}
	if kit.window == nil {
		t.Error("there is no window")
	}
}

func TestTheSavedAppearanceIsUsed(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}

	kit := build(store, options{})
	defer kit.close()

	if kit.appearance != theme.ModeDark {
		t.Errorf("the appearance is %q, want the dark one", kit.appearance)
	}
}

func TestTheCommandLineWinsOverTheSavedAppearance(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}

	kit := build(store, options{theme: "light"})
	defer kit.close()

	if kit.appearance != theme.ModeLight {
		t.Errorf("the appearance is %q, want the light one", kit.appearance)
	}
	// 命令行定的那个不写回配置。
	if store.Theme() != "dark" {
		t.Errorf("the store holds %q, want the dark one it had", store.Theme())
	}
}

func TestTheWindowIsOpened(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	desk := &fakeDesktop{}

	if err := runWith(store, options{}, desk); err != nil {
		t.Fatal(err)
	}

	if !desk.ran {
		t.Error("the main loop never ran")
	}
	if len(desk.windows) != 1 {
		t.Fatalf("%d windows were opened, want one", len(desk.windows))
	}
	win := desk.windows[0]
	if win.Title != "Viper" {
		t.Errorf("the window is called %q, want Viper", win.Title)
	}
	if win.Content == nil {
		t.Error("the window has no content")
	}
}

func TestTheWindowAsksForTheAppearance(t *testing.T) {
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	desk := &fakeDesktop{}

	if err := runWith(store, options{theme: "dark"}, desk); err != nil {
		t.Fatal(err)
	}
	if len(desk.windows) != 1 {
		t.Fatalf("%d windows were opened, want one", len(desk.windows))
	}
	if desk.appearance != theme.ModeDark {
		t.Errorf("the appearance is %q, want the dark one", desk.appearance)
	}
}
