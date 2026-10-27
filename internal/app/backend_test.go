package app

import (
	"errors"
	"strings"
	"testing"

	"viper/internal/hardware"
)

// newTestAppWithoutMachine opens the program with a machine that cannot be
// reached.
//
// 机器连不上的时候，每一个与硬件有关的调用都要报错，不能拿别的东西顶替。
// 这台"机器"是替身，所以用例不会碰到真的机器。
func newTestAppWithoutMachine(t *testing.T) *App {
	t.Helper()

	application := NewWithStore(mustStore(t))
	application.SetBackendOpener(func() (hardware.Gaming, string, error) {
		return nil, "", errors.New("the machine is not there")
	})
	return application
}

// TestThatTheFirmwareIsNotAFallback 原厂服务不在的时候直接报错。
//
// 直接操控固件只是补充，不是这条路兜底的。这条规矩要有用例守着，不然改着
// 改着就又偷偷换了一条路。
func TestThatTheFirmwareIsNotAFallback(t *testing.T) {
	_, _, err := openMachine("127.0.0.1:1")
	if err == nil {
		t.Fatal("原厂服务不在的时候该报错")
	}
	if !strings.Contains(err.Error(), "补充") {
		t.Fatalf("错误里没有说清直接操控固件只是补充: %v", err)
	}
}
