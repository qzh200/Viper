package hardware

import (
	"testing"
)

// The profile word carries several switches at once, they are told apart
// by the bit position that is set.

func TestWinKeyLocked(t *testing.T) {
	// The lock of the Windows key sits in the fourth byte.
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0x01000000)
	backend := newFakeBackend(provider)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("the Windows key is not locked")
	}
}

func TestWinKeyIsNotLocked(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	locked, err := backend.WinKeyLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("the Windows key is locked")
	}
}

func TestSetWinKeyLockedWritesTheSwitchAndTheLock(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0x01000000)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(true); err != nil {
		t.Fatal(err)
	}

	call, ok := provider.callOf("SetGamingProfile")
	if !ok {
		t.Fatal("the lock was not written")
	}
	if call.Input&0xFF != uint64(profileWinKeyLock) {
		t.Errorf("the switch is %d, want %d", call.Input&0xFF, profileWinKeyLock)
	}
	if (call.Input>>24)&0xFF != 1 {
		t.Errorf("the lock is %d, want 1", (call.Input>>24)&0xFF)
	}
}

func TestSetWinKeyUnlocksTheKey(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(false); err != nil {
		t.Fatal(err)
	}

	call, _ := provider.callOf("SetGamingProfile")
	if (call.Input>>24)&0xFF != 0 {
		t.Errorf("the lock is %d, want 0", (call.Input>>24)&0xFF)
	}
}

func TestSetWinKeyLockedReportsAChangeThatIsNotTaken(t *testing.T) {
	provider := newFakeProvider()
	provider.output("GetGamingProfile", "gmOutput", 0)
	backend := newFakeBackend(provider)

	if err := backend.SetWinKeyLocked(true); err == nil {
		t.Error("the change was accepted although the provider kept the old state")
	}
}
