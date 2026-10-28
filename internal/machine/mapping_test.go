package machine

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"viper/internal/agentservice"
	"viper/internal/hardware"
)

func TestTheNamesOfThePatterns(t *testing.T) {

	names := map[string]hardware.LightingEffect{
		"STATIC":    hardware.EffectStatic,
		"BREATHING": hardware.EffectBreathing,
		"NEON":      hardware.EffectNeon,
		"WAVE":      hardware.EffectWave,
		"SHIFTING":  hardware.EffectSwitch,
		"ZOOM":      hardware.EffectScale,
		"METEOR":    hardware.EffectMeteor,
		"TWINKLING": hardware.EffectFlash,
	}

	for name, effect := range names {
		if got := serviceEffect(name); got != effect {
			t.Errorf("名字 %q 对应的是 %v，不是 %v", name, got, effect)
		}
		if got := effectName(effect); got != name {
			t.Errorf("灯效 %v 的名字是 %q，不是 %q", effect, got, name)
		}
	}

	if got := serviceEffect("DIRECT"); got != hardware.EffectStatic {
		t.Errorf("DIRECT 对应的是 %v，不是静态", got)
	}
	if got := serviceEffect(""); got != hardware.EffectStatic {
		t.Errorf("空名字对应的是 %v，不是静态", got)
	}
	if got := effectName(hardware.LightingEffect(99)); got != "STATIC" {
		t.Errorf("不认识的灯效 %v 的名字是 %q，不是 STATIC", hardware.LightingEffect(99), got)
	}
}

func TestTheNumberOfTheGraphicsModes(t *testing.T) {

	if got := serviceGPUMode(0); got != hardware.GPUModeHybrid {
		t.Errorf("编号 0 是 %v，不是混合", got)
	}
	if got := serviceGPUMode(1); got != hardware.GPUModeDiscrete {
		t.Errorf("编号 1 是 %v，不是独显", got)
	}
	if got := serviceGPUMode(2); got != hardware.GPUModeAuto {
		t.Errorf("编号 2 是 %v，不是自动", got)
	}
	if got := serviceGPUMode(7); got != hardware.GPUModeHybrid {
		t.Errorf("不认识的编号 7 是 %v，不是混合", got)
	}
}

func TestColoursThatAreNotColours(t *testing.T) {
	colour, ok := parseColour("#00FF00")
	if !ok {
		t.Fatal("六位颜色不该被丢掉")
	}
	if colour != (hardware.RGBColor{G: 255}) {
		t.Errorf("颜色是 %#v，不是纯绿", colour)
	}

	for _, text := range []string{"", "#12345", "#GGGGGG", "#00FF0000", "红色"} {
		if _, ok := parseColour(text); ok {
			t.Errorf("%q 不该被当成颜色", text)
		}
	}
}

func TestNumbersAreLimited(t *testing.T) {
	if got := clamp(5, 1, 9); got != 5 {
		t.Errorf("范围之内的数是 %d，不是 5", got)
	}
	if got := clamp(0, 1, 9); got != 1 {
		t.Errorf("太小的数是 %d，不是 1", got)
	}
	if got := clamp(20, 1, 9); got != 9 {
		t.Errorf("太大的数是 %d，不是 9", got)
	}
	if got := clamp(300, 0, 100); got != 100 {
		t.Errorf("太大的数是 %d，不是 100", got)
	}
}

func TestTheNameOfTheMachine(t *testing.T) {
	backend := newBackend(t, newFakeService())

	if got := backend.Instance(); got != "AcerAgentService" {
		t.Fatalf("机器的名字是 %q，不是 AcerAgentService", got)
	}
}

func TestOpeningAServiceThatIsNotThere(t *testing.T) {
	_, err := New(Options{Address: "127.0.0.1:1", Timeout: time.Second})
	if err == nil {
		t.Fatal("连不上的服务不该算作连上")
	}
	if !errors.Is(err, agentservice.ErrUnavailable) {
		t.Fatalf("错误是 %v，不是 %v", err, agentservice.ErrUnavailable)
	}
}

func TestTheBrightnessOfTheKeyboard(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = answer(
		map[string]any{"brightness": 42}, agentservice.ResultOK)

	backend := newBackend(t, service)

	brightness, err := backend.LightingBrightness()
	if err != nil {
		t.Fatalf("读取亮度失败: %v", err)
	}
	if brightness != 42 {
		t.Fatalf("亮度是 %d，不是 42", brightness)
	}
}

func TestAnAnswerThatCannotBeParsed(t *testing.T) {
	service := newFakeService()
	service.monitor = agentservice.Answer{
		Data:   json.RawMessage(`{"CPU_TEMPERATURE": "很热"}`),
		Result: agentservice.ResultOK,
	}
	backend := newBackend(t, service)

	if _, err := backend.Sensors(); err == nil {
		t.Fatal("答案不是数字时该报错")
	}
}

func TestAnAnswerWithoutData(t *testing.T) {
	service := newFakeService()
	service.answers[agentservice.FunctionLighting] = agentservice.Answer{
		Request: agentservice.FunctionLighting,
		Result:  agentservice.ResultOK,
	}
	backend := newBackend(t, service)

	if _, err := backend.Lighting(); err == nil {
		t.Fatal("没有数据的应答不该算作读到了灯效")
	}
}

func TestAWriteThatDoesNotHappen(t *testing.T) {
	service := newFakeService()
	service.setErr = errors.New("服务不可达")
	backend := newBackend(t, service)

	if err := backend.SetPowerMode(hardware.PowerPerformance); err == nil {
		t.Fatal("写不进去时该报错")
	}
}

func TestAReadThatDoesNotHappen(t *testing.T) {
	service := newFakeService()
	service.getErr = errors.New("服务不可达")
	backend := newBackend(t, service)

	if _, err := backend.PowerMode(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.GPUMode(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.WinKeyLocked(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.LCDOverdrive(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.BootSound(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.Backlight(); err == nil {
		t.Fatal("读不出来时该报错")
	}
	if _, err := backend.Fans(); err == nil {
		t.Fatal("读不出来时该报错")
	}
}

func TestTheFanOfTheOtherGroup(t *testing.T) {

	service := newFakeService()
	service.answers[agentservice.FunctionFanControl] = answer(map[string]any{
		"mode": 2,
		"custom_fan_data": []any{
			map[string]any{"fan_custom_auto": 0, "fan_custom_speed": 30, "fan_name": "GPU"},
			map[string]any{"fan_custom_auto": 0, "fan_custom_speed": 40, "fan_name": "第三个"},
		},
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	state, err := backend.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if state.GPUMode != hardware.FanCustom || state.GPUPercent != 30 {
		t.Errorf("显卡风扇是 %v 占空比 %d，不是固定占空比 30", state.GPUMode, state.GPUPercent)
	}
	if state.CPUMode != hardware.FanAuto {
		t.Errorf("处理器风扇是 %v，不是自动", state.CPUMode)
	}
}
