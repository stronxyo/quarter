package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// printed is what f writes to standard output.
func printed(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	f()
	os.Stdout = old
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

// The goodbye is printed in colour where the console reads colour codes,
// as plain text where it can't, never as the codes themselves; the
// console's mode is always put back.
func TestPrintGoodbye(t *testing.T) {
	text := "\x1b[38;2;255;224;138mkeep the change\x1b[m\n"
	for _, can := range []bool{true, false} {
		restored := false
		colour := func() (func(), bool) { return func() { restored = true }, can }
		out := printed(t, func() { printGoodbye(text, colour) })
		if !restored {
			t.Errorf("colour %v: the console's mode should be put back", can)
		}
		if can && out != text {
			t.Errorf("colour on: want the goodbye in colour, got %q", out)
		}
		if !can && (strings.Contains(out, "\x1b") || out != "keep the change\n") {
			t.Errorf("colour off: want plain text, got %q", out)
		}
	}
}
