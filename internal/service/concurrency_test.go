//go:build windows

package service

import (
	"sync"
	"testing"
)

// The service answers every connection in its own goroutine, so the
// readings of the processor run side by side. The state of the last
// reading is shared by all of them and has to be guarded.

func TestCPUUsageIsCalledByManyReadersAtOnce(t *testing.T) {
	// The load is a difference between two readings, so the state of the
	// last one is shared. Without the lock a reader can see a half
	// written table and the load comes out wrong.
	const readers = 16
	var group sync.WaitGroup
	group.Add(readers)

	for i := 0; i < readers; i++ {
		go func() {
			defer group.Done()
			for j := 0; j < 50; j++ {
				load := cpuUsage()
				if load < 0 || load > 100 {
					t.Errorf("the load of the processor is %f percent, want a number between zero and one hundred", load)
					return
				}
			}
		}()
	}
	group.Wait()
}

func TestTheFirstReadingHasNoLoad(t *testing.T) {
	// The first answer has nothing to compare with, so it is zero. The
	// state is reset, so that the test does not depend on the other
	// tests.
	processorTimes.mutex.Lock()
	processorTimes.havePrevious = false
	processorTimes.mutex.Unlock()

	if load := cpuUsage(); load != 0 {
		t.Errorf("the first reading is %f percent, want zero", load)
	}
}

func TestTheSecondReadingHasALoad(t *testing.T) {
	processorTimes.mutex.Lock()
	processorTimes.havePrevious = false
	processorTimes.mutex.Unlock()

	cpuUsage()
	// The second reading compares itself with the first one.
	load := cpuUsage()
	if load < 0 || load > 100 {
		t.Errorf("the second reading is %f percent, want a number between zero and one hundred", load)
	}
}

func TestTheMemoryOfTheMachine(t *testing.T) {
	total, used := memoryInfo()
	if total == 0 {
		t.Skip("the memory cannot be read on this system")
	}
	if used < 0 || used > 100 {
		t.Errorf("the memory is used by %f percent, want a number between zero and one hundred", used)
	}
}

func TestTheProcessorOfTheMachine(t *testing.T) {
	// The readings of the processor come from the description of the
	// hardware, they are there on every machine that runs Windows.
	if count := processorCount(); count < 0 {
		t.Errorf("the machine reports %d processors", count)
	}
	if clock := processorClock(); clock < 0 {
		t.Errorf("the machine reports the clock %d megahertz", clock)
	}
	if name := processorName(); name == "" {
		t.Log("the processor has no name in the description of the hardware")
	}
}
