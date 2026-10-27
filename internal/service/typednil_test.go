package service

import (
	"bytes"
	"testing"

	"viper/internal/hardware"
)

// 游戏硬件提供者装在接口里的时候，一个空指针跟 nil 并不相等。
// 只看接口的守卫会放行，第一个需要机器的请求就会让服务崩掉，
// 所以这里把这种情形钉死。

// TestTheProviderOfANilGamingProvider 造一个装着空指针的提供者。
func TestTheProviderOfANilGamingProvider(t *testing.T) {
	var machine *hardware.Backend

	provider := NewProvider(machine, nil)
	if provider.backend != nil {
		t.Error("提供者没有当成没有提供者")
	}
	if provider.hasBackend() {
		t.Error("提供者报告自己可以用")
	}
}

// TestTheProviderThatIsBuiltByHand 绕过构造函数也要看得出来。
func TestTheProviderThatIsBuiltByHand(t *testing.T) {
	var machine *hardware.Backend

	provider := &Provider{backend: machine}
	if provider.hasBackend() {
		t.Error("提供者报告自己可以用")
	}
}

// TestTheNilPointerInsideTheInterfaceCrashes 证明这个坑是真的。
//
// 一个装在接口里的空指针调用机器就会崩，所以只看接口的守卫不够用。
func TestTheNilPointerInsideTheInterfaceCrashes(t *testing.T) {
	var machine *hardware.Backend
	var provider hardware.Gaming = machine

	if provider == nil {
		t.Fatal("接口里的空指针跟 nil 竟然相等")
	}

	defer func() {
		if recover() == nil {
			t.Error("接口里的空指针调用机器居然没有崩")
		}
	}()
	provider.PowerMode()
}

// TestTheSettingOfAMachineThatIsNotThere 读一个需要机器的设置。
//
// 没有机器的时候服务要回答「答不出来」，不能崩掉。
func TestTheSettingOfAMachineThatIsNotThere(t *testing.T) {
	var machine *hardware.Backend
	provider := &Provider{backend: machine}

	for _, name := range []string{
		"OPERATING_MODE",
		"GPU_MODE",
		"WIN_KEY",
		"LCD_OVERDRIVE",
		"BOOT_SOUND",
		"FAN_CONTROL",
		"BACKLIGHT_30S",
	} {
		if _, ok := provider.Setting(name); ok {
			t.Errorf("没有机器的时候设置 %s 却被答出来了", name)
		}
	}
}

// TestTheWriteWithoutAMachine 写一个需要机器的设置。
func TestTheWriteWithoutAMachine(t *testing.T) {
	var machine *hardware.Backend
	provider := &Provider{backend: machine}

	for _, name := range []string{
		"OPERATING_MODE",
		"GPU_MODE",
		"WIN_KEY",
		"LCD_OVERDRIVE",
		"BOOT_SOUND",
		"BACKLIGHT_30S",
		"LIGHTING",
	} {
		err := provider.Apply(name, map[string]any{"mode": float64(1), "status": float64(1)})
		if err == nil {
			t.Errorf("没有机器的时候设置 %s 却写成功了", name)
		}
	}
}

// TestTheAnswerWithoutAMachine 服务要答出「答不出来」，而不是崩掉。
func TestTheAnswerWithoutAMachine(t *testing.T) {
	var machine *hardware.Backend
	server := &Server{provider: &Provider{backend: machine}}

	var out bytes.Buffer
	err := server.answer(&bufferConn{Buffer: &out}, Request{
		Packet:   PacketGetUpdatedData,
		Function: "OPERATING_MODE",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"data\" : null,\n\t\"request\" : \"OPERATING_MODE\",\n\t\"result\" : \"3\"\n}\n"
	if out.String() != want {
		t.Errorf("应答是\n%q\n想要\n%q", out.String(), want)
	}
}

// TestTheMonitorDataWithoutAMachine 监视数据里不能有机器的读数。
func TestTheMonitorDataWithoutAMachine(t *testing.T) {
	withoutTheSlowCalls(t)

	var machine *hardware.Backend
	provider := &Provider{backend: machine}

	data := provider.MonitorData()
	if _, ok := data["CPU_TEMPERATURE"]; ok {
		t.Error("应答里有读不到机器的读数")
	}
	if _, ok := data["RAM_TOTAL"]; !ok {
		t.Error("应答里没有机器的记忆")
	}
}
