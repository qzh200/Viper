package hardware

import (
	"testing"

	"viper/internal/wmi"
)

// 固件的杂项设置表按编号回答，每一项的值都不一样。
//
// 替身如果只按方法名回答，读回校验对编号就是盲的：写进去的是一���
// 值，读回来的是另一项的值，校验要么永远失败，要么永远通过。

func TestTheEmptyAnswerOfAWrite(t *testing.T) {
	// 写设置的调用没有答案，替身要能这样答。
	provider := newFakeProvider()

	if _, err := provider.Call(ClassGaming, "SetGamingLEDColor", "path", map[string]wmi.Value{
		"gmInput": wmi.Uint64(0x8700C701),
	}); err != nil {
		t.Errorf("写设置的调用被拒绝了：%v", err)
	}
}
