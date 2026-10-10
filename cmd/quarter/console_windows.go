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

var procGetConsoleWindow = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow")

// classicConsole reports whether Quarter runs in the classic console
// (conhost), whose window has the class ConsoleWindowClass. Windows
// Terminal, VS Code and other terminals host Quarter through a
// pseudoconsole, whose console window is a hidden one of another class.
func classicConsole() bool {
	if procGetConsoleWindow.Find() != nil {
		return false
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return false
	}
	name := make([]uint16, 64)
	n, err := windows.GetClassName(windows.HWND(hwnd), &name[0], int32(len(name)))
	if err != nil {
		return false
	}
	return windows.UTF16ToString(name[:n]) == "ConsoleWindowClass" && windows.IsWindowVisible(windows.HWND(hwnd))
}
