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
			Brightness: 5, // 对应 UI 4 档
			PerZone:    false,
			Zones: [4]domain.RGBColor{
				{R: 0, G: 136, B: 255},
				{R: 0, G: 136, B: 255},
				{R: 0, G: 136, B: 255},
				{R: 0, G: 136, B: 255},
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

	// 1. 静态模式下检查基本文本与选色板
	for _, want := range []string{
		"键盘 RGB 灯效",
		"全局设置",
		"分区设置",
		"30s 自动休眠",
		"恢复默认",
		"全局运转灯效模式",
		"静态",
		"呼吸",
		"霓虹",
		"波形",
		"切换",
		"缩放",
		"流星雨",
		"闪烁",
		"背光亮度",
		"静态全局色彩调控",
		"色相",
		"HEX",
		"掠夺蓝",
		"极光青",
		"烈焰红",
	} {
		if !tt.HasText(want) {
			t.Errorf("lighting global page in static mode missing text %q in %v", want, tt.Texts())
		}
	}

	// 策略指引卡片已被移除
	if tt.HasText("键盘背光与氛围策略指引") {
		t.Error("strategy card text should be removed")
	}

	// 静态模式不应显示动画速度滑块
	if tt.HasText("动画速度") {
		t.Error("static mode should not display speed slider")
	}

	// 验证色相滑块
	if !tt.HasText("色相") {
		t.Error("should show Hue slider")
	}

	// 2. 点击切换到“呼吸”模式（支持调速与调色）
	if err := tt.Click("呼吸"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	tt.Frame()
	if len(machine.LightWrites) == 0 {
		t.Fatal("expected lighting write after clicking breathing")
	}
	if machine.LightingConfig.Effect != domain.EffectBreathing {
		t.Errorf("expected EffectBreathing, got %v", machine.LightingConfig.Effect)
	}
	if !tt.HasText("动态参数") || !tt.HasText("动画速度") {
		t.Errorf("breathing mode should display dynamic speed parameters, texts: %v", tt.Texts())
	}
	if !tt.HasText("灯效色彩调控") {
		t.Error("breathing mode should allow color adjustment")
	}

	// 测试点击档位直选“极速 (9档)”
	if err := tt.Click("极速 (9档)"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	if state.LightingSpeed != 9 {
		t.Errorf("expected speed 9, got %v", state.LightingSpeed)
	}

	// 3. 点击切换到“波形”模式（支持流动方向与速度，不支持调色）
	if err := tt.Click("波形"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	tt.Frame()
	if !tt.HasText("流动方向") {
		t.Error("wave mode should offer flow direction")
	}
	if tt.HasText("灯效色彩调控") {
		t.Error("wave mode should not allow color adjustment")
	}

	// 4. 点击切换到“切换”模式（支持流动方向、调速、调色）
	if err := tt.Click("切换"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	tt.Frame()
	if !tt.HasText("流动方向") {
		t.Error("switch mode should offer flow direction")
	}
	if !tt.HasText("灯效色彩调控") {
		t.Error("switch mode should allow color adjustment")
	}

	// 5. 点击预设色彩“极光青”
	if err := tt.Click("极光青"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	ctrl.Work()
	if state.ZoneColors[0].G != 230 {
		t.Errorf("expected color green 230, got %d", state.ZoneColors[0].G)
	}

	// 6. 测试重置默认灯效
	if err := tt.Click("恢复默认"); err != nil {
		t.Fatal(err)
	}
	ctrl.Work()
	if machine.ResetLightCalls != 1 {
		t.Errorf("expected ResetLightCalls == 1, got %d", machine.ResetLightCalls)
	}
	if state.LightingBrightness != 4 {
		t.Errorf("expected reset brightness 4, got %v", state.LightingBrightness)
	}
}

func TestLightingPagePerZoneSettings(t *testing.T) {
	machine := &guitest.Hardware{
		LightingConfig: domain.RGBConfig{
			Effect:     domain.EffectStatic,
			Speed:      5,
			Brightness: 5,
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
	time.Sleep(150 * time.Millisecond)
	ctrl.Work()

	if state.ZoneColors[state.SelectedZone].R != 255 ||
		state.ZoneColors[state.SelectedZone].G != 23 {
		t.Errorf("zone color not updated, got %+v", state.ZoneColors[state.SelectedZone])
	}
}

func TestLightingPageNeonAndZeroBrightness(t *testing.T) {
	machine := &guitest.Hardware{
		LightingConfig: domain.RGBConfig{
			Effect:     domain.EffectNeon,
			Speed:      7,
			Brightness: 1, // 硬件 1 档即对应界面 0 档（熄灭）
			PerZone:    false,
		},
	}
	ctrl := controller.New(machine, &guitest.Store{})
	defer ctrl.Stop()
	state := pages.NewState()

	tt := ui.NewTester(func(c *ui.Context) {
		pages.Lighting(c, ctrl, state)
	}, 1020, 680)

	// 验证 0 档熄灭文字展示
	if !tt.HasText("0 档") {
		t.Errorf("expected 0 档 text for brightness 1, got %v", tt.Texts())
	}
	if !tt.HasText("已熄灭") {
		t.Errorf("expected 已熄灭 badge, got %v", tt.Texts())
	}

	// 验证霓虹模式下的动态参数（有动画速度、无流动方向、无调色盘）
	if !tt.HasText("动画速度") {
		t.Error("neon should show animation speed")
	}
	if tt.HasText("流动方向") {
		t.Error("neon should not offer flow direction")
	}
	if tt.HasText("灯效色彩调控") {
		t.Error("neon should not allow color adjustment")
	}
}
