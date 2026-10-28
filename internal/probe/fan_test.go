package probe

import (
	"testing"

	"viper/internal/agentservice"
)

func TestWriteFan(t *testing.T) {
	service := newFakeService()

	err := WriteFan(service, 2,
		FanData{Name: "CPU", Auto: 0, Speed: 50, Index: 0},
		FanData{Name: "GPU", Auto: 0, Speed: 40, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(service.written) != 1 {
		t.Fatalf("写了几次: %#v", service.written)
	}
	if service.written[0].function != agentservice.FunctionFanControl {
		t.Errorf("写错了命令: %s", service.written[0].function)
	}
}

func TestWriteFanOfAServiceThatIsNotThere(t *testing.T) {
	service := newFakeService()
	service.fail = errNoService

	if err := WriteFan(service, 0, FanData{}, FanData{}); err == nil {
		t.Error("服务不在的时候该报错")
	}
	if err := WriteLighting(service, LightingReading{}); err == nil {
		t.Error("服务不在的时候该报错")
	}
	if err := WriteWinKey(service, 0); err == nil {
		t.Error("服务不在的时候该报错")
	}
	if err := WriteFans(service, FanReading{}); err == nil {
		t.Error("服务不在的时候该报错")
	}
}

var errNoService = errService{}

type errService struct{}

func (errService) Error() string { return "服务不可达" }
