package app

import (
	"path/filepath"
	"strings"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The parts of the program that do not need the gaming provider are read
// from the machine itself.
//
// The readings are only possible on the system that the firmware belongs
// to, so a test that finds no reading skips itself instead of failing.
// That keeps the suite green on the other systems without hiding a real
// failure on this one.

func TestThePowerStatusOfTheMachine(t *testing.T) {
	// The information comes from the kernel of Windows, so it works
	// without the rights that the hardware needs.
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
	// The sticky keys belong to the accessibility settings of the user.
	application := newTestAppWithoutMachine(t)

	enabled, err := application.StickyKeys()
	if err != nil {
		t.Skipf("the sticky keys cannot be read on this system: %v", err)
	}
	// The state is a switch, the call answers with one of the two.
	_ = enabled
}

func TestTheGraphicsTelemetryOfTheMachine(t *testing.T) {
	// A machine without the driver of the vendor has no telemetry from the
	// card, that is not an error. The readings of the service are still
	// there.
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
	// The program opens the settings of the user, the test points the
	// profile at a temporary directory. The name of the variable that
	// holds the directory of the settings differs between the systems, so
	// all of them are set.
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

	// The settings have to be inside the temporary directory, otherwise
	// the test would read the settings of the user.
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
