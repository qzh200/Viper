package hardware

import (
	"os"
	"testing"
)

func TestTheIdentityOfThisMachine(t *testing.T) {
	identity, err := SystemInfo()
	if err != nil {
		t.Skipf("这台机器的身份读不到: %v", err)
	}
	if identity.Product == "" {
		t.Error("机器的型号是空的")
	}
	t.Logf("机器: %s %s", identity.Product, identity.Family)
}

func TestThePowerStatusOfThisMachine(t *testing.T) {
	status, err := SystemPowerStatus()
	if err != nil {
		t.Skipf("这台机器的电源状态读不到: %v", err)
	}
	_ = status.String()
}

func TestTheStickyKeysOfThisMachine(t *testing.T) {
	enabled, err := StickyKeys()
	if err != nil {
		t.Skipf("这台机器的粘滞键读不到: %v", err)
	}
	t.Logf("粘滞键: %v", enabled)
}

func TestTheStickyKeysOfThisMachineCanBeSwitched(t *testing.T) {
	if os.Getenv("VIPER_LIVE_MACHINE") == "" {
		t.Skip("设置 VIPER_LIVE_MACHINE=1 之后才会改这台机器的粘滞键（会改回原样）")
	}

	original, err := StickyKeys()
	if err != nil {
		t.Skipf("这台机器的粘滞键读不到: %v", err)
	}

	if err := SetStickyKeys(!original); err != nil {
		t.Fatalf("改粘滞键失败: %v", err)
	}
	current, err := StickyKeys()
	if err != nil {
		t.Fatalf("读回粘滞键失败: %v", err)
	}
	if current == original {
		t.Errorf("粘滞键还是 %v，没有变成 %v", current, !original)
	}

	if err := SetStickyKeys(original); err != nil {
		t.Fatalf("改回粘滞键失败: %v", err)
	}
}
