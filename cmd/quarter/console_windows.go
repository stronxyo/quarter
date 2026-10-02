//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// colourOutput lets the console read colour codes for the goodbye, printed
// after Bubble Tea has given the console its old mode back: the classic
// console (conhost) shows the codes as text unless virtual terminal
// processing is on. Windows Terminal always reads them. ok is false when
// standard output isn't a console that can (redirected to a file, or a
// Windows too old for it); restore puts the mode back.
func colourOutput() (restore func(), ok bool) {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return func() {}, false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return func() {}, true
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return func() {}, false
	}
	return func() { windows.SetConsoleMode(h, mode) }, true
}
