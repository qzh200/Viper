package probe

import (
	"encoding/json"
	"errors"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/domain"
	"viper/internal/machine"
)

type fakeService struct {
	answers map[string]json.RawMessage

	written []written

	fail error
}

type written struct {
	function string
	value    any
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
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	s.written = append(s.written, written{function: function, value: value})
	return agentservice.Answer{Result: "0"}, nil
}

func (s *fakeService) MonitorData() (agentservice.Answer, error) {
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	return agentservice.Answer{Data: s.answers["monitor"], Result: "0"}, nil
}

func TestLightingPayloadKeepsTheGlobalColour(t *testing.T) {
	payload := LightingPayload(LightingReading{
		Color:      "#00AEC7",
		Brightness: 5,
		Effect:     "WAVE",
		Speed:      3,
		Direction:  1,
	})

	if payload["color"] != "#00AEC7" {
		t.Errorf("全局颜色没写回去: %#v", payload["color"])
	}
	if _, present := payload["LEDs"]; present {
		t.Errorf("没有分区的时候不该写分区数组: %#v", payload["LEDs"])
	}
	if payload["brightness"] != 5 || payload["effect"] != "WAVE" {
		t.Errorf("别的几项没写回去: %#v", payload)
	}
}

func TestLightingPayloadKeepsTheZones(t *testing.T) {
	payload := LightingPayload(LightingReading{
		LEDs: []LightingLED{
			{LEDID: 0, Color: "#00AEC7"},
			{LEDID: 1, Color: "#000000"},
		},
		Brightness: 5,
		Effect:     "STATIC",
	})

	leds, ok := payload["LEDs"].([]any)
	if !ok {
		t.Fatalf("分区数组没写回去: %#v", payload["LEDs"])
	}
	if len(leds) != 2 {
		t.Fatalf("分区数不对: %#v", leds)
	}
	first, ok := leds[0].(map[string]any)
	if !ok {
		t.Fatalf("分区的形状不对: %#v", leds[0])
	}
	if first["color"] != "#00AEC7" || first["LED_id"] != 0 {
		t.Errorf("第一个分区的颜色不对: %#v", first)
	}
	if _, present := payload["color"]; present {
		t.Errorf("分区设置的时候不该写全局颜色: %#v", payload["color"])
	}
}

func TestReadLighting(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionLighting,
		`{"brightness":5,"color":"#00aec7","effect":"WAVE","speed":3,"direction":1}`)

	reading, err := ReadLighting(service)
	if err != nil {
		t.Fatal(err)
	}
	if reading.Color != "#00aec7" || reading.Brightness != 5 || reading.Effect != "WAVE" {
		t.Errorf("读错了: %#v", reading)
	}
	if reading.Speed != 3 || reading.Direction != 1 {
		t.Errorf("读错了: %#v", reading)
	}
}

func TestReadLightingOfACommandThatIsNotThere(t *testing.T) {
	service := newFakeService()

	if _, err := ReadLighting(service); err == nil {
		t.Error("服务不实现这一项的时候该报错")
	}
}

func TestWriteLighting(t *testing.T) {
	service := newFakeService()
	reading := LightingReading{Color: "#00AEC7", Brightness: 5, Effect: "WAVE"}

	if err := WriteLighting(service, reading); err != nil {
		t.Fatal(err)
	}
	if len(service.written) != 1 {
		t.Fatalf("写了几次: %#v", service.written)
	}
	if service.written[0].function != agentservice.FunctionLighting {
		t.Errorf("写错了命令: %s", service.written[0].function)
	}
	payload, ok := service.written[0].value.(map[string]any)
	if !ok {
		t.Fatalf("负载的形状不对: %#v", service.written[0].value)
	}
	if payload["color"] != "#00AEC7" {
		t.Errorf("颜色没写回去: %#v", payload)
	}
}

func TestFanPayload(t *testing.T) {
	payload := FanPayload(2,
		FanData{Name: "CPU", Auto: 0, Speed: 50, Index: 0},
		FanData{Name: "GPU", Auto: 0, Speed: 40, Index: 0})

	if payload["mode"] != 2 {
		t.Errorf("模式不对: %#v", payload["mode"])
	}
	fans, ok := payload["custom_fan_data"].([]any)
	if !ok {
		t.Fatalf("风扇数组的形状不对: %#v", payload["custom_fan_data"])
	}
	if len(fans) != 2 {
		t.Fatalf("风扇数不对: %#v", fans)
	}
	first, ok := fans[0].(map[string]any)
	if !ok {
		t.Fatalf("风扇的形状不对: %#v", fans[0])
	}
	if first["fan_name"] != "CPU" || first["fan_custom_auto"] != 0 || first["fan_custom_speed"] != 50 {
		t.Errorf("处理器的风扇不对: %#v", first)
	}
	second := fans[1].(map[string]any)
	if second["fan_name"] != "GPU" || second["fan_custom_speed"] != 40 {
		t.Errorf("显卡的风扇不对: %#v", second)
	}
}

func TestReadFan(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl,
		`{"mode":0,"custom_fan_data":[{"fan_name":"CPU","fan_custom_auto":1,"fan_custom_speed":50,"fan_index":0}]}`)

	reading, err := ReadFan(service)
	if err != nil {
		t.Fatal(err)
	}
	if reading.Mode != 0 || len(reading.Fans) != 1 {
		t.Fatalf("读错了: %#v", reading)
	}
	if reading.Fans[0].Name != "CPU" || reading.Fans[0].Auto != 1 || reading.Fans[0].Speed != 50 {
		t.Errorf("风扇读错了: %#v", reading.Fans[0])
	}
}

func TestWriteFans(t *testing.T) {
	service := newFakeService()
	reading := FanReading{
		Mode: 0,
		Fans: []FanData{
			{Name: "CPU", Auto: 1, Speed: 50, Index: 0},
			{Name: "GPU", Auto: 0, Speed: 30, Index: 0},
		},
	}

	if err := WriteFans(service, reading); err != nil {
		t.Fatal(err)
	}
	payload, ok := service.written[0].value.(map[string]any)
	if !ok {
		t.Fatalf("负载的形状不对: %#v", service.written[0].value)
	}
	fans, ok := payload["custom_fan_data"].([]any)
	if !ok || len(fans) != 2 {
		t.Fatalf("风扇数组不对: %#v", payload["custom_fan_data"])
	}
	second := fans[1].(map[string]any)
	if second["fan_name"] != "GPU" || second["fan_custom_auto"] != 0 || second["fan_custom_speed"] != 30 {
		t.Errorf("显卡的风扇没写回去: %#v", second)
	}
	if payload["mode"] != 0 {
		t.Errorf("模式没写回去: %#v", payload["mode"])
	}
}

func TestWinKey(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionWinKey, `{"status":1}`)

	status, err := ReadWinKey(service)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != 1 {
		t.Errorf("状态读错了: %d", status.Status)
	}

	payload := WinKeyPayload(0)
	if payload["status"] != 0 {
		t.Errorf("负载不对: %#v", payload)
	}

	if err := WriteWinKey(service, 0); err != nil {
		t.Fatal(err)
	}
	if service.written[0].function != agentservice.FunctionWinKey {
		t.Errorf("写错了命令: %s", service.written[0].function)
	}
}

func TestGraphicsLines(t *testing.T) {
	data := json.RawMessage(`{
		"CPU_TEMPERATURE": 89,
		"GPU1_FANSPEED": 2729,
		"GPU1_FREQUENCY": 375,
		"RAM_TOTAL": 32487
	}`)

	lines := GraphicsLines(data)
	text := ""
	for _, line := range lines {
		text += line + "\n"
	}
	for _, want := range []string{"GPU1_FANSPEED", "GPU1_FREQUENCY"} {
		if !contains(text, want) {
			t.Errorf("输出里没有 %s:\n%s", want, text)
		}
	}
	if contains(text, "CPU_TEMPERATURE") {
		t.Errorf("输出里不该有处理器那一项:\n%s", text)
	}
}

func TestGraphicsLinesOfSomethingBroken(t *testing.T) {
	if lines := GraphicsLines(json.RawMessage("not a document")); lines != nil {
		t.Errorf("该读不出来: %#v", lines)
	}
}

func TestMonitorValue(t *testing.T) {
	data := json.RawMessage(`{"CPU_FANSPEED": 2719, "CPU_TEMPERATURE": 89}`)

	value, ok := MonitorValue(data, "CPU_FANSPEED")
	if !ok {
		t.Fatal("该读到风扇转速")
	}
	if value != float64(2719) {
		t.Errorf("风扇转速不对: %v", value)
	}
	if _, ok := MonitorValue(data, "NOPE"); ok {
		t.Error("没有的那一项不该读到")
	}
	if _, ok := MonitorValue(json.RawMessage("broken"), "CPU_FANSPEED"); ok {
		t.Error("读不出来的时候不该报有值")
	}
}

func contains(text, name string) bool {
	for i := 0; i+len(name) <= len(text); i++ {
		if text[i:i+len(name)] == name {
			return true
		}
	}
	return false
}

type fakeMachine struct {
	lighting  domain.RGBConfig
	backlight domain.BacklightState
	power     domain.PowerMode
	overdrive bool
	bootSound bool
	winKey    bool
	fans      domain.FanState

	failWith error

	applied []string
}

func newFakeMachine() *fakeMachine {
	return &fakeMachine{
		lighting:  domain.RGBConfig{Effect: domain.EffectWave, Speed: 3, Brightness: 5, PerZone: true},
		backlight: domain.BacklightState{Brightness: 5, Timeout: true},
		power:     domain.PowerBalanced,
	}
}

func (m *fakeMachine) Instance() string { return "假机器" }

func (m *fakeMachine) Sensors() (domain.Sensors, error) {
	return domain.Sensors{CPUTempC: 87, CPUFanRPM: 2681}, nil
}

func (m *fakeMachine) Lighting() (domain.RGBConfig, error) { return m.lighting, nil }

func (m *fakeMachine) SetLighting(config domain.RGBConfig) error {
	m.applied = append(m.applied, "lighting")
	if m.failWith != nil {
		return m.failWith
	}
	m.lighting = config
	return nil
}

func (m *fakeMachine) Backlight() (domain.BacklightState, error) { return m.backlight, nil }

func (m *fakeMachine) SetBacklight(brightness int, timeout bool) error {
	m.applied = append(m.applied, "backlight")
	if m.failWith != nil {
		return m.failWith
	}
	m.backlight = domain.BacklightState{Brightness: brightness, Timeout: timeout}
	return nil
}

func (m *fakeMachine) PowerMode() (domain.PowerMode, error) { return m.power, nil }

func (m *fakeMachine) SetPowerMode(mode domain.PowerMode) error {
	m.applied = append(m.applied, "power")
	if m.failWith != nil {
		return m.failWith
	}
	m.power = mode
	return nil
}

func (m *fakeMachine) LCDOverdrive() (bool, error) { return m.overdrive, nil }

func (m *fakeMachine) SetLCDOverdrive(enabled bool) error {
	m.applied = append(m.applied, "overdrive")
	if m.failWith != nil {
		return m.failWith
	}
	m.overdrive = enabled
	return nil
}

func (m *fakeMachine) BootSound() (bool, error) { return m.bootSound, nil }

func (m *fakeMachine) SetBootSound(enabled bool) error {
	m.applied = append(m.applied, "bootsound")
	if m.failWith != nil {
		return m.failWith
	}
	m.bootSound = enabled
	return nil
}

func (m *fakeMachine) WinKeyLocked() (bool, error) { return m.winKey, nil }

func (m *fakeMachine) SetWinKeyLocked(locked bool) error {
	m.applied = append(m.applied, "winkey")
	if m.failWith != nil {
		return m.failWith
	}
	m.winKey = locked
	return nil
}

func (m *fakeMachine) Fans() (domain.FanState, error) { return m.fans, nil }

func (m *fakeMachine) SetFans(config domain.FanConfig) error {
	m.applied = append(m.applied, "fans")
	if m.failWith != nil {
		return m.failWith
	}
	m.fans = domain.FanState{CPUMode: config.CPUMode, GPUMode: config.GPUMode}
	return nil
}

func TestVerifyChangesAndRestores(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl, `{"mode":0,"custom_fan_data":[]}`)
	device := newFakeMachine()

	original, results, err := Verify(device, service)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 9 {
		t.Fatalf("验了 %d 项，不是九项", len(results))
	}
	for _, result := range results {
		if !result.OK() {
			t.Errorf("%s 没有通过: %v / %v", result.Name, result.ApplyErr, result.RestoreErr)
		}
	}

	writes := map[string]int{}
	for _, name := range device.applied {
		writes[name]++
	}
	for _, name := range []string{"lighting", "backlight", "power", "overdrive", "bootsound", "winkey"} {
		if writes[name] < 2 {
			t.Errorf("%s 写了 %d 次", name, writes[name])
		}
	}
	if writes["fans"] != 1 {
		t.Errorf("风扇写了 %d 次", writes["fans"])
	}

	restored := false
	for _, item := range service.written {
		if item.function == agentservice.FunctionFanControl {
			restored = true
		}
	}
	if !restored {
		t.Error("风扇没有按服务记着的样子写回去")
	}

	if device.lighting.Effect != original.Lighting.Effect {
		t.Errorf("灯效没有还原: %v", device.lighting.Effect)
	}
	if device.power != original.Power {
		t.Errorf("档位没有还原: %v", device.power)
	}
	if device.backlight != original.Backlight {
		t.Errorf("背光没有还原: %v", device.backlight)
	}
}

func TestVerifyReportsAFailedWrite(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl, `{"mode":0,"custom_fan_data":[]}`)
	device := newFakeMachine()
	device.failWith = errors.New("机器没有接受")

	_, results, err := Verify(device, service)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.OK() {
			t.Errorf("%s 该报失败", result.Name)
		}
	}
}

func TestVerifySkipsTheSettingThatNeedsTheFirmware(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl, `{"mode":0,"custom_fan_data":[]}`)
	device := newFakeMachine()

	_, results, err := Verify(device, service)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Name == "Windows 键" && result.Skipped {
			t.Error("替身没有报错，这一项不该跳过")
		}
	}
}

func TestVerifyReportsAMissingFanPayload(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl, `not a document`)
	device := newFakeMachine()

	if _, _, err := Verify(device, service); err == nil {
		t.Error("读不出风扇配置的时候该报错")
	}
}

var _ Machine = (*machine.Backend)(nil)

var _ Service = (*agentservice.Client)(nil)
