//go:build windows

package hardware

import "testing"

func TestPowerStatusString(t *testing.T) {
	cases := []struct {
		status PowerStatus
		want   string
	}{
		{PowerStatus{OnAC: true}, "on the power supply"},
		{PowerStatus{OnAC: true, Charging: true}, "on the power supply, charging"},
		{PowerStatus{OnBattery: true, BatteryPercent: 42}, "on the battery at 42 percent"},
		{PowerStatus{}, "unknown"},
	}
	for _, c := range cases {
		if got := c.status.String(); got != c.want {
			t.Errorf("%+v renders as %q, want %q", c.status, got, c.want)
		}
	}
}

func TestSystemPowerStatus(t *testing.T) {

	status, err := SystemPowerStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !status.OnAC && !status.OnBattery {
		t.Errorf("the power supply reports %+v, want either the supply or the battery", status)
	}
	if status.OnBattery && (status.BatteryPercent < 0 || status.BatteryPercent > 100) {
		t.Errorf("the battery reports %d percent, want a number between zero and one hundred", status.BatteryPercent)
	}
}
