package app

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The parts of the program that do not need the gaming provider are read
// from the machine itself.

func TestThePowerStatusOfTheMachine(t *testing.T) {
	// The information comes from the kernel of Windows, so it works
	// without the rights that the hardware needs.
	application := NewWithStore(mustStore(t))

	status, err := application.PowerStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !status.OnAC && !status.OnBattery {
		t.Errorf("the machine reports %+v, want either the supply or the battery", status)
	}
}

func TestTheIdentityOfTheMachine(t *testing.T) {
	application := NewWithStore(mustStore(t))

	identity, err := application.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if identity.Manufacturer == "" {
		t.Error("the machine has no manufacturer")
	}
}

func TestTheStickyKeysOfTheMachine(t *testing.T) {
	// The sticky keys belong to the accessibility settings of the user.
	application := NewWithStore(mustStore(t))

	enabled, err := application.StickyKeys()
	if err != nil {
		t.Fatal(err)
	}
	// The state is a switch, the call answers with one of the two.
	_ = enabled
}

func TestTheGraphicsTelemetryOfTheMachine(t *testing.T) {
	// A machine without the driver of the vendor has no telemetry, that
	// is not an error.
	application := NewWithStore(mustStore(t))

	cards, err := application.GPUTelemetry()
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range cards {
		if card.UsagePercent < 0 || card.UsagePercent > 100 {
			t.Errorf("the card reports the load %d percent", card.UsagePercent)
		}
	}
}

func TestTheAvailableProfilesOfTheMachine(t *testing.T) {
	application := NewWithStore(mustStore(t))

	modes := application.AvailablePowerModes()
	if len(modes) == 0 {
		t.Fatal("the machine offers no profile")
	}
	// The quiet profile and the balanced one are offered everywhere.
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
	// The provider refuses the calls without the rights of an
	// administrator, so the program cannot read the fans.
	application := NewWithStore(mustStore(t))

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
	if _, err := application.CoolBoost(); err == nil {
		t.Error("the boost was read although the gaming provider is not there")
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
	application := NewWithStore(mustStore(t))

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
	// The program opens the settings of the user, the test points the
	// profile at a temporary directory.
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)

	application, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	if application.SettingsPath() == "" {
		t.Error("the program has no path for its settings")
	}
}

func TestTheSettingsOfTheProgramInTheProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)

	store, err := config.OpenAt(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	application := NewWithStore(store)

	if application.SettingsPath() != filepath.Join(home, "config.json") {
		t.Errorf("the settings are at %q, want the file of the test", application.SettingsPath())
	}
}
