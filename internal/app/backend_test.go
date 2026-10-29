package app

import (
	"errors"
	"strings"
	"testing"

	"viper/internal/domain"
)

func newTestAppWithoutMachine(t *testing.T) *App {
	t.Helper()

	application := NewWithStore(mustStore(t))
	application.SetBackendOpener(func() (domain.Gaming, string, error) {
		return nil, "", errors.New("the machine is not there")
	})
	return application
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	return newTestAppWithoutMachine(t)
}

func TestThatTheFirmwareIsNotAFallback(t *testing.T) {
	_, _, err := openMachine("127.0.0.1:1")
	if err == nil {
		t.Fatal("原厂服务不在的时候该报错")
	}
	if !strings.Contains(err.Error(), "没有第二条路") {
		t.Fatalf("错误里没有说清没有第二条路: %v", err)
	}
}
