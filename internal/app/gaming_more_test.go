package app

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/domain"
)

func newTestAppWith(t *testing.T) (*App, *fakeGaming) {
	t.Helper()
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	machine := newFakeGaming()
	return NewWithBackend(store, machine), machine
}

func TestSetLightingRemembersTheLighting(t *testing.T) {
	application, machine := newTestAppWith(t)

	lighting := domain.RGBConfig{
		Effect:     domain.EffectWave,
		Speed:      5,
		Direction:  1,
		Brightness: 80,
		PerZone:    true,
	}
	lighting.Zones[0] = domain.RGBColor{R: 199, G: 0, B: 135}

	if err := application.SetLighting(lighting); err != nil {
		t.Fatal(err)
	}

	if len(machine.written) != 1 {
		t.Fatalf("the lighting was written %d times, want once", len(machine.written))
	}
	if machine.written[0].Effect != domain.EffectWave {
		t.Errorf("the written effect is %d, want %d", machine.written[0].Effect, domain.EffectWave)
	}

	stored := application.store.Settings().Lighting
	if stored.Effect != uint64(domain.EffectWave) {
		t.Errorf("the stored effect is %d, want %d", stored.Effect, domain.EffectWave)
	}
	if stored.Brightness != 80 {
		t.Errorf("the stored brightness is %d, want 80", stored.Brightness)
	}
	if stored.Zones[0] != (config.LightingColor{R: 199, G: 0, B: 135}) {
		t.Errorf("the stored first zone is %+v, want the magenta", stored.Zones[0])
	}
}

func TestSetLightingLimitsTheStoredValues(t *testing.T) {

	application, _ := newTestAppWith(t)

	if err := application.SetLighting(domain.RGBConfig{
		Speed:      99,
		Direction:  5,
		Brightness: 199,
	}); err != nil {
		t.Fatal(err)
	}

	stored := application.store.Settings().Lighting
	if stored.Speed != 9 {
		t.Errorf("the stored speed is %d, want 9", stored.Speed)
	}
	if stored.Direction != 1 {
		t.Errorf("the stored direction is %d, want 1", stored.Direction)
	}
	if stored.Brightness != 100 {
		t.Errorf("the stored brightness is %d, want 100", stored.Brightness)
	}
}

func TestResetLightingWritesWhiteAndRemembersIt(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.ResetLighting(); err != nil {
		t.Fatal(err)
	}

	if len(machine.written) != 1 {
		t.Fatalf("the lighting was written %d times, want once", len(machine.written))
	}
	written := machine.written[0]
	if written.Effect != domain.EffectStatic {
		t.Errorf("the written effect is %d, want the static one", written.Effect)
	}
	if written.Brightness != 100 {
		t.Errorf("the written brightness is %d, want 100", written.Brightness)
	}
	if !written.PerZone {
		t.Error("the written lighting does not carry the zones on their own")
	}
	for i, colour := range written.Zones {
		if colour != (domain.RGBColor{R: 255, G: 255, B: 255}) {
			t.Errorf("the written zone %d is %+v, want white", i, colour)
		}
	}

	stored := application.store.Settings().Lighting
	if stored.Brightness != 100 {
		t.Errorf("the stored brightness is %d, want 100", stored.Brightness)
	}
	if stored.Zones[0] != (config.LightingColor{R: 255, G: 255, B: 255}) {
		t.Errorf("the stored first zone is %+v, want white", stored.Zones[0])
	}
}

func TestSetFansRemembersTheFans(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetFans(domain.FanConfig{
		CPUMode:    domain.FanCustom,
		GPUMode:    domain.FanMax,
		CPUPercent: 60,
		GPUPercent: 30,
		CoolBoost:  true,
	}); err != nil {
		t.Fatal(err)
	}

	if len(machine.writtenFans) != 1 {
		t.Fatalf("the fans were written %d times, want once", len(machine.writtenFans))
	}

	stored := application.store.Settings().Fans
	if stored.CPUMode != uint64(domain.FanCustom) {
		t.Errorf("the stored mode of the CPU fan is %d, want %d", stored.CPUMode, domain.FanCustom)
	}
	if stored.CPUPercent != 60 || stored.GPUPercent != 30 {
		t.Errorf("the stored fans are %+v, want 60 and 30 percent", stored)
	}
	if !stored.CoolBoost {
		t.Error("the stored boost is off")
	}
}

func TestSetCoolBoostRemembersIt(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetCoolBoost(true); err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenCoolBoost) != 1 || !machine.writtenCoolBoost[0] {
		t.Errorf("the boost was written %v, want it switched on", machine.writtenCoolBoost)
	}
	if !application.store.Settings().Fans.CoolBoost {
		t.Error("the stored boost is off")
	}
}

func TestSetPowerModeRemembersTheProfile(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatal(err)
	}
	if len(machine.writtenPower) != 1 || machine.writtenPower[0] != domain.PowerTurbo {
		t.Errorf("the profile was written %v, want the turbo one", machine.writtenPower)
	}
	if application.store.Settings().PowerMode != uint64(domain.PowerTurbo) {
		t.Errorf("the stored profile is %d, want %d",
			application.store.Settings().PowerMode, domain.PowerTurbo)
	}
}

func TestSetPowerModeAppliesTheBoundScene(t *testing.T) {

	application, machine := newTestAppWith(t)

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("turbo", "game"); err != nil {
		t.Fatal(err)
	}

	if err := application.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatal(err)
	}

	if len(machine.written) == 0 {
		t.Error("the lighting of the scene was not applied")
	}
	if len(machine.writtenFans) == 0 {
		t.Error("the fans of the scene were not applied")
	}
}

func TestSetPowerModeWithoutABoundScene(t *testing.T) {
	application, _ := newTestAppWith(t)

	if err := application.SetPowerMode(domain.PowerPerformance); err != nil {
		t.Fatal(err)
	}
}

func TestSetPowerModeOfAProfileThatIsNotBound(t *testing.T) {

	application, _ := newTestAppWith(t)
	application.store.Update(func(s *config.Settings) {
		s.Bindings["turbo"] = "missing"
	})

	if err := application.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatal(err)
	}
}

func TestApplyScenarioWritesTheLightingAndTheFans(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApplyScenario("game"); err != nil {
		t.Fatal(err)
	}

	if len(machine.written) != 1 {
		t.Errorf("the lighting was written %d times, want once", len(machine.written))
	}
	if len(machine.writtenFans) != 1 {
		t.Errorf("the fans were written %d times, want once", len(machine.writtenFans))
	}
}

func TestSaveScenarioTakesTheStateOfTheMachine(t *testing.T) {

	application, machine := newTestAppWith(t)
	machine.fans = domain.FanState{
		CPUMode:    domain.FanCustom,
		GPUMode:    domain.FanMax,
		CPUPercent: 70,
		GPUPercent: 40,
		CoolBoost:  true,
	}
	machine.brightness = 80

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}

	scenario := application.Scenarios()["game"]
	if scenario.Fans.CPUMode != uint64(domain.FanCustom) {
		t.Errorf("the scene holds the mode %d, want %d", scenario.Fans.CPUMode, domain.FanCustom)
	}
	if scenario.Fans.CPUPercent != 70 {
		t.Errorf("the scene holds the duty cycle %d, want 70", scenario.Fans.CPUPercent)
	}
	if scenario.Lighting.Brightness != 80 {
		t.Errorf("the scene holds the brightness %d, want 80", scenario.Lighting.Brightness)
	}
}

func TestSaveScenarioKeepsTheBrightnessOfTheSettings(t *testing.T) {

	application, machine := newTestAppWith(t)
	machine.brightness = 0
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Brightness = 65
	})

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}

	if got := application.Scenarios()["game"].Lighting.Brightness; got != 65 {
		t.Errorf("the scene holds the brightness %d, want 65", got)
	}
}

func TestSetBacklightTimeoutKeepsTheBrightness(t *testing.T) {
	application, machine := newTestAppWith(t)
	machine.brightness = 42

	if err := application.SetBacklightTimeout(true); err != nil {
		t.Fatal(err)
	}

	if len(machine.writtenBacklight) != 1 || machine.writtenBacklight[0] != 42 {
		t.Errorf("the backlight was written %v, want the brightness 42", machine.writtenBacklight)
	}
	if len(machine.writtenTimeout) != 1 || !machine.writtenTimeout[0] {
		t.Errorf("the timeout was written %v, want it switched on", machine.writtenTimeout)
	}
	if !application.store.Settings().BacklightTimeout {
		t.Error("the stored timeout is off")
	}
}

func TestSetBacklightTimeoutOfAMachineWithoutTheKeyboard(t *testing.T) {

	application := newTestAppWithoutMachine(t)
	if err := application.SetBacklightTimeout(true); err == nil {
		t.Error("the change was accepted although the brightness cannot be read")
	}
}

func TestTheSettingsOfTheMachine(t *testing.T) {
	application, machine := newTestAppWith(t)
	machine.bootSound = true
	machine.winKeyLocked = true
	machine.lcdOverdrive = true
	machine.discreteGPU = true

	machine.overclockGPU = false
	machine.overclockCPU = true

	machine.fans = domain.FanState{CPUMode: domain.FanMax, GPUMode: domain.FanMax}
	machine.backlight = domain.BacklightState{Timeout: true}

	if value, err := application.BootSound(); err != nil || !value {
		t.Errorf("the sound at the start of the machine is %v (%v), want it on", value, err)
	}
	if value, err := application.WinKeyLocked(); err != nil || !value {
		t.Errorf("the Windows key is locked: %v (%v), want it locked", value, err)
	}
	if value, err := application.LCDOverdrive(); err != nil || !value {
		t.Errorf("the overdrive is %v (%v), want it on", value, err)
	}
	fans, err := application.Fans()
	if err != nil {
		t.Fatalf("读取风扇失败: %v", err)
	}
	if !fans.CoolBoost {
		t.Error("the boost is off, want it on")
	}
	if value, err := application.DiscreteGPU(); err != nil || !value {
		t.Errorf("the graphics switch is %v (%v), want it on", value, err)
	}
	if value, err := application.BacklightTimeout(); err != nil || !value {
		t.Errorf("the backlight timeout is %v (%v), want it on", value, err)
	}

	gpu, cpu, err := application.OverclockCapability()
	if err != nil {
		t.Fatal(err)
	}

	if gpu || !cpu {
		t.Errorf("the overclocking of the card is %v and of the processor is %v", gpu, cpu)
	}
}

func TestTheSettingsThatAreWritten(t *testing.T) {
	application, machine := newTestAppWith(t)

	if err := application.SetBootSound(true); err != nil {
		t.Fatal(err)
	}
	if err := application.SetWinKeyLocked(true); err != nil {
		t.Fatal(err)
	}
	if err := application.SetLCDOverdrive(true); err != nil {
		t.Fatal(err)
	}
	if err := application.SetGPUMode(domain.GPUModeDiscrete); err != nil {
		t.Fatal(err)
	}

	if len(machine.writtenBootSound) != 1 || !machine.writtenBootSound[0] {
		t.Errorf("the sound was written %v, want it switched on", machine.writtenBootSound)
	}
	if len(machine.writtenWinKey) != 1 || !machine.writtenWinKey[0] {
		t.Errorf("the lock of the Windows key was written %v, want it locked", machine.writtenWinKey)
	}
	if len(machine.writtenOverdrive) != 1 || !machine.writtenOverdrive[0] {
		t.Errorf("the overdrive was written %v, want it switched on", machine.writtenOverdrive)
	}
	if len(machine.writtenGPUMode) != 1 || machine.writtenGPUMode[0] != domain.GPUModeDiscrete {
		t.Errorf("the graphics switch was written %v, want the discrete one", machine.writtenGPUMode)
	}
}

func TestTheInstanceOfTheMachine(t *testing.T) {
	application, _ := newTestAppWith(t)

	instance, err := application.Instance()
	if err != nil {
		t.Fatal(err)
	}
	if instance != "gaming\\instance" {
		t.Errorf("the instance is %q, want the path of the gaming class", instance)
	}
}

func TestCloseReleasesTheMachine(t *testing.T) {
	application, machine := newTestAppWith(t)

	application.Close()
	if !machine.closed {
		t.Error("the machine was not released")
	}
}

func TestCloseWithoutAMachine(t *testing.T) {

	application := newTestAppWithoutMachine(t)
	application.Close()
}

func TestTheLightingComesFromTheMachine(t *testing.T) {

	application, machine := newTestAppWith(t)
	machine.brightness = 80
	machine.lighting = domain.RGBConfig{
		Effect: domain.EffectBreathing,
		Speed:  4,
	}
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Effect = uint64(domain.EffectWave)
		s.Lighting.Brightness = 20
	})

	lighting, err := application.Lighting()
	if err != nil {
		t.Fatal(err)
	}
	if lighting.Brightness != 80 {
		t.Errorf("the brightness is %d, want the 80 of the machine", lighting.Brightness)
	}
	if lighting.Effect != domain.EffectBreathing {
		t.Errorf("the effect is %d, want %d", lighting.Effect, domain.EffectBreathing)
	}
}

func mustStore(t *testing.T) *config.Store {
	t.Helper()
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
