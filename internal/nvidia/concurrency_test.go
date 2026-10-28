//go:build windows

package nvidia

import (
	"errors"
	"sync"
	"testing"
)

var errNilLibrary = errors.New("nvidia: the library is nil without an error")

func TestLoadIsCalledByManyReadersAtOnce(t *testing.T) {

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

	Close()

	const readers = 16
	var group sync.WaitGroup
	group.Add(readers)

	for i := 0; i < readers; i++ {
		go func() {
			defer group.Done()

			Read()
			Available()
		}()
	}
	group.Wait()
}

func TestCloseWhileTheLibraryIsUsed(t *testing.T) {

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

	Close()
	if _, err := load(); err != nil && !Available() {
		t.Errorf("the library cannot be opened again: %v", err)
	}
}

func TestTheLibraryIsOpenedOnlyOnce(t *testing.T) {

	Close()

	first, err := load()
	if err != nil {

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
