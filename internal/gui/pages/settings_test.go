package pages_test

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/pages"
	"viper/internal/gui/theme"
)

func TestSettingsPageShowsReadingsAndControls(t *testing.T) {
	machine := &guitest.Hardware{
		InstanceName:          "AcerAgentService",
		RouteName:             "TCP 127.0.0.1:46933",
		BootSoundState:        true,
		BacklightTimeoutState: true,
	}
	store := &guitest.Store{Mode: "auto"}
	ctrl := controller.New(machine, store)
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Settings(c, ctrl, state, func(theme.Mode) {})
	}, 980, 680)

	for _, want := range []string{
		"系统与控制中心设置",
		"原厂服务正常运行",
		"界面外观与主题偏好",
		"跟随系统",
		"浅色模式",
		"深色模式",
		"原厂硬件特色调校",
		"开机专属问候音效 (Boot Sound)",
		"键盘背光 30 秒空闲自动休眠",
		"驱动服务与连接状态",
		"AcerAgentService",
		"TCP 127.0.0.1:46933",
		"config.json (原子写保护)",
		"关于",
		"Viper 控制中心",
	} {
		if !tt.HasText(want) {
			t.Errorf("settings page missing text %q in %v", want, tt.Texts())
		}
	}
}

func TestSettingsPageSwitchAppearance(t *testing.T) {
	machine := &guitest.Hardware{}
	store := &guitest.Store{Mode: "auto"}
	ctrl := controller.New(machine, store)
	state := pages.NewState()

	var appliedTheme theme.Mode
	tt := ui.NewTester(func(c *ui.Context) {
		pages.Settings(c, ctrl, state, func(m theme.Mode) {
			appliedTheme = m
		})
	}, 980, 680)

	// 点击“深色模式”
	if err := tt.Click("深色模式"); err != nil {
		t.Fatal(err)
	}

	if appliedTheme != theme.ModeDark {
		t.Errorf("expected appliedTheme dark, got %v", appliedTheme)
	}
	if len(store.Saved) != 1 || store.Saved[0] != "dark" {
		t.Errorf("expected store saved 'dark', got %v", store.Saved)
	}
}

func TestSettingsPageBootSoundToggle(t *testing.T) {
	machine := &guitest.Hardware{
		BootSoundState: false,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Settings(c, ctrl, state, nil)
	}, 980, 680)

	// 点击开机音效开关
	if err := tt.Click("开机专属问候音效 (Boot Sound)"); err != nil {
		t.Fatal(err)
	}

	ctrl.Work()
	if len(machine.BootSoundWrites) != 1 || !machine.BootSoundWrites[0] {
		t.Errorf("expected BootSoundWrite true, got %v", machine.BootSoundWrites)
	}
}

func TestSettingsPageBacklightTimeoutToggle(t *testing.T) {
	machine := &guitest.Hardware{
		BacklightTimeoutState: false,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Settings(c, ctrl, state, nil)
	}, 980, 680)

	// 点击背光休眠开关
	if err := tt.Click("键盘背光 30 秒空闲自动休眠"); err != nil {
		t.Fatal(err)
	}

	ctrl.Work()
	if len(machine.TimeoutWrites) != 1 || !machine.TimeoutWrites[0] {
		t.Errorf("expected TimeoutWrites true, got %v", machine.TimeoutWrites)
	}
}

func TestSettingsPageErrorNote(t *testing.T) {
	machine := &guitest.Hardware{
		Err: errors.New("原厂服务连接超时"),
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Settings(c, ctrl, state, nil)
	}, 980, 680)

	if !tt.HasText("原厂服务连接超时") {
		t.Errorf("expected error note, got texts %v", tt.Texts())
	}
}
