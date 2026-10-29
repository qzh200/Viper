package pages_test

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/pages"
)

func TestFansPageShowsReadingsAndControls(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData: domain.MonitorData{
			CPUTemperatureC: 68,
			GPUTemperatureC: 60,
			CPUFanRPM:       2200,
			GPUFanRPM:       2400,
		},
		FansState: domain.FanState{
			CPUMode:   domain.FanAuto,
			GPUMode:   domain.FanAuto,
			CoolBoost: false,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Fans(c, ctrl, state)
	}, 980, 620)

	for _, want := range []string{
		"风扇与散热控制",
		"CPU 散热风扇",
		"GPU 散热风扇",
		"CoolBoost 强冷",
		"自动", "全速", "自定义",
		"散热模式与策略指引",
	} {
		if !tt.HasText(want) {
			t.Errorf("fans page missing text %q in %v", want, tt.Texts())
		}
	}
}

func TestFansPageSwitchMode(t *testing.T) {
	machine := &guitest.Hardware{
		FansState: domain.FanState{
			CPUMode:   domain.FanAuto,
			GPUMode:   domain.FanAuto,
			CoolBoost: false,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Fans(c, ctrl, state)
	}, 980, 620)

	// 点击“全速”
	if err := tt.Click("全速"); err != nil {
		t.Fatal(err)
	}

	if state.FanModeIndex != 1 {
		t.Errorf("state.FanModeIndex is %d, want 1", state.FanModeIndex)
	}

	// 此时应该排队了一次写入
	ctrl.Work()
	if len(machine.FanWrites) != 1 {
		t.Fatalf("expected 1 fan write, got %d", len(machine.FanWrites))
	}
	if machine.FanWrites[0].CPUMode != domain.FanMax {
		t.Errorf("expected FanMax mode, got %v", machine.FanWrites[0].CPUMode)
	}
}

func TestFansPageCustomModeShowsSliders(t *testing.T) {
	machine := &guitest.Hardware{
		FansState: domain.FanState{
			CPUMode:    domain.FanCustom,
			GPUMode:    domain.FanCustom,
			CPUPercent: 75,
			GPUPercent: 80,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Fans(c, ctrl, state)
	}, 980, 620)

	if !tt.HasText("占空比") {
		t.Error("custom mode should show duty cycle slider labels")
	}

	// 验证初始化读入
	if state.CPUDuty != 75 || state.GPUDuty != 80 {
		t.Errorf("expected duties 75 and 80, got CPU=%f, GPU=%f", state.CPUDuty, state.GPUDuty)
	}

	// 测试滑到 0% 时绝不被弹回
	state.CPUDuty = 0
	tt.Frame()
	if !tt.HasText("0%") {
		t.Errorf("slider at 0%% should show 0%% text")
	}
	if state.CPUDuty != 0 {
		t.Errorf("duty should remain 0, got %f", state.CPUDuty)
	}
}
