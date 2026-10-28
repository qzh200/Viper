package app

import (
	"path/filepath"
	"strings"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

func TestThePowerStatusOfTheMachine(t *testing.T) {

	application := newTestAppWithoutMachine(t)

	status, err := application.PowerStatus()
	if err != nil {
		t.Skipf("the power status cannot be read on this system: %v", err)
	}
	if !status.OnAC && !status.OnBattery {
		t.Errorf("the machine reports %+v, want either the supply or the battery", status)
	}
}

func TestTheIdentityOfTheMachine(t *testing.T) {
	application := newTestAppWithoutMachine(t)

	identity, err := application.Identity()
	if err != nil {
		t.Skipf("the identity cannot be read on this system: %v", err)
	}
	if identity.Manufacturer == "" {
		t.Error("the machine has no manufacturer")
	}
}

func TestTheStickyKeysOfTheMachine(t *testing.T) {

	application := newTestAppWithoutMachine(t)

	enabled, err := application.StickyKeys()
	if err != nil {
		t.Skipf("the sticky keys cannot be read on this system: %v", err)
	}

	_ = enabled
}

func TestTheGraphicsTelemetryOfTheMachine(t *testing.T) {

	application := newTestAppWithoutMachine(t)

	telemetry, err := application.Telemetry()
	if err != nil {
		t.Skipf("the telemetry of the card cannot be read on this system: %v", err)
	}
	for _, card := range telemetry.Cards {
		if card.UsagePercent < 0 || card.UsagePercent > 100 {
			t.Errorf("the card reports the load %d percent", card.UsagePercent)
		}
	}
}

func TestTheAvailableProfilesOfTheMachine(t *testing.T) {
	application := newTestAppWithoutMachine(t)

	modes := application.AvailablePowerModes()
	if len(modes) == 0 {
		t.Fatal("the machine offers no profile")
	}

	found := map[hardware.PowerMode]bool{}
	for _, mode := range modes {
		found[mode] = true
	}
	if !found[hardware.PowerQuiet] {
		t.Errorf("the profiles are %v, want the quiet one", modes)
	}
	if !found[hardware.PowerBalanced] {
		t.Errorf("the profiles are %v, want the balanced one", modes)
	}
}

func TestTheSettingsOfTheMachineWithoutTheGamingProvider(t *testing.T) {

	application := newTestAppWithoutMachine(t)

	if _, err := application.Fans(); err == nil {
		t.Error("the fans were read although the gaming provider is not there")
	}
	if _, err := application.PowerMode(); err == nil {
		t.Error("the profile was read although the gaming provider is not there")
	}
	if _, err := application.WinKeyLocked(); err == nil {
		t.Error("the Windows key was read although the gaming provider is not there")
	}
	if _, err := application.LCDOverdrive(); err == nil {
		t.Error("the overdrive was read although the gaming provider is not there")
	}
	if _, err := application.BootSound(); err == nil {
		t.Error("the sound was read although the gaming provider is not there")
	}
	if _, err := application.Fans(); err == nil {
		t.Error("the fans were read although the machine is not there")
	}
	if _, err := application.DiscreteGPU(); err == nil {
		t.Error("the graphics switch was read although the gaming provider is not there")
	}
	if _, err := application.GPUMode(); err == nil {
		t.Error("the graphics mode was read although the gaming provider is not there")
	}
	if _, err := application.Sensors(); err == nil {
		t.Error("the readings were taken although the gaming provider is not there")
	}
	if _, _, err := application.OverclockCapability(); err == nil {
		t.Error("the overclocking was read although the gaming provider is not there")
	}
}

func TestTheSettingsThatAreWrittenWithoutTheGamingProvider(t *testing.T) {
	application := newTestAppWithoutMachine(t)

	if err := application.SetPowerMode(hardware.PowerTurbo); err == nil {
		t.Error("the profile was written although the gaming provider is not there")
	}
	if err := application.SetGPUMode(hardware.GPUModeDiscrete); err == nil {
		t.Error("the graphics mode was written although the gaming provider is not there")
	}
	if err := application.SetWinKeyLocked(true); err == nil {
		t.Error("the lock of the Windows key was written although the gaming provider is not there")
	}
	if err := application.SetLCDOverdrive(true); err == nil {
		t.Error("the overdrive was written although the gaming provider is not there")
	}
	if err := application.SetBootSound(true); err == nil {
		t.Error("the sound was written although the gaming provider is not there")
	}
	if err := application.SetCoolBoost(true); err == nil {
		t.Error("the boost was written although the gaming provider is not there")
	}
	if err := application.SetLighting(hardware.RGBConfig{}); err == nil {
		t.Error("the lighting was written although the gaming provider is not there")
	}
	if err := application.SetFans(hardware.FanConfig{}); err == nil {
		t.Error("the fans were written although the gaming provider is not there")
	}
	if err := application.ResetLighting(); err == nil {
		t.Error("the lighting was reset although the gaming provider is not there")
	}
}

func TestTheNewProgram(t *testing.T) {

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	application, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	if !strings.HasPrefix(application.SettingsPath(), home) {
		t.Errorf("the settings are at %q, want them inside %q", application.SettingsPath(), home)
	}
}

func TestTheSettingsOfTheProgramInTheProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	store, err := config.OpenAt(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	application := NewWithStore(store)

	if application.SettingsPath() != filepath.Join(home, "config.json") {
		t.Errorf("the settings are at %q, want the file of the test", application.SettingsPath())
	}
}
