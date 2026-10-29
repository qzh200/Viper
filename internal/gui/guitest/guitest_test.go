package guitest_test

import (
	"testing"

	"viper/internal/domain"
	"viper/internal/gui/guitest"
)

func TestTheFakeMachineAnswers(t *testing.T) {
	machine := &guitest.Hardware{
		MonitorData:  domain.MonitorData{CPUTemperatureC: 44},
		Power:        domain.PowerPerformance,
		InstanceName: "AcerAgentService",
		RouteName:    "原厂服务",
	}

	monitor, err := machine.Monitor()
	if err != nil {
		t.Fatal(err)
	}
	if monitor.CPUTemperatureC != 44 {
		t.Errorf("the temperature is %d, want 44", monitor.CPUTemperatureC)
	}

	mode, err := machine.PowerMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != domain.PowerPerformance {
		t.Errorf("the power mode is %q, want the performance mode", mode)
	}

	if machine.Route() != "原厂服务" {
		t.Errorf("the route is %q, want the service route", machine.Route())
	}

	instance, err := machine.Instance()
	if err != nil {
		t.Fatal(err)
	}
	if instance != "AcerAgentService" {
		t.Errorf("the instance is %q, want AcerAgentService", instance)
	}
}

func TestTheFakeMachineRemembersTheWrites(t *testing.T) {
	machine := &guitest.Hardware{}

	if err := machine.SetPowerMode(domain.PowerTurbo); err != nil {
		t.Fatal(err)
	}
	if len(machine.Writes) != 1 || machine.Writes[0] != domain.PowerTurbo {
		t.Errorf("the writes are %v, want the turbo mode once", machine.Writes)
	}
	if machine.Power != domain.PowerTurbo {
		t.Errorf("the machine is in %q, want the turbo mode", machine.Power)
	}
}

func TestTheFakeMachineFailsWhenItIsToldTo(t *testing.T) {
	machine := &guitest.Hardware{Err: guitest.ErrHardware, SetErr: guitest.ErrHardware}

	if _, err := machine.Monitor(); err == nil {
		t.Error("the machine answered although it was told not to")
	}
	if _, err := machine.PowerMode(); err == nil {
		t.Error("the machine answered although it was told not to")
	}
	if _, err := machine.Instance(); err == nil {
		t.Error("the machine answered although it was told not to")
	}
	if err := machine.SetPowerMode(domain.PowerTurbo); err == nil {
		t.Error("the machine took the write although it was told not to")
	}
}

func TestTheFakeStoreKeepsTheAppearance(t *testing.T) {
	store := &guitest.Store{Mode: "dark"}

	if store.Theme() != "dark" {
		t.Errorf("the mode is %q, want dark", store.Theme())
	}
	if err := store.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if store.Theme() != "light" {
		t.Errorf("the mode is %q, want light", store.Theme())
	}
	if len(store.Saved) != 1 {
		t.Errorf("the store was written %d times, want once", len(store.Saved))
	}
}

func TestTheFakeStoreFailsWhenItIsToldTo(t *testing.T) {
	store := &guitest.Store{Err: guitest.ErrHardware}
	if err := store.SetTheme("light"); err == nil {
		t.Error("the store took the write although it was told not to")
	}
}
