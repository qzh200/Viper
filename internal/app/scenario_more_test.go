package app

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// The scenes hold the lighting and the fans, so that every performance
// profile can have its own look.

func TestSaveScenarioKeepsTheScenes(t *testing.T) {
	application := newTestApp(t)

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.SaveScenario("work"); err != nil {
		t.Fatal(err)
	}

	scenarios := application.Scenarios()
	if len(scenarios) != 2 {
		t.Errorf("the scenes are %v, want two of them", scenarios)
	}
	if _, ok := scenarios["game"]; !ok {
		t.Errorf("the scenes are %v, want the scene game", scenarios)
	}
	if _, ok := scenarios["work"]; !ok {
		t.Errorf("the scenes are %v, want the scene work", scenarios)
	}
}

func TestSaveScenarioReplacesTheSceneOfTheSameName(t *testing.T) {
	application := newTestApp(t)

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Brightness = 42
	})
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}

	scenarios := application.Scenarios()
	if len(scenarios) != 1 {
		t.Fatalf("the scenes are %v, want one of them", scenarios)
	}
	if scenarios["game"].Lighting.Brightness != 42 {
		t.Errorf("the scene holds the brightness %d, want 42",
			scenarios["game"].Lighting.Brightness)
	}
}

func TestDeleteScenarioOfANameThatIsNotThere(t *testing.T) {
	// A scene that does not exist cannot be removed, but the call must
	// not fail either, the program asks for a removal of a scene that the
	// user has just deleted in another window.
	application := newTestApp(t)
	if err := application.DeleteScenario("missing"); err != nil {
		t.Errorf("the removal of a scene that is not there failed: %v", err)
	}
}

func TestDeleteScenarioKeepsTheOtherBindings(t *testing.T) {
	application := newTestApp(t)
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.SaveScenario("work"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("balanced", "game"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("turbo", "work"); err != nil {
		t.Fatal(err)
	}

	if err := application.DeleteScenario("game"); err != nil {
		t.Fatal(err)
	}

	bindings := application.Bindings()
	if _, ok := bindings["balanced"]; ok {
		t.Error("the binding to the removed scene is still there")
	}
	if bindings["turbo"] != "work" {
		t.Errorf("the bindings are %v, want the scene work for the profile turbo", bindings)
	}
}

func TestUnbindScenarioOfAProfileThatIsNotBound(t *testing.T) {
	application := newTestApp(t)
	if err := application.UnbindScenario("balanced"); err != nil {
		t.Errorf("the removal of a binding that is not there failed: %v", err)
	}
}

func TestBindScenarioTwice(t *testing.T) {
	// A profile carries one scene, a second binding replaces the first
	// one.
	application := newTestApp(t)
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.SaveScenario("work"); err != nil {
		t.Fatal(err)
	}

	if err := application.BindScenario("balanced", "game"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("balanced", "work"); err != nil {
		t.Fatal(err)
	}

	if got := application.Bindings()["balanced"]; got != "work" {
		t.Errorf("the binding is %q, want the scene work", got)
	}
}

func TestTheSettingsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store, err := config.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	application := NewWithStore(store)

	if got := application.SettingsPath(); got != path {
		t.Errorf("the settings are at %q, want %q", got, path)
	}
}

func TestTheLightingOfTheStoredSettings(t *testing.T) {
	// The firmware reports the brightness only, the effect and the
	// colours come from the settings of Viper.
	application := newTestApp(t)
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Effect = uint64(hardware.EffectWave)
		s.Lighting.Speed = 6
		s.Lighting.Direction = 1
		s.Lighting.Brightness = 65
		s.Lighting.PerZone = true
		s.Lighting.Zones[0] = config.LightingColor{R: 199, G: 0, B: 135}
	})

	lighting, err := application.Lighting()
	if err != nil {
		t.Fatal(err)
	}
	if lighting.Effect != hardware.EffectWave {
		t.Errorf("the effect is %d, want %d", lighting.Effect, hardware.EffectWave)
	}
	if lighting.Speed != 6 || lighting.Direction != 1 {
		t.Errorf("the lighting is %+v, want the speed 6 and the direction 1", lighting)
	}
	if lighting.Zones[0] != (hardware.RGBColor{R: 199, G: 0, B: 135}) {
		t.Errorf("the first zone is %+v, want the magenta", lighting.Zones[0])
	}
	if !lighting.PerZone {
		t.Error("the per zone flag was lost")
	}
	// The firmware does not report the colours and the effect, the
	// program has to keep its own record.
	if lighting.ColourReadable {
		t.Error("the firmware reports the colours")
	}
	if lighting.EffectReadable {
		t.Error("the firmware reports the effect")
	}
}

func TestToStoredLightingKeepsTheDirection(t *testing.T) {
	// The direction is a switch, both of its values are kept.
	for _, direction := range []uint64{0, 1} {
		stored := toStoredLighting(hardware.RGBConfig{Direction: direction})
		if stored.Direction != direction {
			t.Errorf("the stored direction is %d, want %d", stored.Direction, direction)
		}
	}
}

func TestToStoredLightingLimitsTheSpeed(t *testing.T) {
	// The speed runs from one to nine, a speed of zero is not a speed
	// that the keyboard can do.
	stored := toStoredLighting(hardware.RGBConfig{Speed: 0})
	if stored.Speed != 1 {
		t.Errorf("the stored speed is %d, want 1", stored.Speed)
	}
}

func TestToStoredLightingLimitsTheBrightness(t *testing.T) {
	stored := toStoredLighting(hardware.RGBConfig{Brightness: 0})
	if stored.Brightness != 0 {
		t.Errorf("the stored brightness is %d, want 0", stored.Brightness)
	}
}
