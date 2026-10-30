package main

import (
	"strings"
	"testing"

	"viper/internal/domain"
)

func TestStatusShowsTheReadings(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "status")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"instance:",
		"route  :",
		"CPU temperature   : 87 C",
		"GPU temperature   : 56 C",
		"CPU fan speed     : 2681 RPM",
		"Core clocks       : 2 cores, 1671 to 3408 MHz",
		"Memory            : 43.1 % of 32488 MB @ 5600 MHz",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

func TestTelemetry(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "telemetry")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CPU temperature   : 87 C") {
		t.Errorf("遥测里没有温度:\n%s", out)
	}
}

func TestFanCommands(t *testing.T) {
	machine, _ := installFakes(t)

	out, _, err := capture(t, "fans")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "CPU fan    : auto") {
		t.Errorf("风扇读数不对:\n%s", out)
	}

	if _, _, err := capture(t, "fan", "max"); err != nil {
		t.Fatal(err)
	}
	if machine.fans.CPUMode != domain.FanMax || machine.fans.GPUMode != domain.FanMax {
		t.Errorf("全速没写进去: %#v", machine.fans)
	}

	if _, _, err := capture(t, "fan", "custom", "50", "40"); err != nil {
		t.Fatal(err)
	}
	if machine.fans.CPUMode != domain.FanCustom || machine.fans.CPUPercent != 50 {
		t.Errorf("固定占空比没写进去: %#v", machine.fans)
	}
	if machine.fans.GPUPercent != 40 {
		t.Errorf("显卡风扇的占空比不对: %#v", machine.fans)
	}

	if _, _, err := capture(t, "fan", "auto"); err != nil {
		t.Fatal(err)
	}
	if machine.fans.CPUMode != domain.FanAuto {
		t.Errorf("自动没写进去: %#v", machine.fans)
	}

	if _, _, err := capture(t, "fan", "coolboost", "on"); err != nil {
		t.Fatal(err)
	}
	if !machine.fans.CoolBoost {
		t.Errorf("加速没写进去: %#v", machine.fans)
	}
}

func TestPowerCommand(t *testing.T) {
	machine, _ := installFakes(t)

	out, _, err := capture(t, "power")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "balanced") {
		t.Errorf("档位读数不对:\n%s", out)
	}

	if _, _, err := capture(t, "power", "performance"); err != nil {
		t.Fatal(err)
	}
	if machine.power != domain.PowerPerformance {
		t.Errorf("档位没写进去: %v", machine.power)
	}
}

func TestLightingCommand(t *testing.T) {
	machine, _ := installFakes(t)

	out, _, err := capture(t, "lighting")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "effect      : wave") {
		t.Errorf("灯效读数不对:\n%s", out)
	}

	before := machine.lighting.Zones
	if _, _, err := capture(t, "lighting", "set", "brightness", "4"); err != nil {
		t.Fatal(err)
	}
	if machine.lighting.Brightness != 4 {
		t.Errorf("亮度没写进去: %d", machine.lighting.Brightness)
	}
	if machine.lighting.Zones != before {
		t.Errorf("颜色被改了: %#v", machine.lighting.Zones)
	}
	if machine.lighting.Effect != domain.EffectWave {
		t.Errorf("效果被改了: %v", machine.lighting.Effect)
	}

	if _, _, err := capture(t, "lighting", "set", "zone", "2", "#00FF00"); err != nil {
		t.Fatal(err)
	}
	if !machine.lighting.PerZone {
		t.Errorf("点过分区就该是分区设置: %#v", machine.lighting)
	}
	if machine.lighting.Zones[1] != (domain.RGBColor{G: 0xFF}) {
		t.Errorf("第二个分区的颜色不对: %#v", machine.lighting.Zones[1])
	}
	if machine.lighting.Zones[0] != before[0] {
		t.Errorf("没点到的分区被改了: %#v", machine.lighting.Zones)
	}
}

func TestLightingReset(t *testing.T) {
	machine, _ := installFakes(t)

	if _, _, err := capture(t, "lighting", "reset"); err != nil {
		t.Fatal(err)
	}
	if machine.lighting.Effect != domain.EffectStatic {
		t.Errorf("复位之后效果不对: %v", machine.lighting.Effect)
	}
	if machine.lighting.Brightness != 100 {
		t.Errorf("复位之后亮度不对: %d", machine.lighting.Brightness)
	}
}

func TestProfileCommand(t *testing.T) {
	machine, _ := installFakes(t)

	out, _, err := capture(t, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "LCD overdrive") {
		t.Errorf("开关的读数不对:\n%s", out)
	}

	if _, _, err := capture(t, "profile", "overdrive", "on"); err != nil {
		t.Fatal(err)
	}
	if !machine.overdrive {
		t.Error("屏幕超频没写进去")
	}
}

func TestGPUCommand(t *testing.T) {
	machine, _ := installFakes(t)

	out, _, err := capture(t, "gpu")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "discrete") {
		t.Errorf("显卡模式的读数不对:\n%s", out)
	}

	if _, _, err := capture(t, "gpu", "hybrid"); err != nil {
		t.Fatal(err)
	}
	if machine.gpu != domain.GPUModeHybrid {
		t.Errorf("显卡模式没写进去: %v", machine.gpu)
	}
}

func TestQuickAccessCommands(t *testing.T) {
	_, quick := installFakes(t)

	out, _, err := capture(t, "charge")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "30%") {
		t.Errorf("充电上限的读数不对:\n%s", out)
	}
	if _, _, err := capture(t, "charge", "limit", "20"); err != nil {
		t.Fatal(err)
	}
	if quick.chargeLimit != 20 {
		t.Errorf("充电上限没写进去: %d", quick.chargeLimit)
	}
	if _, _, err := capture(t, "charge", "usb", "off"); err != nil {
		t.Fatal(err)
	}
	if quick.chargeUSB {
		t.Error("USB 充电没关掉")
	}

	if _, _, err := capture(t, "darkmode", "on"); err != nil {
		t.Fatal(err)
	}
	if !quick.dark {
		t.Error("暗色模式没写进去")
	}
	if _, _, err := capture(t, "transparent", "on"); err != nil {
		t.Fatal(err)
	}
	if !quick.transparent {
		t.Error("透明效果没写进去")
	}
	if _, _, err := capture(t, "taskbar", "on"); err != nil {
		t.Fatal(err)
	}
	if !quick.taskbar {
		t.Error("任务栏自动隐藏没写进去")
	}
	if _, _, err := capture(t, "screensaver", "on"); err != nil {
		t.Fatal(err)
	}
	if !quick.saver {
		t.Error("屏保没写进去")
	}
	if _, _, err := capture(t, "screensaver", "timeout", "300"); err != nil {
		t.Fatal(err)
	}
	if quick.saverTimeout != 300 {
		t.Errorf("屏保超时没写进去: %d", quick.saverTimeout)
	}
}

func TestBootSoundCommand(t *testing.T) {
	machine, _ := installFakes(t)

	if _, _, err := capture(t, "bootsound", "on"); err != nil {
		t.Fatal(err)
	}
	if !machine.bootSound {
		t.Error("开机音效没写进去")
	}
}

func TestScenarioCommands(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "scenario", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no scene is saved") {
		t.Errorf("还没有场景的时候该这么说:\n%s", out)
	}

	if _, _, err := capture(t, "scenario", "save", "game"); err != nil {
		t.Fatal(err)
	}
	out, _, err = capture(t, "scenario", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "game") {
		t.Errorf("场景没列出来:\n%s", out)
	}

	if _, _, err := capture(t, "scenario", "bind", "performance", "game"); err != nil {
		t.Fatal(err)
	}
	out, _, err = capture(t, "scenario", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bound scenes") {
		t.Errorf("绑定没列出来:\n%s", out)
	}

	if _, _, err := capture(t, "scenario", "apply", "game"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := capture(t, "scenario", "unbind", "performance"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := capture(t, "scenario", "delete", "game"); err != nil {
		t.Fatal(err)
	}

	out, _, err = capture(t, "scenario", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no scene is saved") {
		t.Errorf("场景删掉了还不该列出来:\n%s", out)
	}
}

func TestDiagReportsTheCapabilities(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "diag")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"CPU fan         : auto",
		"graphics mode   : discrete",
		"boot sound      : off",
		"settings        :",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("报告里没有 %q:\n%s", want, out)
		}
	}
}
