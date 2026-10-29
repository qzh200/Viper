package main

import (
	"bytes"
	"strings"
	"testing"

	"viper/internal/app"
	"viper/internal/hardware"
)

func capture(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	oldOut, oldErr := stdout, stderr
	stdout, stderr = &outBuf, &errBuf
	defer func() {
		stdout, stderr = oldOut, oldErr
	}()

	err := run(args)
	return outBuf.String(), errBuf.String(), err
}

func TestUsageIsPrintedForEveryFormOfHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		out, _, err := capture(t, args...)
		if err != nil {
			t.Fatalf("%v: 用法说明不该报错: %v", args, err)
		}
		if !strings.Contains(out, "viper status") {
			t.Errorf("%v: 输出里没有用法说明:\n%s", args, out)
		}
	}
}

func TestUnknownOptionIsRejected(t *testing.T) {
	_, _, err := capture(t, "--nope")
	if err == nil {
		t.Fatal("不认识的选项该报错")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Errorf("错误信息不对: %v", err)
	}
}

func TestServiceOptionNeedsAnAddress(t *testing.T) {
	_, _, err := capture(t, "--service")
	if err == nil {
		t.Fatal("--service 少了地址该报错")
	}
	if !strings.Contains(err.Error(), "needs an address") {
		t.Errorf("错误信息不对: %v", err)
	}
}

func TestUnknownCommandIsRejected(t *testing.T) {
	out, _, err := capture(t, "nosuchcommand")
	if err == nil {
		t.Fatal("不认识的命令该报错")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("错误信息不对: %v", err)
	}
	if !strings.Contains(out, "viper status") {
		t.Errorf("不认识的命令要把用法说明也打出来:\n%s", out)
	}
}

func TestBadArgumentsAreRejectedBeforeTheMachine(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"性能档位", []string{"power", "bogus"}, "unknown performance profile"},
		{"显卡模式", []string{"gpu", "bogus"}, "unknown graphics mode"},
		{"风扇子命令", []string{"fan", "bogus"}, "unknown fan command"},
		{"风扇占空比", []string{"fan", "custom", "x", "y"}, "parsing"},
		{"灯效名", []string{"lighting", "set", "effect", "bogus"}, "unknown lighting effect"},
		{"灯效缺值", []string{"lighting", "set", "effect"}, "needs a value"},
		{"分区越界", []string{"lighting", "set", "zone", "9", "#FFFFFF"}, "out of range"},
		{"灯效选项", []string{"lighting", "set", "bogus"}, "unknown lighting option"},
		{"充电上限", []string{"charge", "limit", "abc"}, "parsing"},
		{"充电子命令", []string{"charge", "bogus"}, "unknown charge command"},
		{"开关值", []string{"profile", "overdrive", "maybe"}, "expected on or off"},
		{"配置子命令", []string{"profile", "bogus"}, "unknown profile command"},
		{"显示模式", []string{"displaymode", "bogus"}, "unknown display mode"},
		{"暗色模式", []string{"darkmode", "maybe"}, "expected on or off"},
		{"任务栏", []string{"taskbar", "maybe"}, "expected on or off"},
		{"透明效果", []string{"transparent", "maybe"}, "expected on or off"},
		{"屏保", []string{"screensaver", "maybe"}, "expected on or off"},
		{"屏保超时", []string{"screensaver", "timeout", "-5"}, "cannot be negative"},
		{"开机音效", []string{"bootsound", "maybe"}, "expected on or off"},
		{"场景子命令", []string{"scenario", "bogus"}, "unknown scenario command"},
		{"场景绑档位", []string{"scenario", "bind", "bogus", "name"}, "unknown performance profile"},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			_, _, err := capture(t, item.args...)
			if err == nil {
				t.Fatalf("%v 该报错", item.args)
			}
			if !strings.Contains(err.Error(), item.want) {
				t.Errorf("%v 错误信息里没有 %q: %v", item.args, item.want, err)
			}
		})
	}
}

func TestParseOnOff(t *testing.T) {
	for _, word := range []string{"on", "true", "1", "yes", "enable", "ON"} {
		value, err := parseOnOff(word)
		if err != nil {
			t.Errorf("%q: %v", word, err)
		}
		if !value {
			t.Errorf("%q 该是真", word)
		}
	}
	for _, word := range []string{"off", "false", "0", "no", "disable", "OFF"} {
		value, err := parseOnOff(word)
		if err != nil {
			t.Errorf("%q: %v", word, err)
		}
		if value {
			t.Errorf("%q 该是假", word)
		}
	}
	if _, err := parseOnOff("maybe"); err == nil {
		t.Error("看不懂的词该报错")
	}
}

func TestParseColor(t *testing.T) {
	colour, err := parseColor("#FF8800")
	if err != nil {
		t.Fatal(err)
	}
	if colour.R != 255 || colour.G != 136 || colour.B != 0 {
		t.Errorf("颜色不对: %#v", colour)
	}

	colour, err = parseColor("00FF00")
	if err != nil {
		t.Fatal(err)
	}
	if colour.R != 0 || colour.G != 255 || colour.B != 0 {
		t.Errorf("颜色不对: %#v", colour)
	}

	for _, text := range []string{"#FFF", "FFFFFF00", "GGGGGG", ""} {
		if _, err := parseColor(text); err == nil {
			t.Errorf("%q 该报错", text)
		}
	}
}

func lightingBase() hardware.RGBConfig {
	return hardware.RGBConfig{
		Effect:     hardware.EffectWave,
		Speed:      4,
		Direction:  1,
		Brightness: 60,
		PerZone:    true,
		Zones: [4]hardware.RGBColor{
			{R: 255, G: 0, B: 0},
			{G: 128},
			{B: 64},
			{R: 10, G: 20, B: 30},
		},
	}
}

func TestLightingOptionsKeepTheBase(t *testing.T) {
	base := lightingBase()

	config, err := parseLightingOptions(base, []string{"brightness", "80"})
	if err != nil {
		t.Fatal(err)
	}
	if config.Brightness != 80 {
		t.Errorf("亮度是 %d，不是 80", config.Brightness)
	}
	if config.Effect != base.Effect || config.Speed != base.Speed || config.Direction != base.Direction {
		t.Errorf("没点到的几项变了: %#v", config)
	}
	if config.Zones != base.Zones {
		t.Errorf("颜色变了: %#v", config.Zones)
	}
	if !config.PerZone {
		t.Error("分区设置不该变成全局设置")
	}
}

func TestLightingColorSetsEveryZone(t *testing.T) {
	base := lightingBase()

	config, err := parseLightingOptions(base, []string{"color", "#123456"})
	if err != nil {
		t.Fatal(err)
	}
	if config.PerZone {
		t.Error("全局颜色不该是分区设置")
	}
	for i, zone := range config.Zones {
		if zone != (hardware.RGBColor{R: 0x12, G: 0x34, B: 0x56}) {
			t.Errorf("分区 %d 的颜色不对: %#v", i+1, zone)
		}
	}
	if config.Effect != base.Effect {
		t.Errorf("效果变了: %v", config.Effect)
	}
}

func TestLightingZoneKeepsTheOtherZones(t *testing.T) {
	base := lightingBase()

	for _, args := range [][]string{
		{"zone", "2", "#00FF00"},
		{"zone", "2", "#00FF00", "all"},
	} {
		config, err := parseLightingOptions(base, args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !config.PerZone {
			t.Errorf("%v: 点过分区就该是分区设置", args)
		}
		if config.Zones[1] != (hardware.RGBColor{G: 0xFF}) {
			t.Errorf("%v: 第二个分区的颜色不对: %#v", args, config.Zones[1])
		}
		if config.Zones[0] != base.Zones[0] || config.Zones[3] != base.Zones[3] {
			t.Errorf("%v: 没点到的分区变了: %#v", args, config.Zones)
		}
	}
}

func TestLightingOptionsRejectBadInput(t *testing.T) {
	base := lightingBase()

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"effect"}, "needs a value"},
		{[]string{"effect", "bogus"}, "unknown lighting effect"},
		{[]string{"speed", "x"}, "parsing"},
		{[]string{"direction", "x"}, "parsing"},
		{[]string{"brightness", "x"}, "parsing"},
		{[]string{"color", "nope"}, "must be six hex digits"},
		{[]string{"color"}, "needs a value"},
		{[]string{"zone", "2"}, "needs a number and a colour"},
		{[]string{"zone", "0", "#FFFFFF"}, "out of range"},
		{[]string{"zone", "5", "#FFFFFF"}, "out of range"},
		{[]string{"zone", "x", "#FFFFFF"}, "parsing"},
		{[]string{"bogus"}, "unknown lighting option"},
	}

	for _, item := range cases {
		if _, err := parseLightingOptions(base, item.args); err == nil {
			t.Errorf("%v 该报错", item.args)
		} else if !strings.Contains(err.Error(), item.want) {
			t.Errorf("%v 错误信息里没有 %q: %v", item.args, item.want, err)
		}
	}
}

func TestDescribeFan(t *testing.T) {
	if got := describeFan(hardware.FanAuto, 0); got != "auto" {
		t.Errorf("自动是 %q", got)
	}
	if got := describeFan(hardware.FanMax, 0); got != "max" {
		t.Errorf("全速是 %q", got)
	}
	if got := describeFan(hardware.FanCustom, 50); !strings.Contains(got, "50") {
		t.Errorf("固定占空比是 %q", got)
	}
}

func TestDescribeGPUFan(t *testing.T) {
	readable := app.FanStatus{GPUMode: hardware.FanAuto, GPUModeReadable: true}
	if got := describeGPUFan(readable); strings.Contains(got, "not readable") {
		t.Errorf("读得到的值不该带那个标记: %q", got)
	}

	stored := app.FanStatus{GPUMode: hardware.FanAuto, GPUModeReadable: false}
	if got := describeGPUFan(stored); !strings.Contains(got, "not readable") {
		t.Errorf("读不到的值该带那个标记: %q", got)
	}
}

func TestMinMax(t *testing.T) {
	values := []int{3, 9, 5}
	if got := minOf(values); got != 3 {
		t.Errorf("最小的是 %d", got)
	}
	if got := maxOf(values); got != 9 {
		t.Errorf("最大的是 %d", got)
	}
}

func TestOnOff(t *testing.T) {
	if onOff(true) != "on" {
		t.Error("真该是 on")
	}
	if onOff(false) != "off" {
		t.Error("假该是 off")
	}
}
