package machine

import (
	"strings"
	"testing"
	"time"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// fastVerify shortens the wait of the read back check.
//
// 机器没有接受设置的时候，核对要等一会儿才放弃。用例里等三秒太慢，所以把
// 等待的时间改短。
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

// TestThatTheLightingIsCheckedAfterTheWrite 写完灯效要读回核对。
func TestThatTheLightingIsCheckedAfterTheWrite(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := hardware.RGBConfig{
		Effect:     hardware.EffectWave,
		Speed:      5,
		Direction:  1,
		Brightness: 60,
		PerZone:    true,
	}
	config.Zones[0] = hardware.RGBColor{R: 255}

	if err := backend.SetLighting(config); err != nil {
		t.Fatalf("写入灯效失败: %v", err)
	}
}

// TestThatALightingThatWasNotAcceptedIsReported 机器没有接受的时候要报错。
func TestThatALightingThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	// The service keeps reporting the old brightness.
	service.answers[agentservice.FunctionLighting] = answer(
		map[string]any{"brightness": 10}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionLighting: true}

	fastVerify(t)
	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetLighting(hardware.RGBConfig{Brightness: 90})
	if err == nil {
		t.Fatal("机器没有接受这个亮度时该报错")
	}
	if !strings.Contains(err.Error(), "亮度") {
		t.Fatalf("错误里没有说清是亮度: %v", err)
	}
}

// TestThatAColourThatWasNotAcceptedIsReported 分区颜色没有生效时要报错。
func TestThatAColourThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs": []any{
			map[string]any{"LED_id": 0, "color": "#00FF00", "status": 1},
		},
		"brightness": 50,
		"color":      "#ff0000",
		"effect":     "STATIC",
		"speed":      3,
		"direction":  0,
	}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionLighting: true}

	fastVerify(t)
	backend := newBackend(t, service)

	config := hardware.RGBConfig{Brightness: 50, Speed: 3, PerZone: true}
	config.Zones[0] = hardware.RGBColor{R: 255}

	err := backend.SetLighting(config)
	if err == nil {
		t.Fatal("机器没有接受这个颜色时该报错")
	}
	if !strings.Contains(err.Error(), "分区") {
		t.Fatalf("错误里没有说清是分区: %v", err)
	}
}

// TestThatAnEffectThatWasNotAcceptedIsReported 灯效没有生效时要报错。
func TestThatAnEffectThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs":       []any{},
		"brightness": 50,
		"effect":     "STATIC",
		"speed":      3,
		"direction":  0,
	}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionLighting: true}

	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetLighting(hardware.RGBConfig{
		Effect:     hardware.EffectWave,
		Brightness: 50,
		Speed:      3,
	})
	if err == nil {
		t.Fatal("机器没有接受这个灯效时该报错")
	}
	if !strings.Contains(err.Error(), "灯效") {
		t.Fatalf("错误里没有说清是灯效: %v", err)
	}
}

// TestThatTheTimeoutThatWasNotAcceptedIsReported 超时开关没有生效时要报错。
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

// TestThatTheProfileThatWasNotAcceptedIsReported 性能档位没有换时要报错。
func TestThatTheProfileThatWasNotAcceptedIsReported(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionOperatingMode] = answer(
		map[string]any{"mode": 1}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionOperatingMode: true}

	fastVerify(t)
	backend := newBackend(t, service)

	err := backend.SetPowerMode(hardware.PowerPerformance)
	if err == nil {
		t.Fatal("机器没有换档时该报错")
	}
	if !strings.Contains(err.Error(), "性能档位") {
		t.Fatalf("错误里没有说清是性能档位: %v", err)
	}
}

// TestThatASwitchThatWasNotAcceptedIsReported 开关没有生效时要报错。
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

// TestThatTheGraphicsModeIsNotCheckedAfterTheWrite 显卡模式不读回核对。
//
// 图形模式写进的是杂项设置表，机器要到重启才真的换过去，读回来的值说明不了
// 什么，所以这一项只看服务有没有收下。
func TestThatTheGraphicsModeIsNotCheckedAfterTheWrite(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionGPUMode] = answer(
		map[string]any{"mode": 1}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionGPUMode: true}

	backend := newBackend(t, service)

	if err := backend.SetGPUMode(hardware.GPUModeHybrid); err != nil {
		t.Fatalf("显卡模式不该读回核对: %v", err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("服务收到了 %d 次写入，不是 1 次", len(service.writes))
	}
}

// TestThatTheWindowsKeyGoesThroughTheService Windows 键问的是原厂服务。
//
// 实测：原厂服务收下 WIN_KEY 的请求，值却一直不变，所以写完的核对会把它
// 报出来——不假装成功。
func TestThatTheWindowsKeyGoesThroughTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionWinKey] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)
	// 服务收了请求却没改状态：读回来还是老样子。
	service.stale = map[string]bool{agentservice.FunctionWinKey: true}
	backend := newBackend(t, service)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatalf("读取 Windows 键失败: %v", err)
	}
	if locked {
		t.Error("服务报的是这个键还能用")
	}

	// 服务的值没有跟着变，核对要报出来。
	err = backend.SetWinKeyLocked(true)
	if err == nil {
		t.Error("服务没有接受这个锁，却报告成功")
	}
	if len(service.writes) == 0 {
		t.Error("锁没有写给服务")
	}
}

// TestThatTheResetOfTheLightingIsWrittenToTheService 复位灯效走原厂服务。
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
	if brightness := parameter["brightness"]; brightness != 100 {
		t.Errorf("复位的亮度是 %v，不是 100", brightness)
	}
}

// TestThatTheOverclockCapabilityIsNotSupported 超频能力原厂服务不实现。
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
