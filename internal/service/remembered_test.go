package service

import (
	"bytes"
	"testing"

	"viper/internal/hardware"
)

// 服务自己记住的那些设置：固件里没有它们的影子，服务只能记住自己写过的
// 值，读的时候报出来。

// TestTheRememberedSettingsOfANewService 没有写过的时候报原厂的值。
func TestTheRememberedSettingsOfANewService(t *testing.T) {
	provider := NewProvider(nil, nil)

	boost, ok := provider.Setting("BATTERY_BOOST")
	if !ok {
		t.Fatal("电池加速答不出来")
	}
	if value := boost.(map[string]any); value["status"] != 1 {
		t.Errorf("电池加速报的是 %v，想要 1", value["status"])
	}

	sound, ok := provider.Setting("SOUND_MODE")
	if !ok {
		t.Fatal("声音模式答不出来")
	}
	if value := sound.(map[string]any); value["function"] != "DTS" || value["mode"] != 0 {
		t.Errorf("声音模式报的是 %v，想要 DTS 与 0", value)
	}

	logo, ok := provider.Setting("CUSTOM_BOOT_LOGO")
	if !ok {
		t.Fatal("启动标志答不出来")
	}
	if value := logo.(map[string]any); value["imageSource"] != "" {
		t.Errorf("启动标志报的是 %v，想要空文本", value["imageSource"])
	}
}

// TestTheRememberedBatteryBoost 写过的电池加速要被记住。
func TestTheRememberedBatteryBoost(t *testing.T) {
	provider := NewProvider(nil, nil)

	if err := provider.Apply("BATTERY_BOOST", map[string]any{"status": float64(0)}); err != nil {
		t.Fatal(err)
	}

	value, ok := provider.Setting("BATTERY_BOOST")
	if !ok {
		t.Fatal("电池加速答不出来")
	}
	if got := value.(map[string]any)["status"]; got != 0 {
		t.Errorf("电池加速报的是 %v，想要 0", got)
	}
}

// TestTheRememberedSoundMode 写过的声音模式要被记住。
func TestTheRememberedSoundMode(t *testing.T) {
	provider := NewProvider(nil, nil)

	if err := provider.Apply("SOUND_MODE", map[string]any{
		"function": "DTS",
		"mode":     float64(2),
	}); err != nil {
		t.Fatal(err)
	}

	value, ok := provider.Setting("SOUND_MODE")
	if !ok {
		t.Fatal("声音模式答不出来")
	}
	sound := value.(map[string]any)
	if sound["function"] != "DTS" || sound["mode"] != 2 {
		t.Errorf("声音模式报的是 %v，想要 DTS 与 2", sound)
	}
}

// TestTheRememberedBootLogo 写过的启动标志要被记住。
func TestTheRememberedBootLogo(t *testing.T) {
	provider := NewProvider(nil, nil)

	if err := provider.Apply("CUSTOM_BOOT_LOGO", map[string]any{
		"imageSource": `C:\logo.png`,
	}); err != nil {
		t.Fatal(err)
	}

	value, ok := provider.Setting("CUSTOM_BOOT_LOGO")
	if !ok {
		t.Fatal("启动标志答不出来")
	}
	if got := value.(map[string]any)["imageSource"]; got != `C:\logo.png` {
		t.Errorf("启动标志报的是 %v，想要图片的路径", got)
	}
}

// TestTheWriteOfTheRememberedSettings 写入时的数值要能被读懂。
func TestTheWriteOfTheRememberedSettings(t *testing.T) {
	provider := NewProvider(nil, nil)

	if err := provider.Apply("BATTERY_BOOST", map[string]any{"status": "not a number"}); err == nil {
		t.Error("一个不是数字的电池加速被接受了")
	}
	if err := provider.Apply("SOUND_MODE", "wave"); err == nil {
		t.Error("一个不是数值的声音模式被接受了")
	}
	if err := provider.Apply("CUSTOM_BOOT_LOGO", "logo"); err == nil {
		t.Error("一个不是数值的启动标志被接受了")
	}
}

// TestTheGPUModeOutsideTheRange 图形模式只能有三个。
func TestTheGPUModeOutsideTheRange(t *testing.T) {
	provider, machine := newProviderWith(t)

	for _, value := range []float64{-1, 3, 99} {
		err := provider.Apply("GPU_MODE", map[string]any{"mode": value})
		if err == nil {
			t.Errorf("图形模式 %v 被接受了", value)
		}
	}
	if len(machine.writtenGPUMode) != 0 {
		t.Errorf("图形模式写了 %v，非法值不该写进机器", machine.writtenGPUMode)
	}
}

// TestTheGPUModeOfTheRange 三个图形模式都要能写。
func TestTheGPUModeOfTheRange(t *testing.T) {
	provider, machine := newProviderWith(t)

	for value, want := range map[float64]hardware.GPUMode{
		0: hardware.GPUModeHybrid,
		1: hardware.GPUModeDiscrete,
		2: hardware.GPUModeAuto,
	} {
		if err := provider.Apply("GPU_MODE", map[string]any{"mode": value}); err != nil {
			t.Fatalf("图形模式 %v 写不进去: %v", value, err)
		}
		last := machine.writtenGPUMode[len(machine.writtenGPUMode)-1]
		if last != want {
			t.Errorf("图形模式 %v 写成了 %v，想要 %v", value, last, want)
		}
	}
}

// TestTheStoredLightingIsLimited 存下来的灯效要在固件能做的范围里。
func TestTheStoredLightingIsLimited(t *testing.T) {
	cases := []struct {
		name  string
		value hardware.RGBConfig
		speed uint64
		dire  uint64
		brit  uint64
	}{
		{
			name:  "太小",
			value: hardware.RGBConfig{Speed: 0, Direction: 0, Brightness: 0},
			speed: 1, dire: 0, brit: 0,
		},
		{
			name:  "太大",
			value: hardware.RGBConfig{Speed: 20, Direction: 5, Brightness: 200},
			speed: 9, dire: 1, brit: 100,
		},
		{
			name:  "刚好",
			value: hardware.RGBConfig{Speed: 5, Direction: 1, Brightness: 70},
			speed: 5, dire: 1, brit: 70,
		},
	}

	for _, c := range cases {
		stored := storedLighting(c.value)
		if stored.Speed != c.speed {
			t.Errorf("%s 的速度存成了 %d，想要 %d", c.name, stored.Speed, c.speed)
		}
		if stored.Direction != c.dire {
			t.Errorf("%s 的方向存成了 %d，想要 %d", c.name, stored.Direction, c.dire)
		}
		if stored.Brightness != c.brit {
			t.Errorf("%s 的亮度存成了 %d，想要 %d", c.name, stored.Brightness, c.brit)
		}
	}
}

// TestTheWrittenLightingIsLimited 写进机器的灯效也要在范围里。
func TestTheWrittenLightingIsLimited(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("LIGHTING", map[string]any{
		"effect":     "WAVE",
		"speed":      float64(20),
		"direction":  float64(5),
		"brightness": float64(200),
	})
	if err != nil {
		t.Fatal(err)
	}

	written := machine.written[0]
	if written.Speed != 9 {
		t.Errorf("速度写成了 %d，想要 9", written.Speed)
	}
	if written.Direction != 1 {
		t.Errorf("方向写成了 %d，想要 1", written.Direction)
	}
	if written.Brightness != 100 {
		t.Errorf("亮度写成了 %d，想要 100", written.Brightness)
	}
}

// TestTheAnswerOfTheOverclockingCapability 超频能力原厂服务不实现。
func TestTheAnswerOfTheOverclockingCapability(t *testing.T) {
	server := &Server{provider: NewProvider(nil, nil)}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketInitialization,
		Function: capOverclockingModeCapability,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 原厂服务的应答里没有 data 字段，只有请求名与失败码。
	want := "{\n\t\"request\" : \"SUPPORT_OVERCLOCKING_MODE_CAPABILITY\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("应答是\n%q\n想要\n%q", out.String(), want)
	}
}

// TestTheCapabilityOfTheOverclocking 能力表里不能有超频这一项。
func TestTheCapabilityOfTheOverclocking(t *testing.T) {
	if value, ok := NewProvider(nil, nil).Capability(capOverclockingModeCapability); ok {
		t.Errorf("超频能力被答成了 %v", value)
	}
}
