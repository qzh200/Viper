//go:build windows

package nvidia

import (
	"errors"
	"sync"
	"testing"
)

// errNilLibrary marks the state that the lock prevents: a library that
// is not there but no error either.
var errNilLibrary = errors.New("nvidia: the library is nil without an error")

// The service answers every connection in its own goroutine, so the
// readings of the graphics card run side by side. The state of the
// library is shared by all of them and has to be guarded.

func TestLoadIsCalledByManyReadersAtOnce(t *testing.T) {
	// The race is between the goroutine that opens the library and the
	// ones that find it already opened. Without the lock the second
	// goroutine would see a table that is not filled yet and would
	// dereference a nil pointer.
	Close()

	const readers = 32
	var group sync.WaitGroup
	group.Add(readers)

	results := make([]error, readers)
	for i := 0; i < readers; i++ {
		go func(index int) {
			defer group.Done()
			lib, err := load()
			if err == nil && lib == nil {
				// A nil library without an error is the crash that the
				// lock prevents.
				results[index] = errNilLibrary
				return
			}
			results[index] = err
		}(i)
	}
	group.Wait()

	for i, err := range results {
		if err == errNilLibrary {
			t.Fatalf("the reader %d got a nil library without an error", i)
		}
	}
}

func TestReadIsCalledByManyReadersAtOnce(t *testing.T) {
	// The whole reading has to survive the same race, because the
	// service asks for the readings of the card from every connection.
	Close()

	const readers = 16
	var group sync.WaitGroup
	group.Add(readers)

	for i := 0; i < readers; i++ {
		go func() {
			defer group.Done()
			// A machine without the driver answers with an error, that
			// is fine. What must not happen is a panic.
			Read()
			Available()
		}()
	}
	group.Wait()
}

func TestCloseWhileTheLibraryIsUsed(t *testing.T) {
	// Closing the library and reading at the same time must not crash.
	Close()

	var group sync.WaitGroup
	group.Add(8)

	for i := 0; i < 4; i++ {
		go func() {
			defer group.Done()
			for j := 0; j < 20; j++ {
				Read()
			}
		}()
	}
	for i := 0; i < 4; i++ {
		go func() {
			defer group.Done()
			for j := 0; j < 20; j++ {
				Close()
			}
		}()
	}
	group.Wait()

	// The library can be opened again after it was closed.
	Close()
	if _, err := load(); err != nil && !Available() {
		t.Errorf("the library cannot be opened again: %v", err)
	}
}

func TestTheLibraryIsOpenedOnlyOnce(t *testing.T) {
	// Every reading after the first one has to find the same library.
	Close()

	first, err := load()
	if err != nil {
		// A machine without the driver cannot be tested here.
		return
	}
	for i := 0; i < 8; i++ {
		next, err := load()
		if err != nil {
			t.Fatalf("the library cannot be opened: %v", err)
		}
		if next != first {
			t.Fatal("the library was opened a second time")
		}
	}
}
