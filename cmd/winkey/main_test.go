package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"viper/internal/agentservice"
)

// fakeService is a stand for the service of the original software.
type fakeService struct {
	answers map[string]json.RawMessage
	written []string
	fail    error
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
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	return agentservice.Answer{Data: s.answers["monitor"], Result: "0"}, nil
}

// capture runs the tool with the output captured.
func capture(t *testing.T, fn func()) string {
	t.Helper()

	var buffer bytes.Buffer
	old := stdout
	stdout = &buffer
	defer func() { stdout = old }()

	fn()
	return buffer.String()
}

// TestProbeWinKey writes both numbers and reads them back.
func TestProbeWinKey(t *testing.T) {
	service := newFakeService()
	service.answer(agentservice.FunctionWinKey, `{"status":1}`)

	old := wait
	wait = func(time.Duration) {}
	defer func() { wait = old }()

	out := capture(t, func() { probeWinKey(service) })

	for _, want := range []string{"原始状态: 1", "写入 status=0", "写入 status=1", "还原成原始状态 1"} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
	// 四个读回加上最后那一次读回。
	if len(service.written) != 5 {
		t.Errorf("写了 %d 次:\n%s", len(service.written), out)
	}
}

// TestProbeWinKeyOfAServiceThatIsNotThere covers the service that does not
// answer.
func TestProbeWinKeyOfAServiceThatIsNotThere(t *testing.T) {
	service := newFakeService()
	service.fail = errNotThere{}

	var err error
	capture(t, func() { err = probeWinKey(service) })
	if err == nil {
		t.Error("服务不可达的时候该报错")
	}
	if !contains(err.Error(), "读取失败") {
		t.Errorf("错误信息里没有说读不到: %v", err)
	}
}

// errNotThere is the error of a service that is not there.
type errNotThere struct{}

func (errNotThere) Error() string { return "服务不可达" }

// contains reports whether the text holds the name.
func contains(text, name string) bool {
	for i := 0; i+len(name) <= len(text); i++ {
		if text[i:i+len(name)] == name {
			return true
		}
	}
	return false
}
