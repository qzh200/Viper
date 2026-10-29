package machine

import (
	"errors"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

func TestCoolBoostIsTheMaximumOfTheFans(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 1,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 0, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 0, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	enabled, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("两个风扇都是全速的时候，加速该是开着的")
	}
}

func TestCoolBoostIsOffWhenTheControllerIsInCharge(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 0,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 50, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 50, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	enabled, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("控制器在管的时候，加速该是关着的")
	}
}

func TestCoolBoostFailsWhenTheServiceIsGone(t *testing.T) {
	service := newFakeService()
	service.getErr = errors.New("服务不可达")

	backend := newBackend(t, service)

	if _, err := backend.CoolBoostEnabled(); err == nil {
		t.Error("服务不在的时候该报错")
	}
}

func TestCoolBoostTurnsTheFansToFullSpeed(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 0,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 50, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 50, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	if err := backend.CoolBoost(true); err != nil {
		t.Fatal(err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("写了 %d 次", len(service.writes))
	}
	if mode := service.writes[0].Parameter["mode"]; mode != fanModeMax {
		t.Errorf("写的模式是 %v，不是 %d", mode, fanModeMax)
	}
}

func TestCoolBoostHandsTheFansBack(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 1,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 0, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 0, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	if err := backend.CoolBoost(false); err != nil {
		t.Fatal(err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("写了 %d 次", len(service.writes))
	}
	if mode := service.writes[0].Parameter["mode"]; mode != fanModeAuto {
		t.Errorf("写的模式是 %v，不是 %d", mode, fanModeAuto)
	}
}

var _ = domain.FanMax
