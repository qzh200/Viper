package pages_test

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"viper/internal/domain"
	"viper/internal/gui/controller"
	"viper/internal/gui/guitest"
	"viper/internal/gui/pages"
)

func TestLightingPageGlobalSettings(t *testing.T) {
	machine := &guitest.Hardware{
		LightingConfig: domain.RGBConfig{
			Effect:     domain.EffectStatic,
			Speed:      5,
			Brightness: 80,
			PerZone:    true,
			Zones: [4]domain.RGBColor{
				{R: 0, G: 136, B: 255},
				{R: 0, G: 230, B: 118},
				{R: 255, G: 145, B: 0},
				{R: 213, G: 0, B: 249},
			},
		},
		BacklightTimeoutState: false,
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Lighting(c, ctrl, state)
	}, 1020, 680)

	// 验证全局设置页面基本文本
	for _, want := range []string{
		"键盘 RGB 灯效",
		"全局设置",
		"分区设置",
		"30s 自动休眠",
		"恢复默认",
		"全局运转灯效模式",
		"静态常亮",
		"呼吸渐变",
		"霓虹变色",
		"背光亮度",
		"动态参数",
		"动画速度",
		"流动方向",
	} {
		if !tt.HasText(want) {
			t.Errorf("lighting global page missing text %q in %v", want, tt.Texts())
		}
	}

	// 测试点击动态灯效模式（如“呼吸渐变”）
	if err := tt.Click("呼吸渐变"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()

	if len(machine.LightWrites) == 0 {
		t.Fatal("expected lighting write after clicking breathing effect")
	}
	lastWrite := machine.LightWrites[len(machine.LightWrites)-1]
	if lastWrite.Effect != domain.EffectBreathing {
		t.Errorf("expected EffectBreathing, got %v", lastWrite.Effect)
	}

	// 测试重置默认灯效
	if err := tt.Click("恢复默认"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	if machine.ResetLightCalls != 1 {
		t.Errorf("expected ResetLightCalls == 1, got %d", machine.ResetLightCalls)
	}
}

func TestLightingPagePerZoneSettings(t *testing.T) {
	machine := &guitest.Hardware{
		LightingConfig: domain.RGBConfig{
			Effect:     domain.EffectStatic,
			Speed:      5,
			Brightness: 100,
			PerZone:    true,
			Zones: [4]domain.RGBColor{
				{R: 0, G: 136, B: 255},
				{R: 0, G: 230, B: 118},
				{R: 255, G: 145, B: 0},
				{R: 213, G: 0, B: 249},
			},
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()
	state := pages.NewState()
	state.LightingTab = 1 // 切换到分区设置

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Lighting(c, ctrl, state)
	}, 1020, 680)

	// 验证分区设置页面基本文本
	for _, want := range []string{
		"四分区键盘色彩预览",
		"四分区独立调色",
		"分区 1",
		"分区 2",
		"分区 3",
		"分区 4",
		"色彩调控",
		"精选色盘",
		"掠夺蓝",
		"极光青",
		"烈焰红",
	} {
		if !tt.HasText(want) {
			t.Errorf("lighting per-zone page missing text %q in %v", want, tt.Texts())
		}
	}

	// 测试点击预设颜色“烈焰红”
	if err := tt.Click("烈焰红"); err != nil {
		t.Fatal(err)
	}
	// 等待防抖并执行
	time.Sleep(150 * time.Millisecond)
	ctrl.Work()

	if state.ZoneColors[state.SelectedZone].R != 255 ||
		state.ZoneColors[state.SelectedZone].G != 23 {
		t.Errorf("zone color not updated, got %+v", state.ZoneColors[state.SelectedZone])
	}
}
