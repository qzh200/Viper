package hardware

import (
	"strings"
	"testing"

	"viper/internal/wmi"
)

// 固件没有给出 gmOutput 时，读到的不能是零。
//
// 少了这个检查，"读不到"会被当成"值是零"：性能档位会变成静音模式，开机
// 音效会变成关着的，而且上层拿不到任何错误。

func TestTheReadsThatNeedAnAnswer(t *testing.T) {
	// 每个读取都要在固件不给答案时报告错误。
	cases := []struct {
		name string
		read func(b *Backend) error
	}{
		{"性能档位", func(b *Backend) error { _, err := b.PowerMode(); return err }},
		{"显卡模式", func(b *Backend) error { _, err := b.GPUMode(); return err }},
		{"开机音效", func(b *Backend) error { _, err := b.BootSound(); return err }},
		{"独显状态", func(b *Backend) error { _, err := b.DiscreteGPU(); return err }},
		{"键盘亮度", func(b *Backend) error { _, err := b.LightingBrightness(); return err }},
		{"风扇表", func(b *Backend) error { _, err := b.FanTable(); return err }},
		{"风扇行为", func(b *Backend) error { _, err := b.Fans(); return err }},
		{"档位（超频）", func(b *Backend) error { _, err := b.LCDOverdrive(); return err }},
		{"档位（Windows 键）", func(b *Backend) error { _, err := b.WinKeyLocked(); return err }},
	}

	for _, c := range cases {
		provider := newFakeProvider()
		// 固件回了空应答，没有 gmOutput。
		provider.answer("GetGamingMiscSetting", map[string]wmi.Value{})
		provider.answer("GetGamingRgbKb", map[string]wmi.Value{})
		provider.answer("GetGamingFanTable", map[string]wmi.Value{})
		provider.answer("GetGamingFanBehavior", map[string]wmi.Value{})
		provider.answer("GetGamingProfile", map[string]wmi.Value{})
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

func TestTheWriteThatReadsBackAStatus(t *testing.T) {
	// 固件不认识这个设置时会答一个非零的状态，值恰好是零的时候，
	// 只看值就会把"固件拒绝了"当成"写成功了"。
	provider := newFakeProvider()
	// 状态字节是 2，值是 0。
	provider.output("GetGamingMiscSetting", "gmOutput", 0x00000002)
	backend := newFakeBackend(provider)

	err := backend.SetBootSound(false)
	if err == nil {
		t.Fatal("固件拒绝了这次写入，却报告成功")
	}
	if !strings.Contains(err.Error(), "状态") {
		t.Errorf("报错没有说明状态字节，是 %v", err)
	}
}

func TestTheWriteThatReadsBackAnotherValue(t *testing.T) {
	// 固件把值保持在了别的地方，这也要报错。
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x00010100)
	backend := newFakeBackend(provider)

	err := backend.SetBootSound(false)
	if err == nil {
		t.Fatal("固件没有接受这次写入，却报告成功")
	}
}

func TestTheWriteThatTheFirmwareAccepts(t *testing.T) {
	// 固件接受了写入时不能报错。
	provider := newFakeProvider()
	// 状态字节是 0，值是 1。
	provider.output("GetGamingMiscSetting", "gmOutput", 0x00000100)
	backend := newFakeBackend(provider)

	if err := backend.SetBootSound(true); err != nil {
		t.Errorf("固件接受了这次写入，却报错：%v", err)
	}
}
