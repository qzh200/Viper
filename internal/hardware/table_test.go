package hardware

import (
	"testing"

	"viper/internal/wmi"
)

// 固件的杂项设置表按编号回答，每一项的值都不一样。
//
// 替身如果只按方法名回答，读回校验对编号就是盲的：写进去的是一���
// 值，读回来的是另一项的值，校验要么永远失败，要么永远通过。

func TestTheMiscTableAnswersByItsIdentifier(t *testing.T) {
	provider := newFakeProvider()
	// 每一项的值都不一样，和真机上的表一样。
	provider.answerFor("GetGamingMiscSetting", uint64(MiscGPUMode), 0x00000100)
	provider.answerFor("GetGamingMiscSetting", uint64(MiscBootSound), 0x00000000)
	provider.answerFor("GetGamingMiscSetting", uint64(MiscOperationMode), 0x00000400)
	backend := newFakeBackend(provider)

	// 显卡模式是 1，也就是混合。
	mode, err := backend.GPUMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != GPUModeHybrid {
		t.Errorf("the graphics mode is %d, want the hybrid one", mode)
	}

	// 开机音效是关着的。
	sound, err := backend.BootSound()
	if err != nil {
		t.Fatal(err)
	}
	if sound {
		t.Error("the sound at the start of the machine is on")
	}

	// 性能档位是 4，也就是性能。
	power, err := backend.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if power != PowerPerformance {
		t.Errorf("the performance profile is %s, want the performance one", power)
	}
}

func TestTheMiscTableReportsASettingItDoesNotKnow(t *testing.T) {
	// 固件不认识这一项时会答一个非零的状态，替身也要这样答。
	provider := newFakeProvider()
	provider.answerFor("GetGamingMiscSetting", uint64(MiscGPUMode), 0x00000100)
	backend := newFakeBackend(provider)

	// 开机音效没有在表里，读取要报"固件不认识"，而不是报成关着。
	if _, err := backend.BootSound(); err == nil {
		t.Error("一个固件不认识的设置被读成了值")
	}
}

func TestTheWriteBackReadsTheSettingItWrote(t *testing.T) {
	// 写完之后读回来的是刚写进去的那一项，不是别的项。
	provider := newFakeProvider()
	provider.answerFor("GetGamingMiscSetting", uint64(MiscBootSound), 0x00000100)
	backend := newFakeBackend(provider)

	// 写的是开机音效，值是 1，读回来也是 1。
	if err := backend.SetBootSound(true); err != nil {
		t.Errorf("固件接受了这次写入，却报错：%v", err)
	}
}

func TestTheWriteBackOfAnotherSettingIsNotAccepted(t *testing.T) {
	// 表里的每一项都不一样，所以写进去的值和读回来的值对不上时，
	// 校验要能抓到。
	provider := newFakeProvider()
	provider.answerFor("GetGamingMiscSetting", uint64(MiscBootSound), 0x00000000)
	backend := newFakeBackend(provider)

	// 写的是 1，读回来的是 0。
	if err := backend.SetBootSound(true); err == nil {
		t.Error("固件没有接受这次写入，却报告成功")
	}
}

func TestTheFanSpeedIsReadPerFan(t *testing.T) {
	// 风扇转速是按风扇编号回答的，两个风扇的转速不一样。
	provider := newFakeProvider()
	provider.answerFor("GetGamingFanSpeed", FanIndexCPU, 0x00003C00)
	provider.answerFor("GetGamingFanSpeed", FanIndexGPUPrimary, 0x00002800)
	backend := newFakeBackend(provider)

	cpu, err := backend.fanSpeed(FanIndexCPU)
	if err != nil {
		t.Fatal(err)
	}
	if cpu != 60 {
		t.Errorf("the fan of the processor runs at %d percent, want 60", cpu)
	}

	gpu, err := backend.fanSpeed(FanIndexGPUPrimary)
	if err != nil {
		t.Fatal(err)
	}
	if gpu != 40 {
		t.Errorf("the fan of the graphics card runs at %d percent, want 40", gpu)
	}
}

func TestTheSensorIsReadPerCategory(t *testing.T) {
	// 传感器的读数是按分类回答的，处理器和显卡的温度不一样。
	provider := newFakeProvider()
	provider.answerFor("GetGamingSysInfo", uint64(sysInfoCPUTemp), 0x00003200)
	provider.answerFor("GetGamingSysInfo", uint64(sysInfoGPUTemp), 0x00003C00)
	backend := newFakeBackend(provider)

	cpu, err := backend.sysInfo(sysInfoCPUTemp, false)
	if err != nil {
		t.Fatal(err)
	}
	if cpu != 50 {
		t.Errorf("the processor runs at %d degrees, want 50", cpu)
	}

	gpu, err := backend.sysInfo(sysInfoGPUTemp, false)
	if err != nil {
		t.Fatal(err)
	}
	if gpu != 60 {
		t.Errorf("the graphics card runs at %d degrees, want 60", gpu)
	}
}

func TestTheEmptyAnswerOfAWrite(t *testing.T) {
	// 写设置的调用没有答案，替身要能这样答。
	provider := newFakeProvider()

	if _, err := provider.Call(ClassGaming, "SetGamingLEDColor", "path", map[string]wmi.Value{
		"gmInput": wmi.Uint64(0x8700C701),
	}); err != nil {
		t.Errorf("写设置的调用被拒绝了：%v", err)
	}
}
