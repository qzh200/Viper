package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The settings live in a single file below the user profile. A missing or
// a broken file must not stop the program, the defaults are used instead.

func TestOpenInTheUserProfile(t *testing.T) {
	// The settings live below the profile of the user. The test points
	// the profile at a temporary directory, so that it does not touch the
	// settings of the user.
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)

	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if store.Path() == "" {
		t.Error("the store has no path")
	}
	// A missing file is not an error, the defaults are used.
	if store.Settings().PowerMode != Default().PowerMode {
		t.Errorf("the power mode is %d, want the default %d",
			store.Settings().PowerMode, Default().PowerMode)
	}
}

func TestOpenAtAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Path() != path {
		t.Errorf("the store points at %q, want %q", store.Path(), path)
	}
	settings := store.Settings()
	if settings.PowerMode != Default().PowerMode {
		t.Errorf("the power mode is %d, want the default %d", settings.PowerMode, Default().PowerMode)
	}
	if settings.Scenarios == nil {
		t.Error("the scenes are nil, want an empty map")
	}
	if settings.Bindings == nil {
		t.Error("the bindings are nil, want an empty map")
	}
}

func TestOpenAtABrokenFile(t *testing.T) {
	// A broken file must not stop the program.
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Settings().PowerMode != Default().PowerMode {
		t.Errorf("the power mode is %d, want the default %d",
			store.Settings().PowerMode, Default().PowerMode)
	}
}

func TestOpenAtAnOlderFile(t *testing.T) {
	// An older file has no scenes and no bindings, the maps have to exist
	// anyway, otherwise the program cannot store anything.
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"powerMode":4}`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Settings().PowerMode != 4 {
		t.Errorf("the power mode is %d, want 4", store.Settings().PowerMode)
	}
	if store.Settings().Scenarios == nil {
		t.Error("the scenes are nil, want an empty map")
	}
	if store.Settings().Bindings == nil {
		t.Error("the bindings are nil, want an empty map")
	}
}

func TestSaveAndOpenAgain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Viper", "config.json")

	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.Lighting.Effect = 3
		s.Lighting.Speed = 5
		s.Lighting.Brightness = 80
		s.Lighting.PerZone = true
		s.Lighting.Zones[0] = LightingColor{R: 199, G: 0, B: 135}
		s.PowerMode = 5
		s.Fans.CPUMode = 3
		s.Fans.CPUPercent = 60
		s.Fans.CoolBoost = true
		s.BacklightTimeout = false
		s.Scenarios["game"] = Scenario{
			Lighting: Lighting{Effect: 3, Brightness: 80},
			Fans:     Fans{CPUMode: 3, CPUPercent: 60},
		}
		s.Bindings["turbo"] = "game"
	}); err != nil {
		t.Fatal(err)
	}

	// The directory of the file is created on the way.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the settings were not written: %v", err)
	}

	again, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := again.Settings()
	if settings.Lighting.Effect != 3 || settings.Lighting.Speed != 5 {
		t.Errorf("the lighting is %+v, want the effect 3 and the speed 5", settings.Lighting)
	}
	if settings.Lighting.Zones[0] != (LightingColor{R: 199, G: 0, B: 135}) {
		t.Errorf("the first zone is %+v, want the magenta", settings.Lighting.Zones[0])
	}
	if !settings.Lighting.PerZone {
		t.Error("the per zone flag was lost")
	}
	if settings.PowerMode != 5 {
		t.Errorf("the power mode is %d, want 5", settings.PowerMode)
	}
	if settings.Fans.CPUMode != 3 || settings.Fans.CPUPercent != 60 || !settings.Fans.CoolBoost {
		t.Errorf("the fans are %+v, want the custom mode at 60 percent with the boost", settings.Fans)
	}
	if settings.BacklightTimeout {
		t.Error("the backlight timeout was switched on")
	}
	if _, ok := settings.Scenarios["game"]; !ok {
		t.Errorf("the scenes are %v, want the scene game", settings.Scenarios)
	}
	if settings.Bindings["turbo"] != "game" {
		t.Errorf("the bindings are %v, want the scene game for the profile turbo", settings.Bindings)
	}
}

func TestUpdateKeepsTheOtherSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.PowerMode = 4
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.Lighting.Brightness = 90
	}); err != nil {
		t.Fatal(err)
	}

	settings := store.Settings()
	if settings.PowerMode != 4 {
		t.Errorf("the power mode is %d, want 4", settings.PowerMode)
	}
	if settings.Lighting.Brightness != 90 {
		t.Errorf("the brightness is %d, want 90", settings.Lighting.Brightness)
	}
}

func TestSaveWritesReadableText(t *testing.T) {
	// The file is written by hand, so that a person can look at it and
	// change a value.
	path := filepath.Join(t.TempDir(), "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.PowerMode = 4
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"powerMode": 4`) {
		t.Errorf("the file does not carry the power mode:\n%s", text)
	}
	if !strings.Contains(text, "\n") {
		t.Errorf("the file is written on a single line:\n%s", text)
	}
}

func TestDefaultSettingsAreTheAutomaticMode(t *testing.T) {
	// The program starts with the fans on the curve of the embedded
	// controller and with the backlight that switches off on its own.
	settings := Default()
	if settings.Fans.CPUMode != 0 || settings.Fans.GPUMode != 0 {
		t.Errorf("the fans start in the mode %d and %d, want the automatic mode",
			settings.Fans.CPUMode, settings.Fans.GPUMode)
	}
	if !settings.BacklightTimeout {
		t.Error("the backlight timeout is switched off by default")
	}
	if settings.Lighting.Effect != 0 {
		t.Errorf("the lighting starts with the effect %d, want the static pattern", settings.Lighting.Effect)
	}
	if settings.Lighting.Speed != 3 {
		t.Errorf("the lighting starts with the speed %d, want 3", settings.Lighting.Speed)
	}
	if settings.Lighting.Brightness != 50 {
		t.Errorf("the lighting starts with the brightness %d, want 50", settings.Lighting.Brightness)
	}
	if settings.Lighting.PerZone {
		t.Error("the lighting starts with the zones on their own")
	}
}
