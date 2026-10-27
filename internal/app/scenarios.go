// 场景：把灯效与风扇打包，绑到性能档位上。
package app

import (
	"fmt"

	"viper/internal/config"
	"viper/internal/hardware"
)

// Scenarios returns the saved scenes.
//
// The scenes are a copy, the caller cannot change the stored scenes by
// accident.
//
// 返回的是副本，调用方改不动里面的场景。
func (a *App) Scenarios() map[string]config.Scenario {
	return a.store.Settings().Scenarios
}

// Bindings returns the scenes that are bound to a performance profile.
//
// The bindings are a copy, the caller cannot change the stored bindings
// by accident.
//
// 返回的是副本，调用方改不动里面的绑定。
func (a *App) Bindings() map[string]string {
	return a.store.Settings().Bindings
}

// SaveScenario stores the current lighting and fans under a name.
//
// The scene holds what the machine really does: the lighting and the fan
// configuration are read from it.
//
// 场景存的是机器现在的样子。灯效与颜色从机器读（原厂服务读得回来），风扇也
// 从机器读。背光自动熄灭的时候机器报的是零，那不是用户选的亮度，所以零的时
// 候仍然用 Viper 自己记着的那一个。
func (a *App) SaveScenario(name string) error {
	if name == "" {
		return fmt.Errorf("app: 场景需要一个名字")
	}

	settings := a.store.Settings()
	scenario := config.Scenario{
		Lighting: settings.Lighting,
		Fans:     settings.Fans,
	}

	if lighting, err := a.Lighting(); err == nil {
		stored := toStoredLighting(lighting)
		if stored.Brightness == 0 {
			stored.Brightness = settings.Lighting.Brightness
		}
		scenario.Lighting = stored
	}

	if state, err := a.Fans(); err == nil {
		scenario.Fans = config.Fans{
			CPUMode:    storedFanMode(state.CPUMode),
			GPUMode:    storedFanMode(state.GPUMode),
			CPUPercent: state.CPUPercent,
			GPUPercent: state.GPUPercent,
			CoolBoost:  state.CoolBoost,
		}
	}

	if err := a.store.Update(func(s *config.Settings) {
		s.Scenarios[name] = scenario
	}); err != nil {
		return fmt.Errorf("app: 保存场景 %q 失败: %w", name, err)
	}
	return nil
}

// DeleteScenario removes a scene and the bindings that point to it.
func (a *App) DeleteScenario(name string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Scenarios, name)
		for profile, scene := range s.Bindings {
			if scene == name {
				delete(s.Bindings, profile)
			}
		}
	}); err != nil {
		return fmt.Errorf("app: 删除场景 %q 失败: %w", name, err)
	}
	return nil
}

// ApplyScenario applies a scene to the machine.
func (a *App) ApplyScenario(name string) error {
	scenario, ok := a.store.Settings().Scenarios[name]
	if !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", name)
	}
	return a.applyScenario(scenario)
}

// applyScenario writes a scene to the machine and remembers it.
func (a *App) applyScenario(scenario config.Scenario) error {
	lighting := hardware.RGBConfig{
		Effect:     hardware.LightingEffect(scenario.Lighting.Effect),
		Speed:      scenario.Lighting.Speed,
		Direction:  scenario.Lighting.Direction,
		Brightness: scenario.Lighting.Brightness,
		PerZone:    scenario.Lighting.PerZone,
	}
	for i, colour := range scenario.Lighting.Zones {
		lighting.Zones[i] = hardware.RGBColor{R: colour.R, G: colour.G, B: colour.B}
	}
	if err := a.SetLighting(lighting); err != nil {
		return err
	}

	return a.SetFans(hardware.FanConfig{
		CPUMode:    fanMode(scenario.Fans.CPUMode),
		GPUMode:    fanMode(scenario.Fans.GPUMode),
		CPUPercent: scenario.Fans.CPUPercent,
		GPUPercent: scenario.Fans.GPUPercent,
		CoolBoost:  scenario.Fans.CoolBoost,
	})
}

// BindScenario ties a performance profile to a scene.
func (a *App) BindScenario(profile, scene string) error {
	if _, ok := a.store.Settings().Scenarios[scene]; !ok {
		return fmt.Errorf("app: 没有叫 %q 的场景", scene)
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.Bindings[profile] = scene
	}); err != nil {
		return fmt.Errorf("app: 保存档位 %q 的场景绑定失败: %w", profile, err)
	}
	return nil
}

// UnbindScenario removes the scene of a performance profile.
func (a *App) UnbindScenario(profile string) error {
	if err := a.store.Update(func(s *config.Settings) {
		delete(s.Bindings, profile)
	}); err != nil {
		return fmt.Errorf("app: 删除档位 %q 的场景绑定失败: %w", profile, err)
	}
	return nil
}
