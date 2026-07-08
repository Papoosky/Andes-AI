package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"github.com/andespath/andes-ai/internal/cli"
)

// TestBannerContainsCommands verifies that running bare andes (no args) prints
// the names of all three subcommands.
func TestBannerContainsCommands(t *testing.T) {
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	for _, want := range []string{"init", "list", "doctor"} {
		if !strings.Contains(output, want) {
			t.Errorf("banner output missing %q:\n%s", want, output)
		}
	}
}

// TestBannerContainsTitle verifies the CLI name appears in the banner.
func TestBannerContainsTitle(t *testing.T) {
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "andes") {
		t.Errorf("banner does not contain title 'andes':\n%s", out.String())
	}
}

// TestBannerContainsBraille verifies the logo section contains at least one
// braille character in the U+2801–U+28FF range.
func TestBannerContainsBraille(t *testing.T) {
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	brailleRange := &unicode.RangeTable{
		R16: []unicode.Range16{
			{Lo: 0x2801, Hi: 0x28FF, Stride: 1},
		},
	}
	found := false
	for _, r := range out.String() {
		if unicode.Is(brailleRange, r) {
			found = true
			break
		}
	}
	if !found {
		t.Error("banner output contains no braille runes (U+2801–U+28FF); logo may be missing")
	}
}
