package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The settings live in a single file below the user profile. A missing or
// a broken file must not stop the program, the defaults are used instead.

func TestOpenInTheUserProfile(t *testing.T) {
	// The settings live below the profile of the user. The test points
	// the profile at a temporary directory, so that it does not touch the
	// settings of the user.
	//
	// The name of the variable that holds the directory of the settings
	// differs between the systems: Windows reads APPDATA, the others read
	// XDG_CONFIG_HOME. All of them are set, so that the test stays in its
	// own directory everywhere.
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	// The settings have to be inside the temporary directory, otherwise
	// the test would read the settings of the user.
	if !strings.HasPrefix(store.Path(), home) {
		t.Errorf("the settings are at %q, want them inside %q", store.Path(), home)
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
	// The file is written by hand, so that a person can look at it and
	// change a value.
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
	// The program starts with the fans on the curve of the embedded
	// controller and with the backlight that switches off on its own.
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
	if settings.Lighting.Brightness != 50 {
		t.Errorf("the lighting starts with the brightness %d, want 50", settings.Lighting.Brightness)
	}
	if settings.Lighting.PerZone {
		t.Error("the lighting starts with the zones on their own")
	}
}

// The store is shared between the connections of the service of Viper, so
// it has to survive calls that run at the same time.

func TestTheSettingsAreACopy(t *testing.T) {
	// The caller must not be able to change the stored settings by
	// accident, the maps of the settings are not shared with the store.
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
	// The service of Viper shares one store between its connections, a
	// change that runs at the same time as another one must not lose a
	// scene and must not write a broken file.
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

func TestTheFileIsWrittenWithoutALeftover(t *testing.T) {
	// The settings are written beside the file and moved over it, so that
	// a reader never sees a half written file. The temporary file must
	// not stay behind.
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
