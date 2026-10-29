package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTheScreenshotPathIsRead(t *testing.T) {
	opts, err := parseArgs([]string{"--screenshot", "viper.png"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.screenshot != "viper.png" {
		t.Errorf("the path is %q, want viper.png", opts.screenshot)
	}
	if opts.help {
		t.Error("the help was asked for")
	}
}

func TestWithoutArgumentsNothingIsCaptured(t *testing.T) {
	opts, err := parseArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.screenshot != "" {
		t.Errorf("the path is %q, want nothing", opts.screenshot)
	}
}

func TestTheScreenshotPathIsNeeded(t *testing.T) {
	if _, err := parseArgs([]string{"--screenshot"}); err == nil {
		t.Error("the missing path was accepted")
	}
}

func TestAnUnknownOptionIsRefused(t *testing.T) {
	if _, err := parseArgs([]string{"--nope"}); err == nil {
		t.Error("an unknown option was accepted")
	}
}

func TestTheHelpIsAskedFor(t *testing.T) {
	opts, err := parseArgs([]string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.help {
		t.Error("the help was not asked for")
	}
}

func TestTheUsageTalksAboutTheAppearance(t *testing.T) {
	var out bytes.Buffer
	usage(&out)

	text := out.String()
	for _, want := range []string{"viperui", "--screenshot", "light, dark"} {
		if !strings.Contains(text, want) {
			t.Errorf("the usage has no %q in %q", want, text)
		}
	}
}
