package quickaccess

import (
	"testing"
)

// The client talks to the service of the original stack. The tests below
// drive a whole session against a stand for the service, so that the
// handshake, the commands and the answers are all checked.

func TestConnectToTheService(t *testing.T) {
	server := newTestServer(t)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
}

func TestConnectToAServiceThatIsNotThere(t *testing.T) {
	// The service of the vendor is not running on every machine.
	_, err := Connect(Options{Address: "127.0.0.1:1"})
	if err == nil {
		t.Error("the connection to a service that is not there was accepted")
	}
}

func TestConnectToAServiceThatRefusesTheHandshake(t *testing.T) {
	server := newTestServer(t)
	server.refuseHandshake = true

	_, err := Connect(Options{Address: server.address()})
	if err == nil {
		t.Error("the connection to a service that refuses the handshake was accepted")
	}
}

func TestConnectToAServiceThatAnswersTheAuthenticationWithAnotherQuestion(t *testing.T) {
	server := newTestServer(t)
	server.wrongQuestion = true

	_, err := Connect(Options{Address: server.address()})
	if err == nil {
		t.Error("the connection to a service with the wrong keys was accepted")
	}
}

func TestGetACommand(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBCharge, 30)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	result, err := client.Get(CommandUSBCharge)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.Value.(float64)
	if !ok || int(value) != 30 {
		t.Errorf("the value is %v, want 30", result.Value)
	}
}

func TestTheCommandOfARequest(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBCharge, 30)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Get(CommandUSBCharge); err != nil {
		t.Fatal(err)
	}
	if len(server.requests) == 0 {
		t.Fatal("the service received no request")
	}
	request := server.requests[0]
	if request["Command"] != CommandUSBCharge {
		t.Errorf("the request is about %v, want %s", request["Command"], CommandUSBCharge)
	}
	if request["Action"] != ActionGet {
		t.Errorf("the action is %v, want %s", request["Action"], ActionGet)
	}
}

func TestSetACommand(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBCharge, 30)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.Set(CommandUSBCharge, 30); err != nil {
		t.Fatal(err)
	}
	if len(server.requests) == 0 {
		t.Fatal("the service received no request")
	}
	request := server.requests[0]
	if request["Action"] != ActionSet {
		t.Errorf("the action is %v, want %s", request["Action"], ActionSet)
	}
	if request["Param1"] != float64(30) {
		t.Errorf("the value is %v, want 30", request["Param1"])
	}
}

func TestTheBlueLightShield(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandBluelightShield, 2)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	level, err := client.BlueLightShield()
	if err != nil {
		t.Fatal(err)
	}
	if level != BlueLightMedium {
		t.Errorf("the level is %d, want %d", level, BlueLightMedium)
	}
}

func TestTheChargingLimit(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBCharge, 20)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	limit, err := client.USBChargeLimit()
	if err != nil {
		t.Fatal(err)
	}
	if limit != 20 {
		t.Errorf("the limit is %d, want 20", limit)
	}
}

func TestTheChargingOverUSB(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBChargeSwitch, true)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	enabled, err := client.USBChargeEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the charging over USB is not allowed")
	}
}

func TestTheChargingOverUSBThatIsNotAllowed(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBChargeSwitch, false)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	enabled, err := client.USBChargeEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("the charging over USB is allowed")
	}
}

func TestTheChargingOverUSBOfAnAnswerThatIsNotASwitch(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandUSBChargeSwitch, 30)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.USBChargeEnabled(); err == nil {
		t.Error("an answer that is not a switch was taken for one")
	}
}

func TestTheDarkMode(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandDarkMode, true)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	enabled, err := client.DarkMode()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("the dark mode is off")
	}
}

func TestTheDisplayMode(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandDisplayMode, 2)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	mode, err := client.DisplayMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != DisplayModeDiscrete {
		t.Errorf("the display mode is %d, want %d", mode, DisplayModeDiscrete)
	}
}

func TestTheSettingsOfTheService(t *testing.T) {
	// The service is asked for every setting, so that the whole surface
	// of the client is covered.
	server := newTestServer(t)
	server.answer(CommandBluelightShield, 1)
	server.answer(CommandUSBCharge, 10)
	server.answer(CommandUSBChargeSwitch, true)
	server.answer(CommandDisplayMode, 3)
	server.answer(CommandDarkMode, false)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if level, err := client.BlueLightShield(); err != nil || level != BlueLightLow {
		t.Errorf("the level of the blue light shield is %d (%v), want %d", level, err, BlueLightLow)
	}
	if limit, err := client.USBChargeLimit(); err != nil || limit != 10 {
		t.Errorf("the charging limit is %d (%v), want 10", limit, err)
	}
	if enabled, err := client.USBChargeEnabled(); err != nil || !enabled {
		t.Errorf("the charging over USB is %v (%v), want it allowed", enabled, err)
	}
	if mode, err := client.DisplayMode(); err != nil || mode != DisplayModeAuto {
		t.Errorf("the display mode is %d (%v), want %d", mode, err, DisplayModeAuto)
	}
	if enabled, err := client.DarkMode(); err != nil || enabled {
		t.Errorf("the dark mode is %v (%v), want it off", enabled, err)
	}
}

func TestSetTheSettingsOfTheService(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandBluelightShield, 1)
	server.answer(CommandUSBCharge, 10)
	server.answer(CommandUSBChargeSwitch, true)
	server.answer(CommandDisplayMode, 3)
	server.answer(CommandDarkMode, false)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SetBlueLightShield(BlueLightLow); err != nil {
		t.Fatal(err)
	}
	if err := client.SetUSBChargeLimit(10); err != nil {
		t.Fatal(err)
	}
	if err := client.SetUSBChargeEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := client.SetDisplayMode(DisplayModeAuto); err != nil {
		t.Fatal(err)
	}
	if err := client.SetDarkMode(false); err != nil {
		t.Fatal(err)
	}

	if len(server.requests) != 5 {
		t.Errorf("the service received %d requests, want five", len(server.requests))
	}
}

func TestSetTheChargingLimitOfAValueThatIsNotThere(t *testing.T) {
	server := newTestServer(t)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// The limit is refused before the service is asked, so the request
	// never leaves the program.
	if err := client.SetUSBChargeLimit(15); err == nil {
		t.Error("a limit that the service does not offer was accepted")
	}
	if len(server.requests) != 0 {
		t.Errorf("the service received %d requests, want none", len(server.requests))
	}
}

func TestTheAnswerOfACommandThatTheServiceDoesNotKnow(t *testing.T) {
	server := newTestServer(t)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// The service answers with nothing, the caller has to notice that
	// there is no number.
	if _, err := client.USBChargeLimit(); err == nil {
		t.Error("an answer without a number was taken for one")
	}
}

func TestTheSessionOfAServiceThatClosesIt(t *testing.T) {
	server := newTestServer(t)
	server.closeAfterHandshake = true

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		// The service may close the session before the authentication is
		// answered, that is a refusal of the session.
		return
	}
	defer client.Close()

	if _, err := client.Get(CommandUSBCharge); err == nil {
		t.Error("the request to a service that closed the session was accepted")
	}
}
