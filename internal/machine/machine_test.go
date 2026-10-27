package machine

import (
	"encoding/json"
	"errors"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

// fakeWrite is a write that reached the stand for the service.
type fakeWrite struct {
	// Function is the name of the setting.
	Function string
	// Parameter is the value that was written.
	Parameter map[string]any
}

// fakeService is a stand for the AcerAgentService.
//
// The answers are given per function, a function without an answer is
// answered with the value that was written to it, which is what the service
// of the original software does.
type fakeService struct {
	// answers holds the answer of a function.
	answers map[string]agentservice.Answer
	// written holds the value that was written to a function.
	written map[string]map[string]any
	// monitor is the answer of the readings.
	monitor agentservice.Answer
	// writes holds the writes that arrived, in order.
	writes []fakeWrite
	// closed records that the session was ended.
	closed bool
	// getErr is the error of every read.
	getErr error
	// setErr is the error of every write.
	setErr error
	// stale names the functions whose readback stays at the programmed
	// answer even after a write. It stands for a machine that did not
	// take the setting.
	stale map[string]bool
}

func newFakeService() *fakeService {
	return &fakeService{
		answers: map[string]agentservice.Answer{},
		written: map[string]map[string]any{},
	}
}

// answer builds the answer of a request.
func answer(data any, result string) agentservice.Answer {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return agentservice.Answer{Data: raw, Result: result}
}

// Get asks for the value of a setting.
func (f *fakeService) Get(function string) (agentservice.Answer, error) {
	if f.getErr != nil {
		return agentservice.Answer{}, f.getErr
	}
	result, ok := f.answers[function]
	// The service of the original software answers with the value that was
	// written to it, so the stand does the same. A function that is marked
	// as stale keeps the answer it was given, which stands for a machine
	// that did not take the setting.
	if values, written := f.written[function]; written && !f.stale[function] {
		return answer(values, agentservice.ResultOK), nil
	}
	if ok {
		return result, nil
	}
	return agentservice.Answer{Request: function, Result: agentservice.ResultFailed}, nil
}

// Set writes a setting.
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
	// The service of the original software answers with the value that was
	// written, so the stand does the same.
	return answer(parameter, agentservice.ResultOK), nil
}

// MonitorData asks for the live numbers of the machine.
func (f *fakeService) MonitorData() (agentservice.Answer, error) {
	if f.getErr != nil {
		return agentservice.Answer{}, f.getErr
	}
	return f.monitor, nil
}

// Capability asks for an ability of the machine.
func (f *fakeService) Capability(function string) (agentservice.Answer, error) {
	return f.Get(function)
}

// Close ends the session.
func (f *fakeService) Close() error {
	f.closed = true
	return nil
}

// fakeFirmware is a stand for the ACPI provider.
type fakeFirmware struct {
	coolBoost  bool
	fanTable   uint64
	overclock  bool
	overclockG bool
	buffer     []byte
	reset      bool
	closed     bool
	opened     int
	err        error
	// winKey is the state of the lock of the Windows key.
	winKey bool
}

func (f *fakeFirmware) WinKeyLocked() (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.winKey, nil
}

func (f *fakeFirmware) SetWinKeyLocked(locked bool) error {
	if f.err != nil {
		return f.err
	}
	f.winKey = locked
	return nil
}

func (f *fakeFirmware) CoolBoostEnabled() (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.coolBoost, nil
}

func (f *fakeFirmware) CoolBoost(enabled bool) error {
	if f.err != nil {
		return f.err
	}
	f.coolBoost = enabled
	return nil
}

func (f *fakeFirmware) FanTable() (uint64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.fanTable, nil
}

func (f *fakeFirmware) OverclockCapability() (bool, bool, error) {
	if f.err != nil {
		return false, false, f.err
	}
	return f.overclockG, f.overclock, nil
}

func (f *fakeFirmware) BacklightBuffer() ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.buffer, nil
}

func (f *fakeFirmware) ResetLighting() error {
	if f.err != nil {
		return f.err
	}
	f.reset = true
	return nil
}

func (f *fakeFirmware) Close() { f.closed = true }

// newBackend builds a backend on top of the stands.
func newBackend(t *testing.T, service *fakeService, firmware *fakeFirmware) *Backend {
	t.Helper()

	backend := NewWithService(service, func() (hardware.Firmware, error) {
		if firmware == nil {
			return nil, errors.New("没有固件")
		}
		firmware.opened++
		return firmware, nil
	})
	return backend
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

	backend := newBackend(t, service, nil)

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

	backend := newBackend(t, service, nil)

	if _, err := backend.Sensors(); err == nil {
		t.Fatal("服务不可达时不该读出传感器")
	}
}

func TestPowerModeIsReadFromTheService(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionOperatingMode] = answer(
		map[string]any{"mode": 4}, agentservice.ResultOK)

	backend := newBackend(t, service, nil)

	mode, err := backend.PowerMode()
	if err != nil {
		t.Fatalf("读取性能档位失败: %v", err)
	}
	if mode != hardware.PowerPerformance {
		t.Fatalf("性能档位是 %v，不是性能档", mode)
	}
}

func TestPowerModeIsWrittenWithTheNumberOfTheMachine(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service, nil)

	if err := backend.SetPowerMode(hardware.PowerTurbo); err != nil {
		t.Fatalf("写入性能档位失败: %v", err)
	}
	if len(service.writes) != 1 {
		t.Fatalf("服务收到了 %d 次写入，不是 1 次", len(service.writes))
	}
	if mode := service.writes[0].Parameter["mode"]; mode != 5 {
		t.Fatalf("写入的档位是 %v，不是 5", mode)
	}

	if err := backend.SetPowerMode(hardware.PowerEco); err != nil {
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

	backend := newBackend(t, service, nil)

	mode, err := backend.GPUMode()
	if err != nil {
		t.Fatalf("读取显卡模式失败: %v", err)
	}
	if mode != hardware.GPUModeDiscrete {
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
	backend := newBackend(t, service, nil)

	if err := backend.SetGPUMode(hardware.GPUModeHybrid); err != nil {
		t.Fatalf("写入显卡模式失败: %v", err)
	}
	if mode := service.writes[0].Parameter["mode"]; mode != 0 {
		t.Fatalf("写入的显卡模式是 %v，不是 0", mode)
	}

	if err := backend.SetGPUMode(hardware.GPUModeAuto); err != nil {
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

	backend := newBackend(t, service, nil)

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

	backend := newBackend(t, service, nil)

	lighting, err := backend.Lighting()
	if err != nil {
		t.Fatalf("读取灯效失败: %v", err)
	}
	if lighting.Effect != hardware.EffectWave {
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
	if lighting.Zones[0] != (hardware.RGBColor{R: 255}) {
		t.Errorf("第一个分区的颜色是 %#v，不是纯红", lighting.Zones[0])
	}
	if lighting.Zones[3] != (hardware.RGBColor{R: 255, G: 255}) {
		t.Errorf("第四个分区的颜色是 %#v，不是黄色", lighting.Zones[3])
	}
}

func TestLightingIsWrittenWithTheZonesOfTheKeyboard(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service, nil)

	config := hardware.RGBConfig{
		Effect:     hardware.EffectMeteor,
		Speed:      7,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	config.Zones[0] = hardware.RGBColor{R: 255}
	config.Zones[1] = hardware.RGBColor{G: 255}

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
	if len(leds) != hardware.LightingZones {
		t.Fatalf("写入了 %d 个分区，不是 %d 个", len(leds), hardware.LightingZones)
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
	backend := newBackend(t, service, nil)

	config := hardware.RGBConfig{
		Effect:     hardware.EffectStatic,
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

	backend := newBackend(t, service, nil)

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

	backend := newBackend(t, service, nil)

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

	backend := newBackend(t, service, nil)

	state, err := backend.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if state.CPUMode != hardware.FanCustom {
		t.Errorf("处理器风扇的行为是 %v，不是固定占空比", state.CPUMode)
	}
	if state.CPUPercent != 50 {
		t.Errorf("处理器风扇的占空比是 %d，不是 50", state.CPUPercent)
	}
	if state.GPUMode != hardware.FanAuto {
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

	backend := newBackend(t, service, nil)

	state, err := backend.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if state.CPUMode != hardware.FanMax || state.GPUMode != hardware.FanMax {
		t.Fatalf("模式 1 是全速，风扇的行为是 %v 和 %v", state.CPUMode, state.GPUMode)
	}
}

func TestFansInUnknownMode(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(
		map[string]any{"mode": 7}, agentservice.ResultOK)

	backend := newBackend(t, service, nil)

	if _, err := backend.Fans(); err == nil {
		t.Fatal("不认识的风扇模式该报错")
	}
}

func TestFansAreWrittenWithTheModeOfTheService(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service, nil)

	config := hardware.FanConfig{
		CPUMode:    hardware.FanMax,
		GPUMode:    hardware.FanMax,
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
	backend := newBackend(t, service, nil)

	config := hardware.FanConfig{
		CPUMode:    hardware.FanCustom,
		GPUMode:    hardware.FanAuto,
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
	// The service answers with another mode than the one that was written.
	service.answers[agentservice.FunctionFanControl] = answer(
		map[string]any{"mode": fanModeAuto}, agentservice.ResultOK)
	service.stale = map[string]bool{agentservice.FunctionFanControl: true}

	backend := newBackend(t, service, nil)

	err := backend.SetFans(hardware.FanConfig{
		CPUMode: hardware.FanCustom,
		GPUMode: hardware.FanCustom,
	})
	if err == nil {
		t.Fatal("机器没有接受这个设置时该报错")
	}
}

func TestFansNeedTheFirmwareForTheBoost(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service, nil)

	err := backend.SetFans(hardware.FanConfig{
		CPUMode:   hardware.FanAuto,
		GPUMode:   hardware.FanAuto,
		CoolBoost: true,
	})
	if err == nil {
		t.Fatal("加速要固件那一侧，够不到时该报错")
	}
	if !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
	if len(service.writes) != 0 {
		t.Fatal("够不到固件时不该改风扇")
	}
}

func TestFirmwareAnswersTheSettingsTheServiceDoesNotKnow(t *testing.T) {
	service := newFakeService()
	firmware := &fakeFirmware{
		coolBoost:  true,
		fanTable:   hardware.FanTableBoost,
		overclock:  true,
		overclockG: false,
		buffer:     []byte{0, 0x64, 0, 0, 0},
	}
	backend := newBackend(t, service, firmware)

	boost, err := backend.CoolBoostEnabled()
	if err != nil {
		t.Fatalf("读取加速失败: %v", err)
	}
	if !boost {
		t.Error("加速是打开着的")
	}

	if err := backend.CoolBoost(false); err != nil {
		t.Fatalf("写入加速失败: %v", err)
	}
	if firmware.coolBoost {
		t.Error("加速没有写进去")
	}

	table, err := backend.FanTable()
	if err != nil {
		t.Fatalf("读取风扇表失败: %v", err)
	}
	if table != hardware.FanTableBoost {
		t.Errorf("风扇表是 %d，不是 %d", table, hardware.FanTableBoost)
	}

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatalf("读取超频能力失败: %v", err)
	}
	if gpu || !cpu {
		t.Errorf("超频能力是显卡 %v 处理器 %v，不是显卡假处理器真", gpu, cpu)
	}

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Fatalf("读取背光缓冲区失败: %v", err)
	}
	if len(buffer) != 5 {
		t.Errorf("背光缓冲区是 %d 个字节，不是 5 个", len(buffer))
	}

	if err := backend.ResetLighting(); err != nil {
		t.Fatalf("复位灯效失败: %v", err)
	}
	if !firmware.reset {
		t.Error("灯效没有复位")
	}
}

func TestFirmwareIsOpenedOnlyOnce(t *testing.T) {
	service := newFakeService()
	firmware := &fakeFirmware{}
	backend := newBackend(t, service, firmware)

	for i := 0; i < 3; i++ {
		if _, err := backend.CoolBoostEnabled(); err != nil {
			t.Fatalf("第 %d 次读取加速失败: %v", i+1, err)
		}
	}
	if firmware.opened != 1 {
		t.Fatalf("固件开了 %d 次，不是 1 次", firmware.opened)
	}
}

func TestSettingsThatNeedTheFirmwareReportIt(t *testing.T) {
	service := newFakeService()
	backend := newBackend(t, service, nil)

	if _, _, err := backend.OverclockCapability(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
	if _, err := backend.FanTable(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
	if _, err := backend.BacklightBuffer(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
	if err := backend.ResetLighting(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
	if _, err := backend.CoolBoostEnabled(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
}

func TestAFirmwareThatIsNotThere(t *testing.T) {
	service := newFakeService()
	backend := NewWithService(service, func() (hardware.Firmware, error) {
		return nil, errors.New("没有实例")
	})

	if _, err := backend.CoolBoostEnabled(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
}

func TestAFirmwareThatIsAnEmptyPointer(t *testing.T) {
	service := newFakeService()
	backend := NewWithService(service, func() (hardware.Firmware, error) {
		// An interface that holds an empty pointer is not equal to nil, so
		// a guard that only asks the interface would let the call through.
		var firmware *fakeFirmware
		return firmware, nil
	})

	if _, err := backend.CoolBoostEnabled(); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("错误是 %v，不是 %v", err, ErrNoProvider)
	}
}

func TestTheServiceThatRefusesARequest(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionStickyKey] = agentservice.Answer{
		Request: agentservice.FunctionStickyKey,
		Result:  agentservice.ResultFailed,
	}

	backend := newBackend(t, service, nil)

	if _, err := backend.get(agentservice.FunctionStickyKey); err == nil {
		t.Fatal("服务没有执行这次操作时该报错")
	}
}

func TestCloseReleasesBothConnections(t *testing.T) {
	service := newFakeService()
	firmware := &fakeFirmware{}
	backend := newBackend(t, service, firmware)

	if _, err := backend.CoolBoostEnabled(); err != nil {
		t.Fatalf("读取加速失败: %v", err)
	}

	backend.Close()

	if !service.closed {
		t.Error("与服务的会话没有结束")
	}
	if !firmware.closed {
		t.Error("与固件的连接没有结束")
	}
}
