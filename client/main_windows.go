//go:build windows

package main

import (
	"golang.org/x/sys/windows"
)

func init() {
	// ตั้งค่าสีและรูปแบบคอนโซล
	handle := windows.Handle(windows.STD_OUTPUT_HANDLE)
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
