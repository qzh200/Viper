//go:build tools

// Command miscwatch polls the miscellaneous settings of the Acer gaming
// provider and writes every change to a log file.
//
// The provider packs a setting into a single word, the value in the upper
// byte and the identifier in the lower one. Watching the words while the
// official application changes a setting shows which identifier belongs to
// which feature.
//
// With -set the tool writes a single setting and stops, which makes it
// possible to put a setting back to a known value.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"viper/internal/hardware"
)

func main() {
	out := flag.String("out", "miscwatch.log", "file the changes are written to")
	seconds := flag.Int("every", 1, "seconds between two polls")
	count := flag.Int("count", 0x21, "highest setting identifier that is polled")
	set := flag.String("set", "", "write a setting, given as identifier=value")
	flag.Parse()

	if *set != "" {
		if err := writeSetting(*set); err != nil {
			fmt.Fprintln(os.Stderr, "miscwatch:", err)
			os.Exit(1)
		}
		return
	}

	log, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "miscwatch:", err)
		os.Exit(1)
	}
	defer log.Close()

	backend, err := hardware.NewBackend()
	if err != nil {
		fmt.Fprintln(os.Stderr, "miscwatch:", err)
		os.Exit(1)
	}
	defer backend.Close()

	previous := map[uint32]uint64{}
	for {
		for id := uint32(0); id <= uint32(*count); id++ {
			value, err := backend.MiscSettingValue(id)
			if err != nil {
				continue
			}
			if old, ok := previous[id]; !ok || old != value {
				previous[id] = value
				fmt.Fprintf(log, "%s id=0x%02X raw=0x%08X value=%d status=%d\n",
					time.Now().Format("15:04:05.000"), id, value, value>>8, value&0xFF)
				log.Sync()
			}
		}
		time.Sleep(time.Duration(*seconds) * time.Second)
	}
}

// writeSetting writes a single setting, given as identifier=value.
func writeSetting(spec string) error {
	name, raw, ok := strings.Cut(spec, "=")
	if !ok {
		return fmt.Errorf("expected identifier=value, got %q", spec)
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(name, "0x"), 16, 32)
	if err != nil {
		return fmt.Errorf("identifier %q: %w", name, err)
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("value %q: %w", raw, err)
	}

	backend, err := hardware.NewBackend()
	if err != nil {
		return err
	}
	defer backend.Close()

	if err := backend.SetMiscSettingValue(uint32(id), value); err != nil {
		return err
	}
	raw2, err := backend.MiscSettingValue(uint32(id))
	if err != nil {
		return err
	}
	fmt.Printf("id=0x%02X value=%d raw=0x%08X\n", id, value, raw2)
	return nil
}
