package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"viper/internal/agentservice"
	"viper/internal/probe"
)

type fakeService struct {
	answers map[string]json.RawMessage
	written []string
	fail    error

	monitorFails bool
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
	s.written = append(s.written, function)
	return agentservice.Answer{Result: "0"}, nil
}

func (s *fakeService) MonitorData() (agentservice.Answer, error) {
	if s.fail != nil || s.monitorFails {
		return agentservice.Answer{}, errNotThere{}
	}
	return agentservice.Answer{Data: s.answers["monitor"], Result: "0"}, nil
}

func capture(t *testing.T, fn func()) string {
	t.Helper()

	var buffer bytes.Buffer
	old := stdout
	stdout = &buffer
	defer func() { stdout = old }()

	fn()
	return buffer.String()
}

func TestProbeFans(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl,
		`{"mode":0,"custom_fan_data":[{"fan_name":"CPU","fan_custom_auto":1,"fan_custom_speed":50,"fan_index":0}]}`)
	service.answer("monitor", `{"CPU_FANSPEED":2719,"GPU1_FANSPEED":2729,"CPU_TEMPERATURE":89}`)

	old := wait
	wait = func(time.Duration) {}
	defer func() { wait = old }()

	out := capture(t, func() {
		probeFans(service, 2,
			probe.FanData{Name: "CPU", Auto: 0, Speed: 50, Index: 0},
			probe.FanData{Name: "GPU", Auto: 0, Speed: 40, Index: 0})
	})

	for _, want := range []string{"写入之前:", "写入成功", "写入后 2 秒:", "CPU 风扇 2719 RPM", "风扇: mode 0"} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
	if len(service.written) != 1 {
		t.Errorf("写了 %d 次:\n%s", len(service.written), out)
	}
}

func TestShowOfAServiceThatIsNotThere(t *testing.T) {
	service := newFakeService()
	service.fail = errNotThere{}

	out := capture(t, func() { show(service) })
	if !contains(out, "读取风扇失败") {
		t.Errorf("输出里没有说读不到:\n%s", out)
	}
}

func TestShowWithoutMonitorData(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionFanControl, `{"mode":0,"custom_fan_data":[]}`)
	service.monitorFails = true

	out := capture(t, func() { show(service) })
	if !contains(out, "读取监视数据失败") {
		t.Errorf("输出里没有说读不到:\n%s", out)
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
