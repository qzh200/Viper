package pages_test

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/pages"
)

func TestPowerPageShowsReadingsAndControls(t *testing.T) {
	machine := &guitest.Hardware{
		Power:             domain.PowerPerformance,
		GPUModeState:      domain.GPUModeDiscrete,
		DiscreteGPUState:  true,
		LCDOverdriveState: true,
		WinKeyLockedState: true,
		MonitorData: domain.MonitorData{
			CPUTemperatureC:    75,
			CPUUsagePercent:    45.5,
			CPUFrequencyMHz:    4800,
			CPUMaxFrequencyMHz: 5200,
			GPUTemperatureC:    68,
			GPUUsagePercent:    80,
			GPUFrequencyMHz:    2350,
			GPUMaxFrequencyMHz: 2500,
			GPUIdle:            false,
			RAMUsagePercent:    60.0,
			SystemTempC:        55,
		},
		FansState: domain.FanState{
			CoolBoost: true,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Power(c, ctrl, state)
	}, 980, 680)

	for _, want := range []string{
		"性能与功耗调控",
		"系统电源与性能档位",
		"显卡输出与显示模式",
		"纯独显直连输出 (dGPU)",
		"独显核心活跃中",
		"电竞硬件加速与防护",
		"LCD 响应时间超速 (Overdrive)",
		"CoolBoost 强冷加速",
		"核心时钟与负载看板",
		"CPU 处理器",
		"GPU 独立显卡",
		"平衡",
		"性能",
		"狂暴",
	} {
		if !tt.HasText(want) {
			t.Errorf("power page missing text %q in %v", want, tt.Texts())
		}
	}
}

func TestPowerPageSwitchPowerMode(t *testing.T) {
	machine := &guitest.Hardware{
		Power: domain.PowerBalanced,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Power(c, ctrl, state)
	}, 980, 680)

	if err := tt.Click("狂暴"); err != nil {
		t.Fatal(err)
	}

	ctrl.Work()
	if len(machine.Writes) != 1 {
		t.Fatalf("expected 1 power write, got %d", len(machine.Writes))
	}
	if machine.Writes[0] != domain.PowerTurbo {
		t.Errorf("expected PowerTurbo mode, got %v", machine.Writes[0])
	}
}

func TestPowerPageSwitchGPUMode(t *testing.T) {
	machine := &guitest.Hardware{
		GPUModeState: domain.GPUModeHybrid,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Power(c, ctrl, state)
	}, 980, 680)

	if err := tt.Click("独显直连"); err != nil {
		t.Fatal(err)
	}

	ctrl.Work()
	if len(machine.GPUModeWrites) != 1 {
		t.Fatalf("expected 1 gpu mode write, got %d", len(machine.GPUModeWrites))
	}
	if machine.GPUModeWrites[0] != domain.GPUModeDiscrete {
		t.Errorf("expected GPUModeDiscrete, got %v", machine.GPUModeWrites[0])
	}
}

func TestPowerPageSwitches(t *testing.T) {
	machine := &guitest.Hardware{
		LCDOverdriveState: false,
		FansState: domain.FanState{
			CoolBoost: false,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Power(c, ctrl, state)
	}, 980, 680)

	if err := tt.Click("LCD 响应时间超速 (Overdrive)"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	if len(machine.LCDOverdriveWrites) != 1 || !machine.LCDOverdriveWrites[0] {
		t.Errorf("expected LCDOverdriveWrites true, got %v", machine.LCDOverdriveWrites)
	}

	if err := tt.Click("CoolBoost 强冷加速"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	if len(machine.CoolBoostWrites) != 1 || !machine.CoolBoostWrites[0] {
		t.Errorf("expected CoolBoostWrites true, got %v", machine.CoolBoostWrites)
	}
}

func TestPowerPageErrorNote(t *testing.T) {
	machine := &guitest.Hardware{
		Err: errors.New("原厂驱动异常"),
	}
	ctrl := controller.New(machine, &guitest.Store{})
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Power(c, ctrl, state)
	}, 980, 680)

	if !tt.HasText("原厂驱动异常") {
		t.Errorf("expected error note, got texts %v", tt.Texts())
	}
}
