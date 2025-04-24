//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func init() {
	// ตั้งค่าไอคอนสำหรับ Windows Console
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleIcon := kernel32.NewProc("SetConsoleIcon")

	// โหลดไอคอน
	icon, err := os.ReadFile("assets/phi-xml-icon.ico")
	if err == nil {
		setConsoleIcon.Call(uintptr(unsafe.Pointer(&icon[0])))
	}
}
