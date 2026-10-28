package nvidia

import "testing"

func TestAvailable(t *testing.T) {

	if Available() {
		if _, err := Read(); err != nil {
			t.Errorf("the driver is there but the cards cannot be read: %v", err)
		}
	}
}

func TestDriverVersion(t *testing.T) {
	version, err := DriverVersion()
	if err != nil {
		if Available() {
			t.Errorf("the driver is there but the version cannot be read: %v", err)
		}
		return
	}
	if version == "" {
		t.Error("the version of the driver is empty")
	}
}

func TestCloseWithoutTheDriver(t *testing.T) {

	Close()
	Close()
}

func TestTheIdleState(t *testing.T) {

	cases := []struct {
		telemetry Telemetry
		want      bool
	}{
		{Telemetry{UsagePercent: 0}, true},
		{Telemetry{UsagePercent: 1}, false},
		{Telemetry{UsagePercent: 100}, false},
	}
	for _, c := range cases {
		if got := c.telemetry.IsIdle(); got != c.want {
			t.Errorf("the card with the load %d is idle: %v, want %v",
				c.telemetry.UsagePercent, got, c.want)
		}
	}
}

func TestThePerformanceStateOfACardThatIsNotThere(t *testing.T) {

	if (Telemetry{}).PState != 0 {
		t.Error("the empty telemetry carries a performance state")
	}
}
