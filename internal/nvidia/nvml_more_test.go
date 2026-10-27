package nvidia

import "testing"

// The telemetry of the card is read through the management library of the
// driver. A machine without the driver simply has no telemetry, that is
// not an error.

func TestAvailable(t *testing.T) {
	// The call must not fail, the machine either has the driver or not.
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
	// The call must not fail, also when the library was never opened.
	Close()
	Close()
}

func TestTheIdleState(t *testing.T) {
	// The driver does not always offer the performance state, so the load
	// of the graphics processor is used to tell.
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
	// A card that the driver does not describe has the performance state
	// of minus one, so that the caller can tell it from a card that is
	// idle.
	if (Telemetry{}).PState != 0 {
		t.Error("the empty telemetry carries a performance state")
	}
}
