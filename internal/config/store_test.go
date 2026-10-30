package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestOpenInTheUserProfile(t *testing.T) {

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(store.Path(), home) {
		t.Errorf("the settings are at %q, want them inside %q", store.Path(), home)
	}

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

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"powerMode":2}`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Settings().PowerMode != PowerPerformance {
		t.Errorf("the power mode is %d, want %d", store.Settings().PowerMode, PowerPerformance)
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
		s.PowerMode = PowerTurbo
		s.Fans.CPUMode = FanCustom
		s.Fans.CPUPercent = 60
		s.Fans.CoolBoost = true
		s.BacklightTimeout = false
		s.Scenarios["game"] = Scenario{
			Lighting: Lighting{Effect: 3, Brightness: 80},
			Fans:     Fans{CPUMode: FanCustom, CPUPercent: 60},
		}
		s.Bindings["turbo"] = "game"
	}); err != nil {
		t.Fatal(err)
	}

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
	if settings.PowerMode != PowerTurbo {
		t.Errorf("the power mode is %d, want %d", settings.PowerMode, PowerTurbo)
	}
	if settings.Fans.CPUMode != FanCustom || settings.Fans.CPUPercent != 60 || !settings.Fans.CoolBoost {
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
		s.PowerMode = PowerPerformance
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.Lighting.Brightness = 90
	}); err != nil {
		t.Fatal(err)
	}

	settings := store.Settings()
	if settings.PowerMode != PowerPerformance {
		t.Errorf("the power mode is %d, want %d", settings.PowerMode, PowerPerformance)
	}
	if settings.Lighting.Brightness != 90 {
		t.Errorf("the brightness is %d, want 90", settings.Lighting.Brightness)
	}
}

func TestSaveWritesReadableText(t *testing.T) {

	path := filepath.Join(t.TempDir(), "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.PowerMode = PowerPerformance
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"powerMode": 2`) {
		t.Errorf("the file does not carry the power mode:\n%s", text)
	}
	if !strings.Contains(text, "\n") {
		t.Errorf("the file is written on a single line:\n%s", text)
	}
}

func TestDefaultSettingsAreTheAutomaticMode(t *testing.T) {

	settings := Default()
	if settings.Fans.CPUMode != FanAuto || settings.Fans.GPUMode != FanAuto {
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
	if settings.Lighting.Brightness != 5 {
		t.Errorf("the lighting starts with the brightness %d, want 5", settings.Lighting.Brightness)
	}
	if settings.Lighting.PerZone {
		t.Error("the lighting starts with the zones on their own")
	}
}

func TestTheSettingsAreACopy(t *testing.T) {

	store, err := OpenAt(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.Scenarios["game"] = Scenario{Lighting: Lighting{Brightness: 80}}
		s.Bindings["turbo"] = "game"
	}); err != nil {
		t.Fatal(err)
	}

	settings := store.Settings()
	delete(settings.Scenarios, "game")
	delete(settings.Bindings, "turbo")
	settings.Scenarios["other"] = Scenario{}
	settings.Bindings["quiet"] = "other"

	again := store.Settings()
	if _, ok := again.Scenarios["game"]; !ok {
		t.Errorf("the scenes are %v, want the scene game", again.Scenarios)
	}
	if again.Bindings["turbo"] != "game" {
		t.Errorf("the bindings are %v, want the scene game for the profile turbo", again.Bindings)
	}
	if _, ok := again.Scenarios["other"]; ok {
		t.Errorf("the scenes are %v, want no other scene", again.Scenarios)
	}
	if _, ok := again.Bindings["quiet"]; ok {
		t.Errorf("the bindings are %v, want no binding for the profile quiet", again.Bindings)
	}
}

func TestTheChangesThatRunAtTheSameTime(t *testing.T) {

	path := filepath.Join(t.TempDir(), "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(number int) {
			defer wg.Done()
			name := string(rune('a' + number))
			if err := store.Update(func(s *Settings) {
				s.Scenarios[name] = Scenario{Lighting: Lighting{Brightness: uint64(number)}}
				s.Bindings[name] = name
			}); err != nil {
				t.Errorf("the scene %q was not stored: %v", name, err)
			}
		}(i)
	}
	wg.Wait()

	settings := store.Settings()
	if len(settings.Scenarios) != 16 {
		t.Errorf("the store holds %d scenes, want 16", len(settings.Scenarios))
	}
	if len(settings.Bindings) != 16 {
		t.Errorf("the store holds %d bindings, want 16", len(settings.Bindings))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var restored Settings
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("the file is broken: %v\n%s", err, data)
	}
	if len(restored.Scenarios) != 16 {
		t.Errorf("the file holds %d scenes, want 16", len(restored.Scenarios))
	}
}

func TestSaveWritesTheSettingsOnItsOwn(t *testing.T) {

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}

	settings := store.Settings()
	settings.PowerMode = 4
	store.settings = settings

	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	again, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Settings().PowerMode != 4 {
		t.Errorf("the power mode is %d, want 4", again.Settings().PowerMode)
	}
}

func TestSaveOfAMachineThatRefuses(t *testing.T) {

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	blocked, err := OpenAt(filepath.Join(blocker, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := blocked.Save(); err == nil {
		t.Error("the save was accepted although the directory cannot be created")
	}
}

func TestTheFileIsWrittenWithoutALeftover(t *testing.T) {

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *Settings) {
		s.Lighting.Brightness = 70
	}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "config.json" {
			t.Errorf("the directory holds %q, want only the settings", entry.Name())
		}
	}
}
