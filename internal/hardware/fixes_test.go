package hardware

import (
	"testing"

	"viper/internal/wmi"
)

// The tests in this file lock down the fixes of the audit: the readings that
// the firmware does not answer must not be turned into a value, the writes
// must be confirmed by reading them back, and the values that travel to the
// firmware must be the ones that are remembered.

func TestTheEcoProfileIsTheSixthEncoding(t *testing.T) {
	// 官方应用把节能档位暴露给用户，固件的编码是 6。
	if got := PowerEco.providerCode(); got != 6 {
		t.Errorf("the eco profile encodes to %d, want 6", got)
	}
	if got := decodePowerMode(6); got != PowerEco {
		t.Errorf("the value 6 decodes to %s, want the eco profile", got)
	}
	if got := PowerEco.String(); got != "eco" {
		t.Errorf("the eco profile renders as %q, want eco", got)
	}
	got, ok := ParsePowerMode("eco")
	if !ok || got != PowerEco {
		t.Errorf("the name eco decodes to %s (%v), want the eco profile", got, ok)
	}
}

func TestSetTheEcoProfile(t *testing.T) {
	provider := newFakeProvider()
	// The provider answers with the eco profile.
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0600)
	backend := newFakeBackend(provider)

	if err := backend.SetPowerMode(PowerEco); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingMiscSetting")
	if !ok {
		t.Fatal("the profile was not written")
	}
	if value := (call.Input >> 8) & 0xFF; value != 6 {
		t.Errorf("the profile was written as %d, want 6", value)
	}
}

func TestThePowerModeOfASettingThatIsNotThere(t *testing.T) {
	// The status byte of the answer is not zero, so the answer does not
	// describe a profile at all.
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0002)
	backend := newFakeBackend(provider)

	mode, err := backend.PowerMode()
	if err == nil {
		t.Error("the profile was accepted although the setting was not there")
	}
	if mode == PowerQuiet {
		t.Error("a failed read was reported as the quiet profile")
	}
}

func TestTheDiscreteGPUOfASettingThatIsNotThere(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0302)
	backend := newFakeBackend(provider)

	enabled, err := backend.DiscreteGPU()
	if err == nil {
		t.Error("the state of the graphics switch was accepted although the setting was not there")
	}
	if enabled {
		t.Error("a failed read was reported as the graphics switch being on")
	}
}

func TestTheOverclockingOfTheGraphicsCardIsNotTakenFromTheGraphicsSwitch(t *testing.T) {
	provider := newFakeProvider()
	// The machine has a discrete graphics card, that is not a capability.
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	gpu, _, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if gpu {
		t.Error("the overclocking of the graphics card is reported although the firmware does not know it")
	}
	// The state of the graphics switch must not be read at all.
	if calls := provider.countOf("GetGamingMiscSetting"); calls != 1 {
		t.Errorf("the miscellaneous settings were read %d times, want one", calls)
	}
}

func TestTheOverclockingOfTheProcessor(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingMiscSetting", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	gpu, cpu, err := backend.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}
	if !cpu {
		t.Error("the overclocking of the processor is not reported")
	}
	if gpu {
		t.Error("the overclocking of the graphics card is reported")
	}
}

func TestTheFansDoNotReportTheBehaviourOfTheGraphicsCard(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0300)
	provider.output("GetGamingFanSpeed", "gmOutput", 0x3C00)
	provider.output("GetGamingFanTable", "gmOutput", 0x0100)
	provider.output("GetFunction", "uiOutput", 0x0100)
	backend := newFakeBackend(provider)

	state, err := backend.Fans()
	if err != nil {
		t.Fatal(err)
	}
	if state.GPUModeReadable {
		t.Error("the behaviour of the graphics card is reported although the firmware does not answer it")
	}
}

func TestSetFansReportsABehaviourThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	// The firmware keeps the automatic mode.
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanCustom, CPUPercent: 60})
	if err == nil {
		t.Error("the change was accepted although the provider kept the old behaviour")
	}
}

func TestSetFansReportsADutyCycleThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	// The behaviour matches, the duty cycle does not.
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0300)
	provider.output("GetGamingFanSpeed", "gmOutput", 0x3C00)
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanCustom, CPUPercent: 40})
	if err == nil {
		t.Error("the change was accepted although the provider kept the old duty cycle")
	}
}

func TestSetFansReportsAFanTableThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	// The behaviour matches, the table does not.
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0100)
	provider.output("GetGamingFanTable", "gmOutput", 0x0300)
	backend := newFakeBackend(provider)

	err := backend.SetFans(FanConfig{CPUMode: FanAuto})
	if err == nil {
		t.Error("the change was accepted although the provider kept the old fan table")
	}
}

func TestSetFansAcceptsWhatTheFirmwareKeeps(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingFanBehavior", "gmOutput", 0x0300)
	provider.output("GetGamingFanSpeed", "gmOutput", 0x2800)
	provider.output("GetGamingFanTable", "gmOutput", 0x0100)
	backend := newFakeBackend(provider)

	if err := backend.SetFans(FanConfig{CPUMode: FanCustom, CPUPercent: 40}); err != nil {
		t.Fatal(err)
	}
}

func TestSetLightingLimitsTheValuesItWrites(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	err := backend.SetLighting(RGBConfig{Effect: EffectWave, Speed: 20, Direction: 5, Brightness: 150})
	if err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingLEDBehavior")
	if !ok {
		t.Fatal("the behaviour of the lighting was not written")
	}
	// The speed runs from one to nine, the direction is a switch.
	if speed := (call.Input >> 8) & 0xFF; speed != 9 {
		t.Errorf("the speed is %d, want 9", speed)
	}
	if direction := call.Input & 0xFF; direction != 1 {
		t.Errorf("the direction is %d, want 1", direction)
	}

	buffer, ok := provider.callOf("SetGamingKBBacklight")
	if !ok {
		t.Fatal("the buffer of the backlight was not written")
	}
	if buffer.Bytes[2] != 100 {
		t.Errorf("the brightness is %d, want 100", buffer.Bytes[2])
	}
}

func TestSetLightingKeepsTheSwitchOfTheBacklightWhenItCannotBeRead(t *testing.T) {
	// The provider does not answer the action class, so the state of the
	// automatic switch off is unknown and must not be written.
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if err := backend.SetLighting(RGBConfig{Brightness: 80}); err != nil {
		t.Fatal(err)
	}
	if _, ok := provider.callOf("SetFunction"); ok {
		t.Error("the action class was written although the state of the switch is unknown")
	}
}

func TestSetLightingKeepsTheSwitchOfTheBacklightThatIsOn(t *testing.T) {
	provider := newFakeProvider()
	// The action class reports the timeout in the sixth byte.
	provider.output("GetFunction", "uiOutput", uint64(backlightTimeoutSeconds)<<40)
	backend := newFakeBackend(provider)

	if err := backend.SetLighting(RGBConfig{Brightness: 80}); err != nil {
		t.Fatal(err)
	}
	call, ok := provider.callOf("SetFunction")
	if !ok {
		t.Fatal("the backlight was not written")
	}
	if timeout := (call.Input >> 40) & 0xFF; timeout != backlightTimeoutSeconds {
		t.Errorf("the timeout is %d, want %d", timeout, backlightTimeoutSeconds)
	}
}

func TestSetBacklightReportsALevelThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	// The firmware keeps the old level in the buffer.
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.Bytes(backlightBuffer(20)),
	})
	backend := newFakeBackend(provider)

	if err := backend.SetBacklight(42, false); err == nil {
		t.Error("the level was accepted although the provider kept the old one")
	}
}

func TestSetBacklightAcceptsTheLevelThatIsKept(t *testing.T) {
	provider := newFakeProvider()
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.Bytes(backlightBuffer(42)),
	})
	backend := newFakeBackend(provider)

	if err := backend.SetBacklight(42, false); err != nil {
		t.Fatal(err)
	}
}

func TestTheBacklightBufferIsRead(t *testing.T) {
	provider := newFakeProvider()
	// Some builds answer with the bytes as a hexadecimal text.
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.String("00006400006400000300000000000000"),
	})
	backend := newFakeBackend(provider)

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 16 {
		t.Fatalf("the buffer is %d bytes long, want 16", len(buffer))
	}
	if buffer[2] != 100 || buffer[5] != 100 {
		t.Errorf("the buffer carries %d and %d, want 100 and 100", buffer[2], buffer[5])
	}
	if buffer[8] != 3 {
		t.Errorf("the flags byte is %d, want 3", buffer[8])
	}
}

func TestTheBacklightBufferOfTheFirmware(t *testing.T) {
	// The buffer that the firmware reports on a Predator: the level sits in
	// byte two and in byte five, byte eight holds the flags.
	provider := newFakeProvider()
	provider.answer("GetGamingKBBacklight", map[string]wmi.Value{
		"gmOutput": wmi.String("000064000064000003000000000000"),
	})
	backend := newFakeBackend(provider)

	buffer, err := backend.BacklightBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if buffer[2] != 100 {
		t.Errorf("the second byte is %d, want 100", buffer[2])
	}
	if buffer[5] != 100 {
		t.Errorf("the fifth byte is %d, want 100", buffer[5])
	}
	if buffer[8] != 3 {
		t.Errorf("the flags byte is %d, want 3", buffer[8])
	}
	if buffer[9] != 0 {
		t.Errorf("the ninth byte is %d, want 0", buffer[9])
	}
}

func TestTheBacklightBufferOfAMachineThatDoesNotAnswer(t *testing.T) {
	provider := newFakeProvider()
	backend := newFakeBackend(provider)

	if _, err := backend.BacklightBuffer(); err == nil {
		t.Error("the buffer was accepted although the provider does not answer")
	}
}

func TestTheReadingsOfAMachineThatReportsAnError(t *testing.T) {
	// The status byte of the answer is not zero, so the answer does not
	// describe a reading at all.
	provider := newFakeProvider()
	provider.output("GetGamingSysInfo", "gmOutput", 0x0002)
	backend := newFakeBackend(provider)

	if _, err := backend.Sensors(); err == nil {
		t.Error("the readings were accepted although the provider reports an error")
	}
}

func TestAReadingOfZeroDegrees(t *testing.T) {
	// Zero is a reading, not an error.
	provider := newFakeProvider()
	provider.output("GetGamingSysInfo", "gmOutput", 0x0000)
	backend := newFakeBackend(provider)

	sensors, err := backend.Sensors()
	if err != nil {
		t.Fatal(err)
	}
	if sensors.CPUTempC != 0 {
		t.Errorf("the processor runs at %d degrees, want 0", sensors.CPUTempC)
	}
}

func TestTheSpeedOfAFanUsesSixteenBits(t *testing.T) {
	provider := newFakeProvider()
	provider.sequence("GetGamingSysInfo", []uint64{
		0x0000,   // the temperature of the processor
		0x03E800, // the speed of the fan of the processor
		0x0000,   // the temperature of the system
		0x0000,   // the speed of the fan of the graphics card
		0x0000,   // the temperature of the graphics card
	})
	backend := newFakeBackend(provider)

	sensors, err := backend.Sensors()
	if err != nil {
		t.Fatal(err)
	}
	if sensors.CPUFanRPM != 1000 {
		t.Errorf("the fan of the processor runs at %d revolutions, want 1000", sensors.CPUFanRPM)
	}
}

func TestDecodeBytesOfTheAnswers(t *testing.T) {
	buffer, err := decodeBytes(wmi.Bytes([]byte{0, 0, 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 3 || buffer[2] != 100 {
		t.Errorf("the buffer is %v, want three bytes with 100 at the end", buffer)
	}

	buffer, err = decodeBytes(wmi.String("00 00 64"))
	if err != nil {
		t.Fatal(err)
	}
	if len(buffer) != 3 || buffer[2] != 100 {
		t.Errorf("the buffer is %v, want three bytes with 100 at the end", buffer)
	}

	if _, err := decodeBytes(wmi.Uint64(3)); err == nil {
		t.Error("a number was taken for a byte array")
	}
	if _, err := decodeBytes(wmi.String("0x1")); err == nil {
		t.Error("a broken text was taken for a byte array")
	}
}

func TestClampUint(t *testing.T) {
	cases := []struct {
		value uint64
		low   uint64
		high  uint64
		want  uint64
	}{
		{0, 1, 9, 1},
		{5, 1, 9, 5},
		{20, 1, 9, 9},
		{3, 0, 1, 1},
		{150, 0, 100, 100},
	}
	for _, c := range cases {
		if got := clampUint(c.value, c.low, c.high); got != c.want {
			t.Errorf("%d was clamped to %d, want %d", c.value, got, c.want)
		}
	}
}
