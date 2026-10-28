package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"viper/internal/agentservice"
	"viper/internal/nvidia"
)

// fakeService is a stand for the service of the original software.
type fakeService struct {
	answers map[string]json.RawMessage
	fail    error
}

func newFakeService() *fakeService {
	return &fakeService{answers: map[string]json.RawMessage{}}
}

func (s *fakeService) Get(function string) (agentservice.Answer, error) {
	if s.fail != nil {
		return agentservice.Answer{}, s.fail
	}
	return agentservice.Answer{Data: s.answers[function], Result: "0"}, nil
}

func (s *fakeService) Set(function string, value any) (agentservice.Answer, error) {
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

// TestShowCards shows the cards as the management library reports them.
func TestShowCards(t *testing.T) {
	cards := []nvidia.Telemetry{{
		Name:                "NVIDIA GeForce RTX 4060 Laptop GPU",
		UsagePercent:        39,
		MemoryUsagePercent:  50,
		GraphicsClockMHz:    375,
		MaxGraphicsClockMHz: 3105,
		MemoryClockMHz:      6001,
		MemoryUsedBytes:     4 << 30,
		MemoryTotalBytes:    8 << 30,
		PState:              0,
	}}

	out := capture(t, func() { showCards(cards) })

	for _, want := range []string{
		"card 0: NVIDIA GeForce RTX 4060 Laptop GPU",
		"usage        : 39%",
		"graphics clock: 375 MHz (max 3105 MHz)",
		"memory       : 4294967296 of 8589934592 bytes",
	} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

// TestShowService shows the graphics lines of the monitor data.
func TestShowService(t *testing.T) {
	service := newFakeService()
	service.answers["monitor"] = json.RawMessage(`{"GPU1_FANSPEED":2729,"CPU_TEMPERATURE":89}`)

	out := capture(t, func() { showService(service) })
	if !contains(out, "GPU1_FANSPEED") {
		t.Errorf("输出里没有显卡那一项:\n%s", out)
	}
	if contains(out, "CPU_TEMPERATURE") {
		t.Errorf("输出里不该有处理器那一项:\n%s", out)
	}
}

// TestShowServiceOfAServiceThatIsNotThere covers the service that does not
// answer.
func TestShowServiceOfAServiceThatIsNotThere(t *testing.T) {
	service := newFakeService()
	service.fail = errNotThere{}

	out := capture(t, func() { showService(service) })
	if !contains(out, "did not answer") {
		t.Errorf("输出里没有说服务没回答:\n%s", out)
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
