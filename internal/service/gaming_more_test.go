package service

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The official application talks to the service, so the answers have to
// describe the machine the way the original service does.

func newProviderWith(t *testing.T) (*Provider, *fakeGaming) {
	t.Helper()
	machine := newFakeGaming()
	return NewProvider(machine, nil), machine
}

// withoutTheSlowCalls puts a number in place of the calls that need the
// namespace of Windows, so that the tests do not wait for them.
func withoutTheSlowCalls(t *testing.T) {
	t.Helper()
	original := maximumClock
	maximumClock = func() int { return 4200 }
	t.Cleanup(func() { maximumClock = original })
}

func TestTheOperatingModeOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.powerMode = hardware.PowerTurbo

	value, ok := provider.Setting("OPERATING_MODE")
	if !ok {
		t.Fatal("the profile is not known")
	}
	mode, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the profile is %T, want a set of values", value)
	}
	// The service counts the profiles from zero, the provider counts them
	// from one.
	if mode["mode"] != 5 {
		t.Errorf("the profile is %v, want 5", mode["mode"])
	}
}

func TestTheOperatingModeOfEveryProfile(t *testing.T) {
	cases := map[hardware.PowerMode]int{
		hardware.PowerQuiet:       0,
		hardware.PowerBalanced:    1,
		hardware.PowerPerformance: 4,
		hardware.PowerTurbo:       5,
	}
	for mode, want := range cases {
		provider, machine := newProviderWith(t)
		machine.powerMode = mode

		value, ok := provider.Setting("OPERATING_MODE")
		if !ok {
			t.Fatalf("the profile %s is not known", mode)
		}
		settings, _ := value.(map[string]any)
		if settings["mode"] != want {
			t.Errorf("the profile %s is %v, want %d", mode, settings["mode"], want)
		}
	}
}

func TestTheGPUModeOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.gpuMode = hardware.GPUModeDiscrete

	value, ok := provider.Setting("GPU_MODE")
	if !ok {
		t.Fatal("the graphics mode is not known")
	}
	mode, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the graphics mode is %T, want a set of values", value)
	}
	// The service counts the modes from zero, the provider counts them
	// from one.
	if mode["mode"] != 1 {
		t.Errorf("the graphics mode is %v, want 1", mode["mode"])
	}
}

func TestTheGPUModeOfTheHybridSwitch(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.gpuMode = hardware.GPUModeHybrid

	value, ok := provider.Setting("GPU_MODE")
	if !ok {
		t.Fatal("the graphics mode is not known")
	}
	mode, _ := value.(map[string]any)
	if mode["mode"] != 0 {
		t.Errorf("the graphics mode is %v, want 0", mode["mode"])
	}
}

func TestTheFansOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.fans = hardware.FanState{
		CPUMode:    hardware.FanCustom,
		GPUMode:    hardware.FanAuto,
		CPUPercent: 60,
		GPUPercent: 40,
	}

	value, ok := provider.Setting("FAN_CONTROL")
	if !ok {
		t.Fatal("the fans are not known")
	}
	fans, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("the fans are %T, want a set of values", value)
	}
	if fans["mode"] != 0 {
		t.Errorf("the mode is %v, want 0", fans["mode"])
	}
	data, ok := fans["custom_fan_data"].([]any)
	if !ok {
		t.Fatalf("the fans are %T, want a list", fans["custom_fan_data"])
	}
	if len(data) != 2 {
		t.Fatalf("the fans are %d, want two of them", len(data))
	}

	cpu, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("the CPU fan is %T, want a set of values", data[0])
	}
	if cpu["fan_name"] != "CPU" {
		t.Errorf("the first fan is called %v, want CPU", cpu["fan_name"])
	}
	// The CPU fan runs at a fixed duty cycle, the graphics fan follows
	// the curve of the controller.
	if cpu["fan_custom_auto"] != 0 {
		t.Errorf("the CPU fan reports %v for the automatic switch, want 0", cpu["fan_custom_auto"])
	}
	if cpu["fan_custom_speed"] != 60 {
		t.Errorf("the CPU fan runs at %v percent, want 60", cpu["fan_custom_speed"])
	}

	gpu, ok := data[1].(map[string]any)
	if !ok {
		t.Fatalf("the graphics fan is %T, want a set of values", data[1])
	}
	if gpu["fan_custom_auto"] != 1 {
		t.Errorf("the graphics fan reports %v for the automatic switch, want 1", gpu["fan_custom_auto"])
	}
}

func TestTheBacklightTimeoutOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.backlight = hardware.BacklightState{Timeout: true}

	value, ok := provider.Setting("BACKLIGHT_30S")
	if !ok {
		t.Fatal("the backlight is not known")
	}
	backlight, _ := value.(map[string]any)
	if backlight["status"] != 1 {
		t.Errorf("the status is %v, want 1", backlight["status"])
	}
}

func TestTheWindowsKeyOfTheMachine(t *testing.T) {
	// The service tells whether the Windows key may be used, so a locked
	// key is reported as a status of zero.
	provider, machine := newProviderWith(t)
	machine.winKeyLocked = true

	value, ok := provider.Setting("WIN_KEY")
	if !ok {
		t.Fatal("the Windows key is not known")
	}
	key, _ := value.(map[string]any)
	if key["status"] != 0 {
		t.Errorf("the status is %v, want 0", key["status"])
	}
}

func TestTheWindowsKeyThatIsNotLocked(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.winKeyLocked = false

	value, ok := provider.Setting("WIN_KEY")
	if !ok {
		t.Fatal("the Windows key is not known")
	}
	key, _ := value.(map[string]any)
	if key["status"] != 1 {
		t.Errorf("the status is %v, want 1", key["status"])
	}
}

func TestTheOverdriveOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.lcdOverdrive = true

	value, ok := provider.Setting("LCD_OVERDRIVE")
	if !ok {
		t.Fatal("the overdrive is not known")
	}
	overdrive, _ := value.(map[string]any)
	if overdrive["status"] != 1 {
		t.Errorf("the status is %v, want 1", overdrive["status"])
	}
}

func TestTheBootSoundOfTheMachine(t *testing.T) {
	provider, machine := newProviderWith(t)
	machine.bootSound = true

	value, ok := provider.Setting("BOOT_SOUND")
	if !ok {
		t.Fatal("the sound is not known")
	}
	sound, _ := value.(map[string]any)
	if sound["status"] != 1 {
		t.Errorf("the status is %v, want 1", sound["status"])
	}
}

func TestWriteTheOperatingMode(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("OPERATING_MODE", map[string]any{"mode": float64(5)})
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenPower) != 1 || machine.writtenPower[0] != hardware.PowerTurbo {
		t.Errorf("the profile was written %v, want the turbo one", machine.writtenPower)
	}
}

func TestWriteTheGPUMode(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("GPU_MODE", map[string]any{"mode": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	// The service counts the modes from zero, the provider counts them
	// from one.
	if len(machine.writtenGPUMode) != 1 || machine.writtenGPUMode[0] != hardware.GPUModeDiscrete {
		t.Errorf("the graphics switch was written %v, want the discrete one", machine.writtenGPUMode)
	}
}

func TestWriteTheWindowsKey(t *testing.T) {
	// A status of one means that the key works, so it is unlocked.
	provider, machine := newProviderWith(t)

	err := provider.Apply("WIN_KEY", map[string]any{"status": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenWinKey) != 1 || machine.writtenWinKey[0] {
		t.Errorf("the lock of the Windows key was written %v, want it unlocked", machine.writtenWinKey)
	}
}

func TestWriteTheOverdrive(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("LCD_OVERDRIVE", map[string]any{"status": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenOverdrive) != 1 || !machine.writtenOverdrive[0] {
		t.Errorf("the overdrive was written %v, want it switched on", machine.writtenOverdrive)
	}
}

func TestWriteTheBootSound(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("BOOT_SOUND", map[string]any{"status": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenBootSound) != 1 || !machine.writtenBootSound[0] {
		t.Errorf("the sound was written %v, want it switched on", machine.writtenBootSound)
	}
}

func TestWriteTheBacklightTimeout(t *testing.T) {
	// The level of the backlight is read back, the change keeps it.
	provider, machine := newProviderWith(t)
	machine.brightness = 42

	err := provider.Apply("BACKLIGHT_30S", map[string]any{"status": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenBacklight) != 1 || machine.writtenBacklight[0] != 42 {
		t.Errorf("the backlight was written %v, want the brightness 42", machine.writtenBacklight)
	}
	if len(machine.writtenTimeout) != 1 || !machine.writtenTimeout[0] {
		t.Errorf("the timeout was written %v, want it switched on", machine.writtenTimeout)
	}
}

func TestWriteTheLightingOfTheZones(t *testing.T) {
	provider, machine := newProviderWith(t)

	err := provider.Apply("LIGHTING", map[string]any{
		"effect":     "WAVE",
		"speed":      float64(5),
		"direction":  float64(1),
		"brightness": float64(80),
		"LEDs": []any{
			map[string]any{"LED_id": float64(0), "color": "#C70087"},
			map[string]any{"LED_id": float64(1), "color": "#00AEC7"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(machine.written) != 1 {
		t.Fatalf("the lighting was written %d times, want once", len(machine.written))
	}
	written := machine.written[0]
	if written.Effect != hardware.EffectWave {
		t.Errorf("the effect is %d, want %d", written.Effect, hardware.EffectWave)
	}
	if written.Speed != 5 || written.Direction != 1 || written.Brightness != 80 {
		t.Errorf("the lighting is %+v, want the values of the request", written)
	}
	if !written.PerZone {
		t.Error("the zones do not carry their own colours")
	}
	if written.Zones[0] != (hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}) {
		t.Errorf("the first zone is %+v, want the magenta", written.Zones[0])
	}
	if written.Zones[1] != (hardware.RGBColor{R: 0, G: 0xAE, B: 0xC7}) {
		t.Errorf("the second zone is %+v, want the cyan", written.Zones[1])
	}
}

func TestWriteTheLightingOfOneColour(t *testing.T) {
	// A single colour goes to every zone, the program reports it as a
	// global setup.
	provider, machine := newProviderWith(t)

	err := provider.Apply("LIGHTING", map[string]any{
		"color": "#C70087",
	})
	if err != nil {
		t.Fatal(err)
	}

	written := machine.written[0]
	if written.PerZone {
		t.Error("the zones carry their own colours although one colour was sent")
	}
	for i, colour := range written.Zones {
		if colour != (hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}) {
			t.Errorf("the zone %d is %+v, want the magenta", i, colour)
		}
	}
}

func TestWriteTheLightingIgnoresTheBrokenZones(t *testing.T) {
	// A zone with a number that is not there or with a colour that
	// cannot be read is skipped, the rest of the lighting is written.
	provider, machine := newProviderWith(t)

	err := provider.Apply("LIGHTING", map[string]any{
		"LEDs": []any{
			map[string]any{"LED_id": float64(0), "color": "#C70087"},
			map[string]any{"LED_id": float64(9), "color": "#00AEC7"},
			map[string]any{"LED_id": float64(1), "color": "nonsense"},
			"not a zone",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	written := machine.written[0]
	if written.Zones[0] != (hardware.RGBColor{R: 0xC7, G: 0, B: 0x87}) {
		t.Errorf("the first zone is %+v, want the magenta", written.Zones[0])
	}
	if written.Zones[1] != (hardware.RGBColor{}) {
		t.Errorf("the second zone is %+v, want the default", written.Zones[1])
	}
}

func TestWriteTheLightingRemembersIt(t *testing.T) {
	// The colours cannot be read back from the firmware, so the service
	// has to remember them.
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	machine := newFakeGaming()
	provider := NewProvider(machine, store)

	err = provider.Apply("LIGHTING", map[string]any{
		"effect": "METEOR",
		"speed":  float64(7),
		"LEDs": []any{
			map[string]any{"LED_id": float64(0), "color": "#C70087"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	stored := store.Settings().Lighting
	if stored.Effect != uint64(hardware.EffectMeteor) {
		t.Errorf("the stored effect is %d, want %d", stored.Effect, hardware.EffectMeteor)
	}
	if stored.Speed != 7 {
		t.Errorf("the stored speed is %d, want 7", stored.Speed)
	}
	if stored.Zones[0] != (config.LightingColor{R: 0xC7, G: 0, B: 0x87}) {
		t.Errorf("the stored first zone is %+v, want the magenta", stored.Zones[0])
	}
}

func TestWriteTheLightingOfAMachineWithoutTheGamingProvider(t *testing.T) {
	provider := NewProvider(nil, nil)

	err := provider.Apply("LIGHTING", map[string]any{"color": "#C70087"})
	if err == nil {
		t.Error("the lighting was written although the gaming provider is not there")
	}
}

func TestTheMonitorDataOfTheMachine(t *testing.T) {
	withoutTheSlowCalls(t)
	provider, machine := newProviderWith(t)
	machine.sensors = hardware.Sensors{
		CPUTempC:    50,
		CPUFanRPM:   1000,
		GPUTempC:    60,
		GPUFanRPM:   2000,
		SystemTempC: 45,
	}

	data := provider.MonitorData()
	if data["CPU_TEMPERATURE"] != 50 {
		t.Errorf("the processor runs at %v degrees, want 50", data["CPU_TEMPERATURE"])
	}
	if data["GPU1_TEMPERATURE"] != 60 {
		t.Errorf("the graphics card runs at %v degrees, want 60", data["GPU1_TEMPERATURE"])
	}
	if data["SYS1_TEMPERATURE"] != 45 {
		t.Errorf("the system runs at %v degrees, want 45", data["SYS1_TEMPERATURE"])
	}
	if data["CPU_FANSPEED"] != 1000 {
		t.Errorf("the fan of the processor runs at %v revolutions, want 1000", data["CPU_FANSPEED"])
	}
	// The memory of the machine comes from the kernel of Windows.
	if _, ok := data["RAM_TOTAL"]; !ok {
		t.Error("the answer does not carry the memory of the machine")
	}
}

func TestTheMonitorDataOfAMachineWithoutTheGamingProvider(t *testing.T) {
	withoutTheSlowCalls(t)
	provider := NewProvider(nil, nil)

	data := provider.MonitorData()
	if _, ok := data["CPU_TEMPERATURE"]; ok {
		t.Error("the answer carries the readings of a machine that is not there")
	}
	// The parts that do not need the gaming provider are still there.
	if _, ok := data["RAM_TOTAL"]; !ok {
		t.Error("the answer does not carry the memory of the machine")
	}
}
