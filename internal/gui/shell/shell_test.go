package shell_test

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/shell"
	"viper/internal/gui/theme"
)

func TestTheWindowShowsTheNavigationAndTheHeader(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData:  domain.MonitorData{CPUTemperatureC: 61},
		Power:        domain.PowerBalanced,
		InstanceName: "AcerAgentService",
		RouteName:    "原厂服务 AcerAgentService",
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	for _, want := range []string{
		"Viper",
		"Acer 游戏本控制中心",
		"外观",
		"自动", "亮色", "暗色",
		"总览", "风扇", "灯效", "性能", "设置",
		"CPU", "GPU",
	} {
		if !tt.HasText(want) {
			t.Errorf("the window has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheWindowFollowsTheAppearance(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{
		MonitorData: domain.MonitorData{CPUTemperatureC: 50},
	}, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	tt.SetDark(true)
	tt.Frame()

	if !tt.HasText("CPU") {
		t.Errorf("the dark appearance lost the readings: %q", tt.Texts())
	}
}

func TestChoosingAnAppearanceSavesItAndAppliesIt(t *testing.T) {
	store := &guitest.Store{}
	ctrl := controller.New(&guitest.Hardware{}, store)
	defer ctrl.Stop()

	var applied theme.Mode
	window := shell.New(ctrl, shell.WithApply(func(mode theme.Mode) { applied = mode }))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if err := tt.Click("暗色"); err != nil {
		t.Fatal(err)
	}

	if applied != theme.ModeDark {
		t.Errorf("the applied mode is %q, want the dark one", applied)
	}
	if store.Mode != "dark" {
		t.Errorf("the store holds %q, want dark", store.Mode)
	}
}

func TestAFailureToSaveLeavesTheAppearanceAlone(t *testing.T) {
	store := &guitest.Store{Err: errStore}
	ctrl := controller.New(&guitest.Hardware{}, store)
	defer ctrl.Stop()

	applied := false
	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) { applied = true }))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if err := tt.Click("暗色"); err != nil {
		t.Fatal(err)
	}

	if applied {
		t.Error("the appearance was changed although the mode could not be saved")
	}
}

func TestTheSavedAppearanceIsTheOneTheWindowStartsWith(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{Mode: "dark"})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))
	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if !tt.HasText("暗色") {
		t.Errorf("the window has no %q in %q", "暗色", tt.Texts())
	}
}

func TestThePagesAreTheOnesTheSidebarOffers(t *testing.T) {
	if len(shell.Pages) == 0 {
		t.Fatal("there are no pages")
	}
	seen := map[string]bool{}
	for _, page := range shell.Pages {
		if page.ID == "" || page.Label == "" {
			t.Errorf("the page %+v has no name", page)
		}
		if seen[page.ID] {
			t.Errorf("the page %q is listed twice", page.ID)
		}
		seen[page.ID] = true
	}
	if !seen["home"] {
		t.Error("there is no home page")
	}
}

func TestAnotherPageShowsThatItIsNotThereYet(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}), shell.WithPage("unknown"))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if !tt.HasText("这一页还在路上") {
		t.Errorf("the window has no %q in %q", "这一页还在路上", tt.Texts())
	}
}

func TestNavigatingToSettingsPage(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("系统与控制中心设置") {
		t.Errorf("the window has no settings page title in %q", tt.Texts())
	}
}

func TestNavigatingToLightingPage(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if err := tt.Click("灯效"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("键盘 RGB 灯效") {
		t.Errorf("the window has no lighting page title in %q", tt.Texts())
	}
}

func TestNavigatingToFansPage(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{})
	defer ctrl.Stop()

	window := shell.New(ctrl, shell.WithApply(func(theme.Mode) {}))

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if err := tt.Click("风扇"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("风扇与散热控制") {
		t.Errorf("the window has no fans page title in %q", tt.Texts())
	}

	if err := tt.Click("性能"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("性能与功耗调控") {
		t.Errorf("the window has no power page title in %q", tt.Texts())
	}
}

func TestTheSourceOfEachAppearance(t *testing.T) {
	cases := map[theme.Mode]mygo.ThemeSource{
		theme.ModeAuto:  mygo.ThemeSystem,
		theme.ModeLight: mygo.ThemeLight,
		theme.ModeDark:  mygo.ThemeDark,
		theme.Mode("x"): mygo.ThemeSystem,
	}
	for mode, want := range cases {
		if got := shell.Source(mode); got != want {
			t.Errorf("the source of %q is %q, want %q", mode, got, want)
		}
	}
}

func TestAnAppearanceCanBeGivenToTheWindow(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{}, &guitest.Store{Mode: "auto"})
	defer ctrl.Stop()

	window := shell.New(ctrl,
		shell.WithApply(func(theme.Mode) {}),
		shell.WithAppearance(theme.ModeDark),
	)

	tt := ui.NewTester(func(c *ui.Context) {
		window.View(c)
	}, 1120, 720)

	if !tt.HasText("暗色") {
		t.Errorf("the window has no %q in %q", "暗色", tt.Texts())
	}
}

var errStore = errSave{}

type errSave struct{}

func (errSave) Error() string { return "磁盘满了" }
