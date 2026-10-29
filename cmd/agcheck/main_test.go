package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/domain"
)

type fakeService struct {
	answers map[string]json.RawMessage
	written []string
	fail    error

	setFails bool
}

func newFakeService() *fakeService {
	return &fakeService{answers: map[string]json.RawMessage{}}
}

func (s *fakeService) answer(function, payload string) {
	s.answers[function] = json.RawMessage(payload)
}

func (s *fakeService) Get(function string) (agentservice.Answer, error) {
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	payload, ok := s.answers[function]
	if !ok {
		return agentservice.Answer{Result: "3"}, nil
	}
	return agentservice.Answer{Data: payload, Result: "0"}, nil
}

func (s *fakeService) Set(function string, value any) (agentservice.Answer, error) {
	if s.fail != nil || s.setFails {
		return agentservice.Answer{}, errNotThere{}
	}
	s.written = append(s.written, function)
	return agentservice.Answer{Result: "0"}, nil
}

func (s *fakeService) MonitorData() (agentservice.Answer, error) {
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	return agentservice.Answer{Data: s.answers["monitor"], Result: "0"}, nil
}

type fakeMachine struct {
	lighting  domain.RGBConfig
	backlight domain.BacklightState
	power     domain.PowerMode
	gpu       domain.GPUMode
	overdrive bool
	bootSound bool
	winKey    bool
	fans      domain.FanState

	failWith error
}

func newFakeMachine() *fakeMachine {
	return &fakeMachine{
		lighting:  domain.RGBConfig{Effect: domain.EffectWave, Speed: 3, Brightness: 5, PerZone: true},
		backlight: domain.BacklightState{Brightness: 5, Timeout: true},
		power:     domain.PowerBalanced,
		gpu:       domain.GPUModeDiscrete,
	}
}

func (m *fakeMachine) Instance() string { return "假机器" }

func (m *fakeMachine) Sensors() (domain.Sensors, error) {
	if m.failWith != nil {
		return domain.Sensors{}, m.failWith
	}
	return domain.Sensors{CPUTempC: 87, CPUFanRPM: 2681, SystemTempC: 61, GPUTempC: 56, GPUFanRPM: 2681}, nil
}

func (m *fakeMachine) Lighting() (domain.RGBConfig, error) {
	if m.failWith != nil {
		return domain.RGBConfig{}, m.failWith
	}
	return m.lighting, nil
}

func (m *fakeMachine) SetLighting(config domain.RGBConfig) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.lighting = config
	return nil
}

func (m *fakeMachine) Backlight() (domain.BacklightState, error) {
	if m.failWith != nil {
		return domain.BacklightState{}, m.failWith
	}
	return m.backlight, nil
}

func (m *fakeMachine) SetBacklight(brightness int, timeout bool) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.backlight = domain.BacklightState{Brightness: brightness, Timeout: timeout}
	return nil
}

func (m *fakeMachine) PowerMode() (domain.PowerMode, error) {
	if m.failWith != nil {
		return 0, m.failWith
	}
	return m.power, nil
}

func (m *fakeMachine) SetPowerMode(mode domain.PowerMode) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.power = mode
	return nil
}

func (m *fakeMachine) GPUMode() (domain.GPUMode, error) {
	if m.failWith != nil {
		return 0, m.failWith
	}
	return m.gpu, nil
}

func (m *fakeMachine) LCDOverdrive() (bool, error) {
	if m.failWith != nil {
		return false, m.failWith
	}
	return m.overdrive, nil
}

func (m *fakeMachine) SetLCDOverdrive(enabled bool) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.overdrive = enabled
	return nil
}

func (m *fakeMachine) BootSound() (bool, error) {
	if m.failWith != nil {
		return false, m.failWith
	}
	return m.bootSound, nil
}

func (m *fakeMachine) SetBootSound(enabled bool) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.bootSound = enabled
	return nil
}

func (m *fakeMachine) WinKeyLocked() (bool, error) {
	if m.failWith != nil {
		return false, m.failWith
	}
	return m.winKey, nil
}

func (m *fakeMachine) SetWinKeyLocked(locked bool) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.winKey = locked
	return nil
}

func (m *fakeMachine) Fans() (domain.FanState, error) {
	if m.failWith != nil {
		return domain.FanState{}, m.failWith
	}
	return m.fans, nil
}

func (m *fakeMachine) SetFans(config domain.FanConfig) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.fans = domain.FanState{CPUMode: config.CPUMode, GPUMode: config.GPUMode}
	return nil
}

func (m *fakeMachine) Close() {}

func capture(t *testing.T, fn func()) string {
	t.Helper()

	var buffer bytes.Buffer
	old := stdout
	stdout = &buffer
	defer func() { stdout = old }()

	fn()
	return buffer.String()
}

func TestRead(t *testing.T) {
	service := newFakeService()
	service.answer("monitor", `{"CPU_TEMPERATURE":89}`)
	service.answer(agentservice.FunctionOperatingMode, `{"mode":1}`)
	service.answer(agentservice.FunctionLighting, `{"brightness":5}`)

	out := capture(t, func() { read(service) })

	for _, want := range []string{"监视数据:", "OPERATING_MODE", "LIGHTING"} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

func TestReadOfACommandThatIsNotThere(t *testing.T) {
	service := newFakeService()

	out := capture(t, func() { read(service) })
	if !contains(out, "STICKY_KEY") {
		t.Errorf("输出里没有那一项:\n%s", out)
	}
}

func TestWriteWritesBackWhatTheServiceReports(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionOperatingMode, `{"mode":1}`)
	service.answer(agentservice.FunctionGPUMode, `{"mode":1}`)
	service.answer(agentservice.FunctionWinKey, `{"status":1}`)
	service.answer(agentservice.FunctionLCDOverdrive, `{"status":1}`)
	service.answer(agentservice.FunctionBootSound, `{"status":0}`)
	service.answer(agentservice.FunctionBacklightTimeout, `{"status":1}`)
	service.answer(agentservice.FunctionLighting, `{"brightness":5,"color":"#00aec7","effect":"WAVE"}`)
	service.answer(agentservice.FunctionFanControl,
		`{"mode":0,"custom_fan_data":[{"fan_name":"CPU","fan_custom_auto":1,"fan_custom_speed":50,"fan_index":0}]}`)

	out := capture(t, func() { write(service) })

	for _, want := range []string{
		"OPERATING_MODE: 写入成功",
		"LIGHTING: 写入成功",
		"FAN_CONTROL: 写入成功",
	} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}

	for _, item := range service.written {
		if item != agentservice.FunctionLighting {
			continue
		}
	}
}

func TestWriteOfAServiceThatDoesNotTakeTheWrite(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionOperatingMode, `{"mode":1}`)
	service.setFails = true

	out := capture(t, func() { write(service) })
	if !contains(out, "写入失败") {
		t.Errorf("输出里没有说写不进去:\n%s", out)
	}
}

func TestWriteOfAServiceThatIsNotThere(t *testing.T) {
	service := newFakeService()
	service.fail = errNotThere{}

	capture(t, func() { write(service) })
	if len(service.written) != 0 {
		t.Errorf("写了 %#v", service.written)
	}
}

func TestReadMachine(t *testing.T) {
	device := newFakeMachine()

	out := capture(t, func() { readMachine(device) })

	for _, want := range []string{"机器: 假机器", "处理器 87 度", "性能档位:", "显卡模式:", "灯效"} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

func TestReadMachineOfAMachineThatIsNotThere(t *testing.T) {
	device := newFakeMachine()
	device.failWith = errNotThere{}

	out := capture(t, func() { readMachine(device) })

	for _, want := range []string{"读取传感器失败", "读取性能档位失败", "读取灯效失败", "读取风扇失败"} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

func TestReportOfARejectedWrite(t *testing.T) {
	out := capture(t, func() {
		report("STICKY_KEY", agentservice.Answer{Result: "4"}, nil)
	})
	if !contains(out, "STICKY_KEY") {
		t.Errorf("输出里没有那一项:\n%s", out)
	}
}

func TestReportOfAWriteThatFailed(t *testing.T) {
	out := capture(t, func() {
		report("LIGHTING", agentservice.Answer{}, errNotThere{})
	})
	if !contains(out, "写入失败") {
		t.Errorf("输出里没有说写失败:\n%s", out)
	}
}

func TestPrintJSON(t *testing.T) {
	out := capture(t, func() { printJSON(json.RawMessage(`{"a":1}`)) })
	if !contains(out, `"a": 1`) {
		t.Errorf("输出不是那个文档:\n%s", out)
	}

	out = capture(t, func() { printJSON(json.RawMessage("not a document")) })
	if !contains(out, "not a document") {
		t.Errorf("读不出来的时候该照原样打出来:\n%s", out)
	}
}

type errNotThere struct{}

func (errNotThere) Error() string { return "服务不可达" }

func contains(text, name string) bool {
	for i := 0; i+len(name) <= len(text); i++ {
		if text[i:i+len(name)] == name {
			return true
		}
	}
	return false
}
