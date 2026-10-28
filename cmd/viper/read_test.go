package main

import (
	"testing"

	"viper/internal/hardware"
	"viper/internal/quickaccess"
)

// TestReadTheSettingsOfTheQuickAccessService reads the settings that belong
// to the Quick Access service.
//
// 这些命令只读，所以每一条都要报出机器现在的样子。
func TestReadTheSettingsOfTheQuickAccessService(t *testing.T) {
	installFakes(t)

	cases := []struct {
		command []string
		want    string
	}{
		{[]string{"displaymode"}, "display mode: discrete"},
		{[]string{"darkmode"}, "dark mode: off"},
		{[]string{"transparent"}, "transparent effect: off"},
		{[]string{"taskbar"}, "auto hide taskbar: off"},
		{[]string{"screensaver"}, "screen saver: off"},
		{[]string{"screensaver"}, "timeout     : 600 seconds"},
		{[]string{"charge"}, "charging limit  : 30%"},
		{[]string{"charge"}, "charging over USB: on"},
	}

	for _, item := range cases {
		out, _, err := capture(t, item.command...)
		if err != nil {
			t.Fatalf("%v: %v", item.command, err)
		}
		if !containsText(out, item.want) {
			t.Errorf("%v 的输出里没有 %q:\n%s", item.command, item.want, out)
		}
	}
}

// TestReadTheSwitchesOfTheMachine reads the switches of the machine.
func TestReadTheSwitchesOfTheMachine(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "profile")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"LCD overdrive : off", "Windows key   : off", "Sticky keys   : off"} {
		if !containsText(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
}

// TestReadTheBootSound reads the sound at the start.
func TestReadTheBootSound(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "bootsound")
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(out, "boot sound: off") {
		t.Errorf("输出不对:\n%s", out)
	}
}

// TestReadTheBacklightTimeout reads the automatic switch off of the
// backlight.
func TestReadTheBacklightTimeout(t *testing.T) {
	installFakes(t)

	out, _, err := capture(t, "lighting")
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(out, "timeout     : on") {
		t.Errorf("输出不对:\n%s", out)
	}
}

// TestTheMachineThatCannotBeRead checks that a machine that is not there is
// reported instead of being replaced by something else.
//
// 读不到的时候要报错，不拿别的东西顶替。
func TestTheMachineThatCannotBeRead(t *testing.T) {
	machine, _ := installFakes(t)
	machine.lightingErr = errNotThere{}

	out, errOut, err := capture(t, "lighting")
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(out, "not readable") {
		t.Errorf("输出里没有说机器读不到:\n%s", out)
	}
	if !containsText(errOut, "机器不在那里") {
		t.Errorf("错误信息里没有说读不到:\n%s", errOut)
	}
}

// errNotThere is the error of a machine that is not there.
type errNotThere struct{}

func (errNotThere) Error() string { return "机器不在那里" }

// containsText reports whether the text holds the name.
func containsText(text, name string) bool {
	for i := 0; i+len(name) <= len(text); i++ {
		if text[i:i+len(name)] == name {
			return true
		}
	}
	return false
}

// The modes of the graphics chip are the ones the machine reports.
var _ = hardware.GPUModeDiscrete
var _ = quickaccess.DisplayModeDiscrete
