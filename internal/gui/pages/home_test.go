package pages_test

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/pages"
)

func TestTheHomePageShowsTheReadings(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData:  domain.MonitorData{CPUTemperatureC: 72, RAMTotalMB: 16384},
		Power:        domain.PowerBalanced,
		InstanceName: "AcerAgentService",
		RouteName:    "原厂服务 AcerAgentService",
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, state)
	}, 980, 620)

	for _, want := range []string{
		"已连接",
		"原厂服务 AcerAgentService",
		"CPU", "GPU",
		"温度", "风扇转速",
		"内存", "主板温度", "网络",
		"温度走势",
		"性能档位",
		"狂暴",
	} {
		if !tt.HasText(want) {
			t.Errorf("the home page has no %q in %q", want, tt.Texts())
		}
	}
}

func TestTheHomePageSaysWhenTheMachineIsSilent(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{Err: guitest.ErrHardware}, &guitest.Store{})

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, pages.NewState())
	}, 980, 620)

	if !tt.HasText("还没有数据") {
		t.Errorf("the home page has no hint in %q", tt.Texts())
	}
	if !tt.HasText("确认原厂服务 AcerAgentService 正在运行") {
		t.Errorf("the home page does not say what to do: %q", tt.Texts())
	}
	if !tt.HasText("未连接") {
		t.Errorf("the home page does not say it is not connected: %q", tt.Texts())
	}
}

func TestTheHomePageFollowsTheAppearance(t *testing.T) {
	ctrl := controller.New(&guitest.Hardware{
		MonitorData: domain.MonitorData{CPUTemperatureC: 55},
	}, &guitest.Store{})

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, pages.NewState())
	}, 980, 620)

	tt.SetDark(true)
	tt.Frame()

	if !tt.HasText("CPU") {
		t.Errorf("the dark appearance lost the readings: %q", tt.Texts())
	}
}

func TestChoosingAPowerModeQueuesTheWrite(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData: domain.MonitorData{CPUTemperatureC: 50},
		Power:       domain.PowerBalanced,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, state)
	}, 980, 620)

	if err := tt.Click("狂暴"); err != nil {
		t.Fatal(err)
	}

	if state.PowerIndex < 0 {
		t.Fatal("the page forgot the choice")
	}
	if len(machine.Writes) != 0 {
		t.Fatal("the machine was written to before the worker ran")
	}

	ctrl.Work()

	if len(machine.Writes) != 1 {
		t.Fatalf("the machine was written to %d times, want once", len(machine.Writes))
	}
	if machine.Writes[0] != domain.PowerTurbo {
		t.Errorf("the machine was written to %q, want the turbo mode", machine.Writes[0])
	}
}

func TestTheChoiceIsKeptUntilTheMachineAgrees(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData: domain.MonitorData{CPUTemperatureC: 50},
		Power:       domain.PowerBalanced,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, state)
	}, 980, 620)

	if err := tt.Click("狂暴"); err != nil {
		t.Fatal(err)
	}
	if state.PowerIndex < 0 {
		t.Fatal("the page forgot the choice")
	}

	// The machine did it: the page may hand the choice back.
	ctrl.Work()
	tt.Frame()

	if state.PowerIndex != -1 {
		t.Errorf("the page still holds the choice %d, want it handed back", state.PowerIndex)
	}
}

func TestAFailedWriteLetsGoOfTheChoice(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData: domain.MonitorData{CPUTemperatureC: 50},
		Power:       domain.PowerBalanced,
		SetErr:      guitest.ErrHardware,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Home(c, ctrl, state)
	}, 980, 620)

	if err := tt.Click("狂暴"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	tt.Frame()

	if state.PowerIndex != -1 {
		t.Errorf("the page still holds the choice %d after a failed write", state.PowerIndex)
	}
}

func TestANewStateHoldsNoChoice(t *testing.T) {
	if state := pages.NewState(); state.PowerIndex != -1 {
		t.Errorf("a new state holds %d, want no choice", state.PowerIndex)
	}
}
