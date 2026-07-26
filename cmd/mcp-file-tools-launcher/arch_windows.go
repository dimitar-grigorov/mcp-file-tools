// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

// IMAGE_FILE_MACHINE values as reported by IsWow64Process2.
const (
	imageFileMachineAMD64 = 0x8664
	imageFileMachineARM64 = 0xAA64
)

var procIsWow64Process2 = syscall.NewLazyDLL("kernel32.dll").NewProc("IsWow64Process2")

// nativeArch is the machine's architecture, not the launcher's. Only one launcher is
// committed and it is amd64, so runtime.GOARCH would send arm64 machines an emulated
// amd64 server when a native build is published for them.
func nativeArch() string {
	if procIsWow64Process2.Find() != nil {
		return runtime.GOARCH // pre-1709 Windows, which predates arm64 anyway
	}

	var process, native uint16
	ok, _, _ := procIsWow64Process2.Call(
		^uintptr(0), // GetCurrentProcess() pseudo-handle
		uintptr(unsafe.Pointer(&process)),
		uintptr(unsafe.Pointer(&native)))
	if ok == 0 {
		return runtime.GOARCH
	}

	switch native {
	case imageFileMachineARM64:
		return "arm64"
	case imageFileMachineAMD64:
		return "amd64"
	default:
		return runtime.GOARCH
	}
}
