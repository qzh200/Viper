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

// newTestApp opens the program with a machine that is not there.
//
// 用例不去碰真的机器：真的机器上有原厂服务在跑，不注入替身的话，程序会
// 真的去问那台机器，测出来的数就随机器的状态变了。
func newTestApp(t *testing.T) *App {
	t.Helper()
	return newTestAppWithoutMachine(t)
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
	if !strings.Contains(err.Error(), "没有第二条路") {
		t.Fatalf("错误里没有说清没有第二条路: %v", err)
	}
}
