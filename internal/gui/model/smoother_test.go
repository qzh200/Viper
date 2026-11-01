package model_test

import (
	"testing"

	"viper/internal/domain"
	"viper/internal/gui/model"
)

func TestSmootherInitializesDirectly(t *testing.T) {
	s := model.NewSmoother()
	if s.Initialized() {
		t.Fatal("expected smoother to be uninitialized initially")
	}

	raw := domain.MonitorData{
		CPUTemperatureC: 80,
		CPUUsagePercent: 65.5,
		CPUFanRPM:       2500,
	}
	s.SetTarget(raw)

	if !s.Initialized() {
		t.Fatal("expected smoother to be initialized")
	}

	cur, metrics := s.Current()
	if cur.CPUTemperatureC != 80 {
		t.Errorf("got CPU temp %d, want 80", cur.CPUTemperatureC)
	}
	if cur.CPUUsagePercent != 65.5 {
		t.Errorf("got CPU usage %f, want 65.5", cur.CPUUsagePercent)
	}
	if cur.CPUFanRPM != 2500 {
		t.Errorf("got CPU fan %d, want 2500", cur.CPUFanRPM)
	}
	if len(metrics) != 6 {
		t.Errorf("got %d metrics, want 6", len(metrics))
	}
	if metrics[0].Number != 80.0 {
		t.Errorf("got metric number %f, want 80.0", metrics[0].Number)
	}

	if s.Step(model.DefaultSmoothAlpha) {
		t.Error("expected Step to return false when already at target")
	}
}

func TestSmootherInterpolatesSmoothly(t *testing.T) {
	s := model.NewSmoother()
	s.SetTarget(domain.MonitorData{CPUTemperatureC: 50})

	s.SetTarget(domain.MonitorData{CPUTemperatureC: 100})

	changed := s.Step(0.2)
	if !changed {
		t.Fatal("expected Step to return true during transition")
	}

	cur, metrics := s.Current()
	if cur.CPUTemperatureC != 60 {
		t.Errorf("after 1 step got CPU temp %d, want 60", cur.CPUTemperatureC)
	}
	if metrics[0].Number < 59.9 || metrics[0].Number > 60.1 {
		t.Errorf("after 1 step got metric number %f, want ~60.0", metrics[0].Number)
	}

	steps := 0
	for s.Step(0.2) {
		steps++
		if steps > 100 {
			t.Fatal("smoother did not converge within 100 steps")
		}
	}

	curFinal, _ := s.Current()
	if curFinal.CPUTemperatureC != 100 {
		t.Errorf("final converged CPU temp %d, want 100", curFinal.CPUTemperatureC)
	}
}

func TestSmootherFiltersNegativeAndIdleValues(t *testing.T) {
	s := model.NewSmoother()
	raw := domain.MonitorData{
		CPUTemperatureC:      -5,
		CPUUsagePercent:      -1.0,
		GPUIdle:              true,
		GPUUsagePercent:      -1,
		GPUFrequencyMHz:      -100,
		WIFIDownloadKbps:     -50.0,
		EthernetDownloadKbps: -10.0,
	}
	s.SetTarget(raw)

	cur, metrics := s.Current()
	if cur.CPUUsagePercent < 0 {
		t.Errorf("CPU usage is %f, want >= 0", cur.CPUUsagePercent)
	}
	if cur.GPUUsagePercent < 0 {
		t.Errorf("GPU usage is %d, want >= 0", cur.GPUUsagePercent)
	}
	if cur.WIFIDownloadKbps < 0 {
		t.Errorf("WIFI download is %f, want >= 0", cur.WIFIDownloadKbps)
	}
	if cur.CPUTemperatureC < 0 {
		t.Errorf("CPU temp is %d, want >= 0", cur.CPUTemperatureC)
	}

	for _, m := range metrics {
		if m.Number < 0 {
			t.Errorf("metric %q number is %f, want >= 0", m.Label, m.Number)
		}
		if m.Percent < 0 {
			t.Errorf("metric %q percent is %f, want >= 0", m.Label, m.Percent)
		}
	}
}

func TestSmootherDampensSuddenDrop(t *testing.T) {
	s := model.NewSmoother()
	s.SetTarget(domain.MonitorData{CPUUsagePercent: 30.0})

	s.SetTarget(domain.MonitorData{CPUUsagePercent: 0.0})

	s.Step(model.DefaultSmoothAlpha)
	cur, _ := s.Current()
	if cur.CPUUsagePercent == 0 {
		t.Error("CPU usage dropped to 0 instantly, expected dampening")
	}
}
