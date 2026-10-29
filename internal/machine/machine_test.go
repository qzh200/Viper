package machine

import (
	"encoding/json"
	"errors"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

type fakeWrite struct {
	Function string

	Parameter map[string]any
}

type fakeService struct {
	answers map[string]agentservice.Answer

	written map[string]map[string]any

	monitor agentservice.Answer

	writes []fakeWrite

	closed bool

	getErr error

	setErr error

	stale map[string]bool
}

func newFakeService() *fakeService {
	return &fakeService{
		answers: map[string]agentservice.Answer{},
		written: map[string]map[string]any{},
	}
}

func answer(data any, result string) agentservice.Answer {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return agentservice.Answer{Data: raw, Result: result}
}

func (f *fakeService) Get(function string) (agentservice.Answer, error) {
	if f.getErr != nil {
		return agentservice.Answer{}, f.getErr
	}
	result, ok := f.answers[function]

	if values, written := f.written[function]; written && !f.stale[function] {
		return answer(values, agentservice.ResultOK), nil
	}
	if ok {
		return result, nil
	}
	return agentservice.Answer{Request: function, Result: agentservice.ResultFailed}, nil
}

func (f *fakeService) Set(function string, parameter any) (agentservice.Answer, error) {
	if f.setErr != nil {
		return agentservice.Answer{}, f.setErr
	}

	values, _ := parameter.(map[string]any)
	f.writes = append(f.writes, fakeWrite{Function: function, Parameter: values})
	f.written[function] = values

	if result, ok := f.answers[function]; ok {
		return result, nil
	}

	return answer(parameter, agentservice.ResultOK), nil
}

func (f *fakeService) MonitorData() (agentservice.Answer, error) {
	if f.getErr != nil {
		return agentservice.Answer{}, f.getErr
	}
	return f.monitor, nil
}

func (f *fakeService) Capability(function string) (agentservice.Answer, error) {
	return f.Get(function)
}

func (f *fakeService) Close() error {
	f.closed = true
	return nil
}

func newBackend(t *testing.T, service *fakeService) *Backend {
	t.Helper()
	return NewWithService(service)
}
func TestSensorsComeFromTheReadingsOfTheService(t *testing.T) {
	service := newFakeService()
	service.monitor = answer(map[string]any{
		"CPU_TEMPERATURE":  90,
		"CPU_FANSPEED":     2722,
		"SYS1_TEMPERATURE": 63,
		"GPU1_TEMPERATURE": 58,
		"GPU1_FANSPEED":    2705,
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	sensors, err := backend.Sensors()
	if err != nil {
		t.Fatalf("读取传感器失败: %v", err)
	}
	if sensors.CPUTempC != 90 {
		t.Errorf("处理器温度是 %d，不是 90", sensors.CPUTempC)
	}
	if sensors.CPUFanRPM != 2722 {
		t.Errorf("处理器风扇转速是 %d，不是 2722", sensors.CPUFanRPM)
	}
	if sensors.SystemTempC != 63 {
		t.Errorf("主板温度是 %d，不是 63", sensors.SystemTempC)
	}
	if sensors.GPUTempC != 58 {
		t.Errorf("显卡温度是 %d，不是 58", sensors.GPUTempC)
	}
	if sensors.GPUFanRPM != 2705 {
		t.Errorf("显卡风扇转速是 %d，不是 2705", sensors.GPUFanRPM)
	}
}

func TestSensorsFailWhenTheServiceIsGone(t *testing.T) {
	service := newFakeService()
	service.getErr = errors.New("服务不可达")

	backend := newBackend(t, service)

	if _, err := backend.Sensors(); err == nil {
		t.Fatal("服务不可达时不该读出传感器")
	}
}

func TestPowerModeIsReadFromTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionOperatingMode] = answer(
		map[string]any{"mode": 4}, agentservice.ResultOK)

	backend := newBackend(t, service)

	mode, err := backend.PowerMode()
	if err != nil {
		t.Fatalf("读取性能档位失败: %v", err)
	}
	if mode != domain.PowerPerformance {
		t.Fatalf("性能档位是 %v，不是性能档", mode)
	}
}

func TestPowerModeIsWrittenWithTheNumberOfTheMachine(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	if err := backend.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatalf("写入性能档位失败: %v", err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("服务收到了 %d 次写入，不是 1 次", len(service.writes))
	}
	if mode := service.writes[0].Parameter["mode"]; mode != 5 {
		t.Fatalf("写入的档位是 %v，不是 5", mode)
	}

	if err := backend.SetPowerMode(domain.PowerEco); err != nil {
		t.Fatalf("写入性能档位失败: %v", err)
	}
	if mode := service.writes[1].Parameter["mode"]; mode != 6 {
		t.Fatalf("写入的档位是 %v，不是 6", mode)
	}
}

func TestGPUModeCountsFromZeroInTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionGPUMode] = answer(
		map[string]any{"mode": 1}, agentservice.ResultOK)

	backend := newBackend(t, service)

	mode, err := backend.GPUMode()
	if err != nil {
		t.Fatalf("读取显卡模式失败: %v", err)
	}
	if mode != domain.GPUModeDiscrete {
		t.Fatalf("显卡模式是 %v，不是独显", mode)
	}

	discrete, err := backend.DiscreteGPU()
	if err != nil {
		t.Fatalf("读取独显状态失败: %v", err)
	}
	if !discrete {
		t.Fatal("独显模式下的独显状态不该是关着的")
	}
}

func TestGPUModeIsWrittenWithTheNumberOfTheService(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	if err := backend.SetGPUMode(domain.GPUModeHybrid); err != nil {
		t.Fatalf("写入显卡模式失败: %v", err)
	}
	if mode := service.writes[0].Parameter["mode"]; mode != 0 {
		t.Fatalf("写入的显卡模式是 %v，不是 0", mode)
	}

	if err := backend.SetGPUMode(domain.GPUModeAuto); err != nil {
		t.Fatalf("写入显卡模式失败: %v", err)
	}
	if mode := service.writes[1].Parameter["mode"]; mode != 2 {
		t.Fatalf("写入的显卡模式是 %v，不是 2", mode)
	}
}

func TestSwitchesAreReadAndWrittenWithTheirState(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionWinKey] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)
	service.answers[agentservice.FunctionLCDOverdrive] = answer(
		map[string]any{"status": 0}, agentservice.ResultOK)
	service.answers[agentservice.FunctionBootSound] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)

	backend := newBackend(t, service)

	overdrive, err := backend.LCDOverdrive()
	if err != nil {
		t.Fatalf("读取屏幕超频失败: %v", err)
	}
	if overdrive {
		t.Fatal("屏幕超频的编号是 0，不该报成打开")
	}

	sound, err := backend.BootSound()
	if err != nil {
		t.Fatalf("读取开机音效失败: %v", err)
	}
	if !sound {
		t.Fatal("开机音效的编号是 1，不该报成关着")
	}

	if err := backend.SetLCDOverdrive(true); err != nil {
		t.Fatalf("写入屏幕超频失败: %v", err)
	}
	if status := service.writes[0].Parameter["status"]; status != 1 {
		t.Fatalf("打开屏幕超频时写入的编号是 %v，不是 1", status)
	}

	if err := backend.SetBootSound(false); err != nil {
		t.Fatalf("写入开机音效失败: %v", err)
	}
	if status := service.writes[1].Parameter["status"]; status != 0 {
		t.Fatalf("关掉开机音效时写入的编号是 %v，不是 0", status)
	}
}

func TestLightingIsReadFromTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs": []any{
			map[string]any{"LED_id": 0, "color": "#FF0000", "status": 1},
			map[string]any{"LED_id": 1, "color": "#00FF00", "status": 1},
			map[string]any{"LED_id": 2, "color": "#0000FF", "status": 1},
			map[string]any{"LED_id": 3, "color": "#FFFF00", "status": 1},
		},
		"brightness": 100,
		"effect":     "WAVE",
		"speed":      5,
		"direction":  1,
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	lighting, err := backend.Lighting()
	if err != nil {
		t.Fatalf("读取灯效失败: %v", err)
	}
	if lighting.Effect != domain.EffectWave {
		t.Errorf("灯效是 %v，不是波形", lighting.Effect)
	}
	if lighting.Speed != 5 {
		t.Errorf("速度是 %d，不是 5", lighting.Speed)
	}
	if lighting.Direction != 1 {
		t.Errorf("方向是 %d，不是 1", lighting.Direction)
	}
	if lighting.Brightness != 100 {
		t.Errorf("亮度是 %d，不是 100", lighting.Brightness)
	}
	if !lighting.PerZone {
		t.Error("四个分区各有一个颜色，该报成分区颜色")
	}
	if !lighting.ColourReadable || !lighting.EffectReadable {
		t.Error("原厂服务读得回颜色与灯效")
	}
	if lighting.Zones[0] != (domain.RGBColor{R: 255}) {
		t.Errorf("第一个分区的颜色是 %#v，不是纯红", lighting.Zones[0])
	}
	if lighting.Zones[3] != (domain.RGBColor{R: 255, G: 255}) {
		t.Errorf("第四个分区的颜色是 %#v，不是黄色", lighting.Zones[3])
	}
}

func TestLightingIsWrittenWithTheZonesOfTheKeyboard(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := domain.RGBConfig{
		Effect:     domain.EffectMeteor,
		Speed:      7,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	config.Zones[0] = domain.RGBColor{R: 255}
	config.Zones[1] = domain.RGBColor{G: 255}

	if err := backend.SetLighting(config); err != nil {
		t.Fatalf("写入灯效失败: %v", err)
	}

	parameter := service.writes[0].Parameter
	if effect := parameter["effect"]; effect != "METEOR" {
		t.Errorf("写入的灯效是 %v，不是 METEOR", effect)
	}
	if speed := parameter["speed"]; speed != 7 {
		t.Errorf("写入的速度是 %v，不是 7", speed)
	}
	if brightness := parameter["brightness"]; brightness != 80 {
		t.Errorf("写入的亮度是 %v，不是 80", brightness)
	}

	leds, ok := parameter["LEDs"].([]any)
	if !ok {
		t.Fatalf("写入的分区不是一组数值: %#v", parameter["LEDs"])
	}
	if len(leds) != domain.LightingZones {
		t.Fatalf("写入了 %d 个分区，不是 %d 个", len(leds), domain.LightingZones)
	}

	first, ok := leds[0].(map[string]any)
	if !ok {
		t.Fatalf("分区不是一组数值: %#v", leds[0])
	}
	if colour := first["color"]; colour != "#ff0000" {
		t.Errorf("第一个分区的颜色是 %v，不是 #ff0000", colour)
	}
}

func TestLightingIsLimitedToTheRangeOfTheMachine(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := domain.RGBConfig{
		Effect:     domain.EffectStatic,
		Speed:      20,
		Direction:  5,
		Brightness: 200,
	}
	if err := backend.SetLighting(config); err != nil {
		t.Fatalf("写入灯效失败: %v", err)
	}

	parameter := service.writes[0].Parameter
	if speed := parameter["speed"]; speed != 9 {
		t.Errorf("写入的速度是 %v，不是 9", speed)
	}
	if direction := parameter["direction"]; direction != 1 {
		t.Errorf("写入的方向是 %v，不是 1", direction)
	}
	if brightness := parameter["brightness"]; brightness != 100 {
		t.Errorf("写入的亮度是 %v，不是 100", brightness)
	}
}

func TestBacklightIsReadFromTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(
		map[string]any{"brightness": 60}, agentservice.ResultOK)
	service.answers[agentservice.FunctionBacklightTimeout] = answer(
		map[string]any{"status": 1}, agentservice.ResultOK)

	backend := newBackend(t, service)

	backlight, err := backend.Backlight()
	if err != nil {
		t.Fatalf("读取背光失败: %v", err)
	}
	if backlight.Brightness != 60 {
		t.Errorf("亮度是 %d，不是 60", backlight.Brightness)
	}
	if !backlight.Timeout {
		t.Error("超时开关的编号是 1，不该报成关着")
	}
}

func TestBacklightIsWrittenWithoutLosingTheColours(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(map[string]any{
		"LEDs": []any{
			map[string]any{"LED_id": 0, "color": "#123456", "status": 1},
		},
		"brightness": 30,
		"effect":     "STATIC",
		"speed":      3,
		"direction":  0,
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	if err := backend.SetBacklight(70, true); err != nil {
		t.Fatalf("写入背光失败: %v", err)
	}

	lighting := service.writes[0].Parameter
	if brightness := lighting["brightness"]; brightness != 70 {
		t.Errorf("写入的亮度是 %v，不是 70", brightness)
	}
	leds, ok := lighting["LEDs"].([]any)
	if !ok || len(leds) != 1 {
		t.Fatalf("写入的分区不是原来那一个: %#v", lighting["LEDs"])
	}
	first, _ := leds[0].(map[string]any)
	if colour := first["color"]; colour != "#123456" {
		t.Errorf("写入的颜色是 %v，不是 #123456", colour)
	}

	timeout := service.writes[1]
	if timeout.Function != agentservice.FunctionBacklightTimeout {
		t.Fatalf("第二次写入的是 %q，不是超时开关", timeout.Function)
	}
	if status := timeout.Parameter["status"]; status != 1 {
		t.Errorf("写入的超时编号是 %v，不是 1", status)
	}
}

func TestFansAreReadFromTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 2,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 0, "fan_custom_speed": 50, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 70, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	state, err := backend.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if state.CPUMode != domain.FanCustom {
		t.Errorf("处理器风扇的行为是 %v，不是固定占空比", state.CPUMode)
	}
	if state.CPUPercent != 50 {
		t.Errorf("处理器风扇的占空比是 %d，不是 50", state.CPUPercent)
	}
	if state.GPUMode != domain.FanAuto {
		t.Errorf("显卡风扇的行为是 %v，不是自动", state.GPUMode)
	}
	if !state.GPUModeReadable {
		t.Error("原厂服务读得回显卡风扇的行为")
	}
}

func TestFansInMaximumMode(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 1,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 0, "fan_name": "CPU"},
			map[string]any{"fan_custom_auto": 1, "fan_custom_speed": 70, "fan_name": "GPU"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	state, err := backend.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if state.CPUMode != domain.FanMax || state.GPUMode != domain.FanMax {
		t.Fatalf("模式 1 是全速，风扇的行为是 %v 和 %v", state.CPUMode, state.GPUMode)
	}
}

func TestFansInUnknownMode(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(
		map[string]any{"mode": 7}, agentservice.ResultOK)

	backend := newBackend(t, service)

	if _, err := backend.Fans(); err == nil {
		t.Fatal("不认识的风扇模式该报错")
	}
}

func TestFansAreWrittenWithTheModeOfTheService(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := domain.FanConfig{
		CPUMode:    domain.FanMax,
		GPUMode:    domain.FanMax,
		CPUPercent: 0,
		GPUPercent: 0,
	}
	if err := backend.SetFans(config); err != nil {
		t.Fatalf("写入风扇失败: %v", err)
	}
	if mode := service.writes[0].Parameter["mode"]; mode != fanModeMax {
		t.Fatalf("两个风扇都是全速时写的模式是 %v，不是 %d", mode, fanModeMax)
	}
}

func TestFansAreWrittenWithTheDutyCycles(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	config := domain.FanConfig{
		CPUMode:    domain.FanCustom,
		GPUMode:    domain.FanAuto,
		CPUPercent: 60,
	}
	if err := backend.SetFans(config); err != nil {
		t.Fatalf("写入风扇失败: %v", err)
	}

	parameter := service.writes[0].Parameter
	if mode := parameter["mode"]; mode != fanModeCustom {
		t.Fatalf("写的模式是 %v，不是 %d", mode, fanModeCustom)
	}

	fans, ok := parameter["custom_fan_data"].([]any)
	if !ok || len(fans) != 2 {
		t.Fatalf("写入的风扇不是两个: %#v", parameter["custom_fan_data"])
	}
	cpu, _ := fans[0].(map[string]any)
	if auto := cpu["fan_custom_auto"]; auto != 0 {
		t.Errorf("处理器风扇的自动标记是 %v，不是 0", auto)
	}
	if speed := cpu["fan_custom_speed"]; speed != 60 {
		t.Errorf("处理器风扇的占空比是 %v，不是 60", speed)
	}
	gpu, _ := fans[1].(map[string]any)
	if auto := gpu["fan_custom_auto"]; auto != 1 {
		t.Errorf("显卡风扇的自动标记是 %v，不是 1", auto)
	}
}

func TestFansAreCheckedAfterTheWrite(t *testing.T) {
	service := newFakeService()

	service.answers[agentservice.FunctionFanControl] = answer(
		map[string]any{"mode": fanModeAuto}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionFanControl: true}

	backend := newBackend(t, service)

	err := backend.SetFans(domain.FanConfig{
		CPUMode: domain.FanCustom,
		GPUMode: domain.FanCustom,
	})
	if err == nil {
		t.Fatal("机器没有接受这个设置时该报错")
	}
}

func TestTheServiceThatRefusesARequest(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionStickyKey] = agentservice.Answer{
		Request: agentservice.FunctionStickyKey,
		Result:  agentservice.ResultFailed,
	}

	backend := newBackend(t, service)

	if _, err := backend.get(agentservice.FunctionStickyKey); err == nil {
		t.Fatal("服务没有执行这次操作时该报错")
	}
}

func TestCloseEndsTheSessionWithTheService(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service)

	backend.Close()

	if !service.closed {
		t.Error("与服务的会话没有结束")
	}
}
