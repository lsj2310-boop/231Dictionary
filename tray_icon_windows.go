//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	pCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	pDestroyTrayIcon          = user32.NewProc("DestroyIcon")
	trayIconOn                uintptr
	trayIconOff               uintptr
	trayIconOnOwned           bool
	trayIconOffOwned          bool
	trayDoubleClickUntil      int64 // Unix nanoseconds; only accessed on the window thread
)

func initializeTrayIcons() {
	trayIconOn = loadEmbeddedTrayIcon("tray_on.bin")
	trayIconOnOwned = trayIconOn != 0
	trayIconOff = loadEmbeddedTrayIcon("tray_off.bin")
	trayIconOffOwned = trayIconOff != 0
	if trayIconOn == 0 {
		trayIconOn, _, _ = pLoadIcon.Call(0, 32516) // system information symbol, no ownership
		appendDiagnostic("colored notification icon unavailable; using Windows fallback")
	}
	if trayIconOff == 0 {
		trayIconOff, _, _ = pLoadIcon.Call(0, 32512) // system application icon, no ownership
		appendDiagnostic("gray notification icon unavailable; using Windows fallback")
	}
}

func loadEmbeddedTrayIcon(filename string) uintptr {
	data, err := embedded.ReadFile(filename)
	if err != nil || len(data) == 0 {
		appendDiagnostic("cannot load " + filename + ": " + fmt.Sprint(err))
		return 0
	}
	icon, _, callErr := pCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 1, 0x00030000, 32, 32, 0)
	if icon == 0 {
		appendDiagnostic(fmt.Sprintf("CreateIconFromResourceEx(%s) failed: %v", filename, callErr))
	}
	return icon
}

func selectedTrayIcon() uintptr {
	if active {
		return trayIconOn
	}
	return trayIconOff
}
func trayTooltip() string {
	if active {
		return "231Dictionary - Translation ON (click to pause)"
	}
	return "231Dictionary - Translation OFF (click to resume)"
}
func fillTrayNotification(ni *notifyIconData, flags uint32) {
	ni.CbSize = uint32(unsafe.Sizeof(*ni))
	ni.Hwnd = mainHWND
	ni.ID = 1
	ni.Flags = flags
	ni.Icon = selectedTrayIcon()
	copy(ni.Tip[:], syscall.StringToUTF16(trayTooltip()))
}
func refreshTrayIcon() {
	if mainHWND == 0 {
		return
	}
	var ni notifyIconData
	fillTrayNotification(&ni, NIF_ICON|NIF_TIP)
	if r, _, err := pShellNotify.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&ni))); r == 0 {
		appendDiagnostic("Shell_NotifyIcon(NIM_MODIFY) failed: " + fmt.Sprint(err))
	}
}
func disposeTrayIcons() {
	if trayIconOnOwned && trayIconOn != 0 {
		pDestroyTrayIcon.Call(trayIconOn)
	}
	if trayIconOffOwned && trayIconOff != 0 {
		pDestroyTrayIcon.Call(trayIconOff)
	}
	trayIconOn, trayIconOff = 0, 0
	trayIconOnOwned, trayIconOffOwned = false, false
}
func translationStatePath() string { return filepath.Join(dirPath(), "translation_enabled_v1.txt") }
func loadTranslationState() {
	data, err := os.ReadFile(translationStatePath())
	if os.IsNotExist(err) {
		return
	} // Default is enabled for new installations.
	if err != nil {
		appendDiagnostic("Cannot read toggle state: " + err.Error())
		return
	}
	value, parseErr := persistedTranslationState(string(data))
	if parseErr != nil {
		appendDiagnostic("Bad translation toggle setting: " + parseErr.Error())
		return
	}
	active = value
}
func saveTranslationState() {
	if err := os.MkdirAll(dirPath(), 0700); err != nil {
		appendDiagnostic("Cannot create settings directory: " + err.Error())
		return
	}
	if err := os.WriteFile(translationStatePath(), []byte(serializeTranslationState(active)), 0600); err != nil {
		appendDiagnostic("Cannot save toggle state: " + err.Error())
	}
}
