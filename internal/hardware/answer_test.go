package hardware

import (
	"strings"
	"testing"

	"viper/internal/wmi"
)

// 固件没有给出 gmOutput 时，读到的不能是零。
//
// 少了这个检查，"读不到"会被当成"值是零"，而且上层拿不到任何错误。
//
// 这一层现在只管原厂程序给不了的那几项，所以要守的读取也只有这几项。

func TestTheReadsThatNeedAnAnswer(t *testing.T) {
	// 每个读取都要在固件不给答案时报告错误。
	cases := []struct {
		name string
		read func(b *Backend) error
	}{
		{"风扇表", func(b *Backend) error { _, err := b.FanTable(); return err }},
		{"档位（Windows 键）", func(b *Backend) error { _, err := b.WinKeyLocked(); return err }},
		{"背光缓冲区", func(b *Backend) error { _, err := b.BacklightBuffer(); return err }},
		{"超频能力", func(b *Backend) error { _, _, err := b.OverclockCapability(); return err }},
	}

	for _, c := range cases {
		provider := newFakeProvider()
		// 固件回了空应答，没有 gmOutput。
		provider.answer("GetGamingFanTable", map[string]wmi.Value{})
		provider.answer("GetGamingProfile", map[string]wmi.Value{})
		provider.answer("GetGamingKBBacklight", map[string]wmi.Value{})
		provider.answer("GetGamingMiscSetting", map[string]wmi.Value{})
		backend := newFakeBackend(provider)

		err := c.read(backend)
		if err == nil {
			t.Errorf("%s：固件没有给出答案时没有报错", c.name)
			continue
		}
		if !strings.Contains(err.Error(), "gmOutput") {
			t.Errorf("%s：报错没有说明固件没给答案，是 %v", c.name, err)
		}
	}
}

func TestTheOverclockThatIsNotTaken(t *testing.T) {
	// 固件不认识这个设置（状态字节是 2），超频能力只能报成不支持，不能当成
	// 支持，也不能当成出错。
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x00000002)
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatalf("读取超频能力失败: %v", err)
	}
	if gpu || cpu {
		t.Errorf("超频能力是显卡 %v 处理器 %v，不该报成支持", gpu, cpu)
	}
}
