// Package config keeps the settings of Viper between runs.
//
// The settings live in a single JSON file below the user profile, so that
// the state of the keyboard lighting and of the fans can be reported even
// though the firmware does not report the lighting state back.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Lighting is the remembered keyboard lighting.
type Lighting struct {
	// Effect is the lighting pattern.
	Effect uint64 `json:"effect"`
	// Speed is the animation speed.
	Speed uint64 `json:"speed"`
	// Direction is the animation direction.
	Direction uint64 `json:"direction"`
	// Brightness is the overall brightness.
	Brightness uint64 `json:"brightness"`
	// Zones holds the colour of each zone.
	Zones [4]LightingColor `json:"zones"`
	// PerZone reports whether the zones carry individual colours.
	PerZone bool `json:"perZone"`
}

// LightingColor is the colour of one lighting zone.
type LightingColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Fans is the remembered fan configuration.
type Fans struct {
	// CPUMode is the behaviour of the CPU fan.
	CPUMode uint64 `json:"cpuMode"`
	// GPUMode is the behaviour of the GPU fan.
	GPUMode uint64 `json:"gpuMode"`
	// CPUPercent is the duty cycle of the CPU fan.
	CPUPercent int `json:"cpuPercent"`
	// GPUPercent is the duty cycle of the GPU fan.
	GPUPercent int `json:"gpuPercent"`
	// CoolBoost reports whether the boost table is selected.
	CoolBoost bool `json:"coolBoost"`
}

// Settings is the complete state of Viper.
type Settings struct {
	// Lighting is the keyboard lighting.
	Lighting Lighting `json:"lighting"`
	// Fans is the fan configuration.
	Fans Fans `json:"fans"`
	// PowerMode is the performance profile.
	PowerMode uint64 `json:"powerMode"`
	// BacklightTimeout reports whether the backlight switches off.
	BacklightTimeout bool `json:"backlightTimeout"`
}

// Store reads and writes the settings.
type Store struct {
	path     string
	settings Settings
}

// Default returns the settings Viper starts with.
func Default() Settings {
	return Settings{
		Lighting: Lighting{
			Effect:     0,
			Speed:      3,
			Direction:  0,
			Brightness: 50,
			PerZone:    false,
		},
		Fans: Fans{
			CPUMode:    1,
			GPUMode:    1,
			CPUPercent: 0,
			GPUPercent: 0,
		},
		PowerMode:        1,
		BacklightTimeout: true,
	}
}

// Open loads the settings from the user profile.
//
// A missing file is not an error, the defaults are used instead.
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "Viper", "config.json")

	store := &Store{path: path, settings: Default()}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &store.settings); err != nil {
		// A broken file must not stop the program, the defaults are used.
		store.settings = Default()
	}
	return store, nil
}

// Settings returns the current settings.
func (s *Store) Settings() Settings { return s.settings }

// Path returns the location of the settings file.
func (s *Store) Path() string { return s.path }

// Save writes the settings back.
func (s *Store) Save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", s.path, err)
	}
	return nil
}

// Update applies a change to the settings and stores them.
func (s *Store) Update(change func(*Settings)) error {
	change(&s.settings)
	return s.Save()
}
