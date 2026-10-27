package hardware

import (
	"errors"
	"os"
	"testing"
)

// 固件那一侧现在只管原厂程序给不了的那几项：风扇表、超频能力、背光原始
// 缓冲区、复位灯效、锁 Windows 键。这一层的用例守着的就是这几项。

func TestTheResetOfTheLightingWritesWhiteInEveryZone(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.ResetLighting(); err != nil {
		t.Fatalf("复位灯效失败: %v", err)
	}

	// 四个分区各写一次纯白。
	writes := provider.callsOf("SetGamingLEDColor")
	if len(writes) != LightingZones {
		t.Fatalf("写了 %d 个分区，不是 %d 个", len(writes), LightingZones)
	}
	for _, write := range writes {
		if write.Input&0xFFFFFF00 != 0xFFFFFF00 {
			t.Errorf("分区 %d 的颜色是 %#x，不是纯白", write.Input&0xFF, write.Input)
		}
	}

	// 行为是静态效果、速度 3、方向 0。
	behaviour, ok := provider.callOf("SetGamingLEDBehavior")
	if !ok {
		t.Fatal("灯效的行为没有写")
	}
	if behaviour.Input&0xFF0000>>16 != uint64(EffectStatic) {
		t.Errorf("灯效是 %#x，不是静态", behaviour.Input&0xFF0000>>16)
	}
	if (behaviour.Input>>8)&0xFF != 3 {
		t.Errorf("速度是 %d，不是 3", (behaviour.Input>>8)&0xFF)
	}
}

func TestResetLightingStopsAtTheFirstError(t *testing.T) {
	provider := newFakeProvider()
	provider.fail(errors.New("固件拒绝了这次写入"))
	backend := newFakeBackend(provider)

	if err := backend.ResetLighting(); err == nil {
		t.Fatal("固件拒绝了这次写入，却报告成功")
	}
}

func TestTheCloseOfTheBackend(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	// 关掉之后不该崩。
	backend.Close()
}

func TestTheNumbersOfThePerformanceProfiles(t *testing.T) {
	// 原厂服务与固件用的是同一套编号。
	cases := map[PowerMode]uint64{
		PowerQuiet:       0,
		PowerBalanced:    1,
		PowerPerformance: 4,
		PowerTurbo:       5,
		PowerEco:         6,
	}
	for mode, want := range cases {
		if got := PowerModeCode(mode); got != want {
			t.Errorf("%v 的编号是 %d，不是 %d", mode, got, want)
		}
		if got := DecodePowerMode(want); got != mode {
			t.Errorf("编号 %d 解出来是 %v，不是 %v", want, got, mode)
		}
	}

	// 不认识的编号只能解成"不知道"。
	if got := DecodePowerMode(2); got != PowerUnknown {
		t.Errorf("编号 2 解出来是 %v，不是不知道", got)
	}
	if got := DecodePowerMode(99); got != PowerUnknown {
		t.Errorf("编号 99 解出来是 %v，不是不知道", got)
	}
}

// TestTheFirmwareOfThisMachine 连的是本机真正的固件，要管理员权限。
//
// 它平时是跳过的，只有在环境变量 VIPER_LIVE_FIRMWARE=1 之后才会跑，
// 用来核对补充的那几项在本机上读得到。
func TestTheFirmwareOfThisMachine(t *testing.T) {
	if os.Getenv("VIPER_LIVE_FIRMWARE") == "" {
		t.Skip("设置 VIPER_LIVE_FIRMWARE=1 之后才会连本机的固件（要管理员权限）")
	}

	backend, err := OpenFirmware()
	if err != nil {
		t.Fatalf("连不上固件: %v", err)
	}
	defer backend.Close()

	table, err := backend.FanTable()
	if err != nil {
		t.Errorf("读取风扇表失败: %v", err)
	} else {
		t.Logf("风扇表: %d", table)
	}

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Errorf("读取超频能力失败: %v", err)
	} else {
		t.Logf("超频能力: 显卡 %v 处理器 %v", gpu, cpu)
	}

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Errorf("读取背光缓冲区失败: %v", err)
	} else {
		t.Logf("背光缓冲区: % X", buffer)
	}

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Errorf("读取 Windows 键失败: %v", err)
	} else {
		t.Logf("Windows 键锁着: %v", locked)
	}
}
