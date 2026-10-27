package app

import (
	"errors"
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
