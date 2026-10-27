package app

import (
	"path/filepath"
	"testing"

	"viper/internal/config"
	"viper/internal/hardware"
)

// newTestApp opens the program with settings in a temporary file.
func newTestApp(t *testing.T) *App {
	t.Helper()
	store, err := config.OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewWithStore(store)
}

func TestSaveScenario(t *testing.T) {
	application := newTestApp(t)

	// The machine is not there in a test, so the scene keeps the settings
	// that are stored.
	application.store.Update(func(s *config.Settings) {
		s.Lighting.Effect = uint64(hardware.EffectWave)
		s.Lighting.Brightness = 80
		s.Fans.CPUMode = uint64(hardware.FanCustom)
	})

	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}

	scenarios := application.Scenarios()
	scenario, ok := scenarios["game"]
	if !ok {
		t.Fatalf("the scene was not saved, the scenes are %v", scenarios)
	}
	if scenario.Lighting.Effect != uint64(hardware.EffectWave) {
		t.Errorf("the scene holds the effect %d, want %d", scenario.Lighting.Effect, hardware.EffectWave)
	}
	if scenario.Lighting.Brightness != 80 {
		t.Errorf("the scene holds the brightness %d, want 80", scenario.Lighting.Brightness)
	}
}

func TestSaveScenarioNeedsAName(t *testing.T) {
	application := newTestApp(t)
	if err := application.SaveScenario(""); err == nil {
		t.Error("a scene without a name was accepted")
	}
}

func TestBindScenario(t *testing.T) {
	application := newTestApp(t)
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}

	if err := application.BindScenario("balanced", "game"); err != nil {
		t.Fatal(err)
	}
	if application.Bindings()["balanced"] != "game" {
		t.Errorf("the binding is %v, want the scene game", application.Bindings())
	}

	// A scene that does not exist cannot be bound.
	if err := application.BindScenario("quiet", "missing"); err == nil {
		t.Error("a missing scene was bound")
	}
}

func TestUnbindScenario(t *testing.T) {
	application := newTestApp(t)
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("balanced", "game"); err != nil {
		t.Fatal(err)
	}

	if err := application.UnbindScenario("balanced"); err != nil {
		t.Fatal(err)
	}
	if _, ok := application.Bindings()["balanced"]; ok {
		t.Error("the binding is still there")
	}
}

func TestDeleteScenarioRemovesTheBindings(t *testing.T) {
	application := newTestApp(t)
	if err := application.SaveScenario("game"); err != nil {
		t.Fatal(err)
	}
	if err := application.BindScenario("balanced", "game"); err != nil {
		t.Fatal(err)
	}

	if err := application.DeleteScenario("game"); err != nil {
		t.Fatal(err)
	}
	if _, ok := application.Scenarios()["game"]; ok {
		t.Error("the scene is still there")
	}
	if _, ok := application.Bindings()["balanced"]; ok {
		t.Error("the binding to the removed scene is still there")
	}
}

func TestApplyAMissingScene(t *testing.T) {
	application := newTestApp(t)
	if err := application.ApplyScenario("missing"); err == nil {
		t.Error("a missing scene was applied")
	}
}
