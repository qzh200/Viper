package widgets

import "testing"

func TestTheFanTurnsFasterWhenItRunsFaster(t *testing.T) {
	if got := fanTurns(0); got != 0 {
		t.Errorf("a stopped fan turns at %v, want nothing", got)
	}
	if got := fanTurns(-100); got != 0 {
		t.Errorf("a fan below zero turns at %v, want nothing", got)
	}

	slow := fanTurns(1000)
	fast := fanTurns(3000)
	faster := fanTurns(5000)
	if !(slow < fast && fast < faster) {
		t.Errorf("the speeds are %v, %v and %v, want them to grow with the rpm",
			slow, fast, faster)
	}

	if got := fanTurns(10000); got != fanMaxTurns {
		t.Errorf("a fan above the limit turns at %v, want %v", got, fanMaxTurns)
	}
}
