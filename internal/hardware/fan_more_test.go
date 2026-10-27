package hardware

import (
	"errors"
	"testing"
)

// The fans are driven through the gaming class of the provider. The
// numbers that travel to the firmware are built here, so they are worth
// pinning down.

func TestSetFansWritesTheBehaviourOfBothFans(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{
		CPUMode:    FanCustom,
		GPUMode:    FanMax,
		CPUPercent: 60,
		GPUPercent: 30,
	})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingFanBehavior")
	if !ok {
		t.Fatal("the behaviour of the fans was not written")
	}
	// The word carries the behaviour of the CPU fan in the third byte and
	// the behaviour of the GPU fan in the fourth one. Each of them takes
	// two bits, the three values of the behaviour fit into them.
	cpu := (call.Input >> 16) & 0x3
	gpu := (call.Input >> 22) & 0x3
	if cpu != 3 {
		t.Errorf("the behaviour of the CPU fan is %d, want 3", cpu)
	}
	if gpu != 2 {
		t.Errorf("the behaviour of the GPU fan is %d, want 2", gpu)
	}
}

func TestSetFansWritesTheDutyCycleOfTheCPUFan(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanCustom, GPUMode: FanAuto, CPUPercent: 60})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingFanSpeed")
	if !ok {
		t.Fatal("the speed of the fan was not written")
	}
	// The duty cycle sits in the second byte, the identifier of the fan
	// in the lowest one.
	speed := (call.Input >> 8) & 0xFF
	if speed != 60 {
		t.Errorf("the duty cycle is %d, want 60", speed)
	}
	if call.Input&0xFF != FanIndexCPU {
		t.Errorf("the fan has the identifier %d, want %d", call.Input&0xFF, FanIndexCPU)
	}
}

func TestSetFansDrivesTheMaximumFanAtFullSpeed(t *testing.T) {
	// The maximum mode ignores the duty cycle, the fan runs at full
	// speed.
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanMax, GPUMode: FanMax, CPUPercent: 20})
	if err != nil {
		t.Fatal(err)
	}

	for _, call := range provider.callsOf("SetGamingFanSpeed") {
		speed := (call.Input >> 8) & 0xFF
		if speed != 100 {
			t.Errorf("the duty cycle is %d, want 100", speed)
		}
	}
}

func TestSetFansDrivesBothGPUFans(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanAuto, GPUMode: FanCustom, GPUPercent: 45})
	if err != nil {
		t.Fatal(err)
	}

	// A machine with two graphics fans carries a fan for each of them.
	calls := provider.callsOf("SetGamingFanSpeed")
	gpuCalls := 0
	for _, call := range calls {
		index := call.Input & 0xFF
		if index == FanIndexGPUPrimary || index == FanIndexGPUSecondary {
			gpuCalls++
			speed := (call.Input >> 8) & 0xFF
			if speed != 45 {
				t.Errorf("the duty cycle of the fan %d is %d, want 45", index, speed)
			}
		}
	}
	if gpuCalls != 2 {
		t.Errorf("the graphics fans were written %d times, want two", gpuCalls)
	}
}

func TestSetFansHandsTheDutyCycleBackInTheAutomaticMode(t *testing.T) {
	// In the automatic mode the last requested duty cycle keeps running
	// otherwise, so the controller has to be told to forget it.
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanAuto, GPUMode: FanAuto, CPUPercent: 70})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingFanSpeed")
	if !ok {
		t.Fatal("the speed of the fan was not written")
	}
	if call.Input != FanIndexCPU {
		t.Errorf("the word is %d, want the identifier of the CPU fan alone", call.Input)
	}
}

func TestSetFansSelectsTheNormalTable(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanAuto, GPUMode: FanAuto})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingFanTable")
	if !ok {
		t.Fatal("the table of the fans was not written")
	}
	if call.Input != FanTableNormal {
		t.Errorf("the table is %d, want %d", call.Input, FanTableNormal)
	}
}

func TestSetFansSelectsTheBoostTable(t *testing.T) {
	// The boost table is used by CoolBoost and by the maximum mode of
	// both fans.
	cases := []struct {
		name   string
		config FanConfig
	}{
		{"the boost of the machine", FanConfig{CPUMode: FanAuto, GPUMode: FanAuto, CoolBoost: true}},
		{"the maximum of both fans", FanConfig{CPUMode: FanMax, GPUMode: FanMax}},
	}
	for _, c := range cases {
		provider := newFakeProvider()
		backend := newFakeBackend(provider)

		if err := backend.SetFans(c.config); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		call, ok := provider.callOf("SetGamingFanTable")
		if !ok {
			t.Fatalf("%s: the table of the fans was not written", c.name)
		}
		if call.Input != FanTableBoost {
			t.Errorf("%s: the table is %d, want %d", c.name, call.Input, FanTableBoost)
		}
	}
}

func TestSetFansClampsTheDutyCycle(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanCustom, GPUMode: FanAuto, CPUPercent: 199})
	if err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetGamingFanSpeed")
	speed := (call.Input >> 8) & 0xFF
	if speed != 100 {
		t.Errorf("the duty cycle is %d, want 100", speed)
	}
}

func TestSetFansStopsAtTheFirstError(t *testing.T) {
	provider := newFakeProvider()
	provider.fail(errors.New("the provider refuses"))
	backend := newFakeBackend(provider)

	if err := backend.SetFans(FanConfig{CPUMode: FanAuto}); err == nil {
		t.Error("the call was accepted although the provider refuses")
	}
}

func TestFansReadsTheBehaviourOfTheCPUFan(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0300)
	provider.output("GetGamingFanSpeed", "gmOutput", 0x3C00)
	provider.output("GetGamingFanTable", "gmOutput", 0x0300)
	// The action class reports whether the boost table is selected.
	provider.output("GetFunction", "uiOutput", 0x0100)
	backend := newFakeBackend(provider)

	state, err := backend.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if state.CPUMode != FanCustom {
		t.Errorf("the CPU fan is in the mode %s, want the fixed duty cycle", state.CPUMode)
	}
}

func TestFansReadsTheDutyCycle(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0100)
	// The answer of the speed carries the duty cycle in the second byte.
	provider.output("GetGamingFanSpeed", "gmOutput", 0x3C00)
	provider.output("GetGamingFanTable", "gmOutput", 0x0100)
	provider.output("GetFunction", "uiOutput", 0x0100)
	backend := newFakeBackend(provider)

	state, err := backend.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if state.CPUPercent != 60 {
		t.Errorf("the CPU fan runs at %d percent, want 60", state.CPUPercent)
	}
}

func TestFanTable(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanTable", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	table, err := backend.FanTable()
	if err != nil {
		t.Fatal(err)
	}
	if table != FanTableBoost {
		t.Errorf("the table is %d, want %d", table, FanTableBoost)
	}
}

func TestFanTableOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.FanTable(); err == nil {
		t.Error("the call was accepted although the provider does not answer")
	}
}
