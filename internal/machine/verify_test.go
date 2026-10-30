package machine

import (
	"errors"
	"strings"
	"testing"
	"time"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

func fastVerify(t *testing.T) {
	t.Helper()

	timeout := verifyTimeout
	interval := verifyInterval
	verifyTimeout = 20 * time.Millisecond
	verifyInterval = time.Millisecond
	t.Cleanup(func() {
		verifyTimeout = timeout
		verifyInterval = interval
	})
}

func TestThatTheLightingIsCheckedAfterTheWrite(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := domain.RGBConfig{
		Effect:     domain.EffectWave,
		Speed:      5,
		Direction:  1,
		Brightness: 60,
		PerZone:    true,
	}
	config.Zones[0] = domain.RGBColor{R: 255}

	if err := backend.SetLighting(config); err != nil {
		t.Fatalf("写入灯效失败: %v", err)
	}
}

func TestThatTheReadKeepsWaitingWhenTheServiceAnswersWithoutData(t *testing.T) {
	fastVerify(t)

	tries := 0
	ok, err := poll(func() (bool, error) {
		tries++
		if tries < 3 {
			return false, errors.New("应答 LIGHTING 没有数据")
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("读了几次之后就不该再报错: %v", err)
	}
	if !ok {
		t.Fatal("读到最后成功了，该算是核对通过")
	}
	if tries < 3 {
		t.Errorf("只读了 %d 次", tries)
	}
}

func TestThatTheLastErrorIsReportedWhenTheReadingKeepsFailing(t *testing.T) {
	fastVerify(t)

	_, err := poll(func() (bool, error) {
		return false, errors.New("应答 LIGHTING 没有数据")
	})
	if err == nil {
		t.Fatal("一直读不到的时候该报错")
	}
	if !strings.Contains(err.Error(), "没有数据") {
		t.Errorf("错误信息不对: %v", err)
	}
}

func TestThatALightingThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(
		map[string]any{"brightness": 3}, agentservice.ResultOK)
	// 设备一直报 5 档，写下去的 3 档没有生效。
	service.devices = answer([]any{
		map[string]any{
			"device_index":      0,
			"device_type":       5,
			"brightness":        5,
			"active_mode_index": 1,
			"vector_of_modes": []any{
				map[string]any{"mode_name": "DIRECT", "mode_value": 255},
				map[string]any{"mode_name": "STATIC", "mode_value": 0},
			},
		},
	}, agentservice.ResultOK)

	fastVerify(t)
	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetLighting(domain.RGBConfig{Brightness: 3})
	if err == nil {
		t.Fatal("机器没有接受这个亮度时该报错")
	}
	if !strings.Contains(err.Error(), "亮度") {
		t.Fatalf("错误里没有说清是亮度: %v", err)
	}
}

func TestThatAColourThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs": []any{
			map[string]any{"LED_id": 0, "color": "#00FF00", "status": 1},
		},
		"brightness": 3,
		"color":      "#ff0000",
		"effect":     "STATIC",
		"speed":      3,
		"direction":  0,
	}, agentservice.ResultOK)
	// 设备亮度与写下去的不一样，核验就该拦下来。
	service.devices = answer([]any{
		map[string]any{
			"device_index":      0,
			"device_type":       5,
			"brightness":        5,
			"active_mode_index": 1,
			"vector_of_modes": []any{
				map[string]any{"mode_name": "DIRECT", "mode_value": 255},
				map[string]any{"mode_name": "STATIC", "mode_value": 0},
			},
		},
	}, agentservice.ResultOK)

	fastVerify(t)
	backend := newBackend(t, service)

	config := domain.RGBConfig{Brightness: 3, Speed: 3, PerZone: true}
	config.Zones[0] = domain.RGBColor{R: 255}

	err := backend.SetLighting(config)
	if err == nil {
		t.Fatal("机器没有接受这个颜色时该报错")
	}
	if !strings.Contains(err.Error(), "亮度") {
		t.Fatalf("错误里没有说清是亮度: %v", err)
	}
}

func TestThatAnEffectThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs":       []any{},
		"brightness": 3,
		"effect":     "STATIC",
		"speed":      3,
		"direction":  0,
	}, agentservice.ResultOK)
	// 设备亮度对得上，但跑的灯效不是波形。
	service.devices = answer([]any{
		map[string]any{
			"device_index":      0,
			"device_type":       5,
			"brightness":        3,
			"active_mode_index": 1,
			"vector_of_modes": []any{
				map[string]any{"mode_name": "DIRECT", "mode_value": 255},
				map[string]any{"mode_name": "STATIC", "mode_value": 0},
				map[string]any{"mode_name": "WAVE", "mode_value": 3},
			},
		},
	}, agentservice.ResultOK)

	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetLighting(domain.RGBConfig{
		Effect:     domain.EffectWave,
		Brightness: 3,
		Speed:      3,
	})
	if err == nil {
		t.Fatal("机器没有接受这个灯效时该报错")
	}
	if !strings.Contains(err.Error(), "灯效") {
		t.Fatalf("错误里没有说清是灯效: %v", err)
	}
}

func TestThatTheTimeoutThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(
		map[string]any{"brightness": 50}, agentservice.ResultOK)
	service.answers[agentservice.FunctionBacklightTimeout] = answer(
		map[string]any{"status": 0}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionBacklightTimeout: true}

	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetBacklight(50, true)
	if err == nil {
		t.Fatal("机器没有接受这个超时时该报错")
	}
}

func TestThatTheProfileThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionOperatingMode] = answer(
		map[string]any{"mode": 1}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionOperatingMode: true}

	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetPowerMode(domain.PowerPerformance)
	if err == nil {
		t.Fatal("机器没有换档时该报错")
	}
	if !strings.Contains(err.Error(), "性能档位") {
		t.Fatalf("错误里没有说清是性能档位: %v", err)
	}
}

func TestThatASwitchThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionWinKey] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)
	service.answers[agentservice.FunctionLCDOverdrive] = answer(
		map[string]any{"status": 0}, agentservice.ResultOK)
	service.answers[agentservice.FunctionBootSound] = answer(
		map[string]any{"status": 0}, agentservice.ResultOK)
	service.stale = map[string]bool{
		agentservice.FunctionWinKey:       true,
		agentservice.FunctionLCDOverdrive: true,
		agentservice.FunctionBootSound:    true,
	}

	fastVerify(t)
	backend := newBackend(t, service)

	if err := backend.SetWinKeyLocked(true); err == nil {
		t.Fatal("机器没有锁住 Windows 键时该报错")
	}
	if err := backend.SetLCDOverdrive(true); err == nil {
		t.Fatal("机器没有打开屏幕超频时该报错")
	}
	if err := backend.SetBootSound(true); err == nil {
		t.Fatal("机器没有打开开机音效时该报错")
	}
}

func TestThatTheGraphicsModeIsNotCheckedAfterTheWrite(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionGPUMode] = answer(
		map[string]any{"mode": 1}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionGPUMode: true}

	backend := newBackend(t, service)

	if err := backend.SetGPUMode(domain.GPUModeHybrid); err != nil {
		t.Fatalf("显卡模式不该读回核对: %v", err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("服务收到了 %d 次写入，不是 1 次", len(service.writes))
	}
}

func TestThatTheWindowsKeyGoesThroughTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionWinKey] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)

	service.stale = map[string]bool{agentservice.FunctionWinKey: true}
	backend := newBackend(t, service)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatalf("读取 Windows 键失败: %v", err)
	}
	if locked {
		t.Error("服务报的是这个键还能用")
	}

	err = backend.SetWinKeyLocked(true)
	if err == nil {
		t.Error("服务没有接受这个锁，却报告成功")
	}
	if len(service.writes) == 0 {
		t.Error("锁没有写给服务")
	}
}

func TestThatTheResetOfTheLightingIsWrittenToTheService(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	if err := backend.ResetLighting(); err != nil {
		t.Fatalf("复位灯效失败: %v", err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("服务收到了 %d 次写入，不是 1 次", len(service.writes))
	}

	parameter := service.writes[0].Parameter
	if effect := parameter["effect"]; effect != "STATIC" {
		t.Errorf("复位的灯效是 %v，不是静态", effect)
	}
	if brightness := parameter["brightness"]; brightness != 5 {
		t.Errorf("复位的亮度是 %v，不是 5", brightness)
	}
}

func TestThatTheOverclockCapabilityIsNotSupported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.CapabilityOverclockingMode] = agentservice.Answer{
		Request: agentservice.CapabilityOverclockingMode,
		Result:  agentservice.ResultUnsupported,
	}
	backend := newBackend(t, service)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatalf("读取超频能力失败: %v", err)
	}
	if gpu || cpu {
		t.Errorf("超频能力是显卡 %v 处理器 %v，不该报成支持", gpu, cpu)
	}
}
