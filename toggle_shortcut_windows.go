//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	CMD_SHORTCUT_CONFIG   = 222
	CMD_SHORTCUT_CLEAR    = 223
	shortcutHotkeyIDFirst = 2
	shortcutHotkeyIDNext  = 3
	shortcutDialogSaveID  = 301
	shortcutDialogClearID = 302
	shortcutDialogCloseID = 303

	shortcutNoRepeat       = 0x4000
	shortcutWSChild        = 0x40000000
	shortcutWSVisible      = 0x10000000
	shortcutWSBorder       = 0x00800000
	shortcutWSTabStop      = 0x00010000
	shortcutWSOverlapped   = 0x00C80000 // Caption + system menu
	shortcutESAutoScroll   = 0x0080
	shortcutWSClientEdge   = 0x00000200
	shortcutSWShow         = 5
	shortcutWMSetFont      = 0x0030
	shortcutDefaultGUIFont = 17
)

var (
	shortcutDialogWindow          uintptr
	shortcutDialogClassRegistered bool
	shortcutInputWindow           uintptr
	currentShortcut               toggleShortcut
	registeredShortcutID          uintptr
	shortcutLoadError             string
	pDestroyShortcutWindow        = user32.NewProc("DestroyWindow")
	pFocusShortcutWindow          = user32.NewProc("SetFocus")
	pSendShortcutMessage          = user32.NewProc("SendMessageW")
	pGUIFont                      = gdi32.NewProc("GetStockObject")
)

func toggleShortcutPath() string { return filepath.Join(dirPath(), "toggle_shortcut_v1.txt") }

func loadToggleShortcut() {
	data, err := os.ReadFile(toggleShortcutPath())
	if errors.Is(err, os.ErrNotExist) {
		return
	} // No shortcut until the user chooses one.
	if err != nil {
		shortcutLoadError = "Could not load shortcut: " + err.Error()
		appendDiagnostic(shortcutLoadError)
		return
	}
	spec, err := parseToggleShortcut(string(data))
	if err != nil {
		shortcutLoadError = "Invalid saved shortcut: " + err.Error()
		appendDiagnostic(shortcutLoadError)
		return
	}
	ok, _, registerErr := pRegisterHotKey.Call(mainHWND, shortcutHotkeyIDFirst, uintptr(spec.Modifiers)|shortcutNoRepeat, uintptr(spec.Key))
	if ok == 0 {
		shortcutLoadError = fmt.Sprintf("Shortcut %s could not be registered (already used?): %v", spec.Display, registerErr)
		appendDiagnostic(shortcutLoadError)
		currentShortcut = spec // Show the stored shortcut and its unavailability in the menu.
		return
	}
	registeredShortcutID = shortcutHotkeyIDFirst
	currentShortcut = spec
	shortcutLoadError = ""
	appendDiagnostic("Toggle shortcut registered: " + spec.Display)
}

func saveToggleShortcut(input string) error {
	spec, err := parseToggleShortcut(input)
	if err != nil {
		return err
	}
	if registeredShortcutID != 0 && currentShortcut == spec {
		return nil
	}
	newID := uintptr(shortcutHotkeyIDFirst)
	if registeredShortcutID == shortcutHotkeyIDFirst {
		newID = shortcutHotkeyIDNext
	}
	ok, _, registerErr := pRegisterHotKey.Call(mainHWND, newID, uintptr(spec.Modifiers)|shortcutNoRepeat, uintptr(spec.Key))
	if ok == 0 {
		return fmt.Errorf("%s is already in use or reserved by Windows (error: %v). The previous shortcut was kept", spec.Display, registerErr)
	}
	if err := os.MkdirAll(dirPath(), 0700); err != nil {
		pUnregisterHotKey.Call(mainHWND, newID)
		return err
	}
	if err := os.WriteFile(toggleShortcutPath(), []byte(spec.Display+"\n"), 0600); err != nil {
		pUnregisterHotKey.Call(mainHWND, newID)
		return fmt.Errorf("shortcut registered temporarily but could not be saved: %w", err)
	}
	if registeredShortcutID != 0 {
		pUnregisterHotKey.Call(mainHWND, registeredShortcutID)
	}
	registeredShortcutID = newID
	currentShortcut = spec
	shortcutLoadError = ""
	appendDiagnostic("Toggle shortcut updated: " + spec.Display)
	return nil
}

func clearToggleShortcut() error {
	err := os.Remove(toggleShortcutPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not clear shortcut setting: %w", err)
	}
	if registeredShortcutID != 0 {
		pUnregisterHotKey.Call(mainHWND, registeredShortcutID)
	}
	registeredShortcutID = 0
	currentShortcut = toggleShortcut{}
	shortcutLoadError = ""
	appendDiagnostic("Toggle shortcut removed")
	return nil
}

func unhookToggleShortcut() {
	if registeredShortcutID != 0 {
		pUnregisterHotKey.Call(mainHWND, registeredShortcutID)
		registeredShortcutID = 0
	}
}

func toggleTooltipsFromMenuOrShortcut() {
	active = !active
	// Existing recognition/translation requests must not show a stale tooltip
	// when the user switches the feature off and quickly on again.
	hoverGeneration++
	selectionSequence++
	dragDown, dragQualified, selectionBusy, sentenceShown = false, false, false, false
	lastChecked = point{}
	lastMoved = time.Now()
	lastCheckAt = time.Time{}
	hide231Dictionary()
	state := "OFF"
	if active {
		state = "ON"
	}
	appendDiagnostic("Tooltips toggled " + state)
}

func shortcutMenuLabel() string {
	if currentShortcut.Display == "" {
		return "Set / Change Toggle Shortcut... (Not set)"
	}
	if registeredShortcutID == 0 {
		return "Set / Change Toggle Shortcut... (Unavailable: " + currentShortcut.Display + ")"
	}
	return "Set / Change Toggle Shortcut... (" + currentShortcut.Display + ")"
}

func shortcutCreateChild(class, label string, style uintptr, x, y, w, h int, id uintptr) uintptr {
	instance, _, _ := pGetModuleHandle.Call(0)
	hwnd, _, _ := pCreateWindow.Call(0,
		uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(label))),
		shortcutWSChild|shortcutWSVisible|style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), shortcutDialogWindow, id, instance, 0)
	if hwnd != 0 {
		font, _, _ := pGUIFont.Call(shortcutDefaultGUIFont)
		pSendShortcutMessage.Call(hwnd, shortcutWMSetFont, font, 1)
	}
	return hwnd
}

func openShortcutDialog() {
	if shortcutDialogWindow != 0 {
		pSetForeground.Call(shortcutDialogWindow)
		if shortcutInputWindow != 0 {
			pFocusShortcutWindow.Call(shortcutInputWindow)
		}
		return
	}
	instance, _, _ := pGetModuleHandle.Call(0)
	// Use a distinct window class so that the self-installer's FindWindow on
	// 231DictionaryWindow can never accidentally close this settings window.
	if !shortcutDialogClassRegistered {
		class := ptr("231DictionaryShortcutSettings")
		wc := wndClassEx{
			Size:       uint32(unsafe.Sizeof(wndClassEx{})),
			Proc:       syscall.NewCallback(windowProc),
			Instance:   instance,
			Background: 6, // COLOR_WINDOW+1
			ClassName:  class,
		}
		if ok, _, registerErr := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); ok == 0 {
			callMessage("Could not register shortcut settings window: "+fmt.Sprint(registerErr), MB_ICONWARNING)
			return
		}
		shortcutDialogClassRegistered = true
	}
	cx, _, _ := pGetSystemMetrics.Call(0)
	cy, _, _ := pGetSystemMetrics.Call(1)
	x, y := int(cx)/2-238, int(cy)/2-111
	if x < 0 {
		x = 10
	}
	if y < 0 {
		y = 10
	}
	hwnd, _, err := pCreateWindow.Call(WS_EX_TOOLWINDOW|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(ptr("231DictionaryShortcutSettings"))),
		uintptr(unsafe.Pointer(ptr("231Dictionary - Toggle Shortcut"))),
		shortcutWSOverlapped|shortcutWSVisible,
		uintptr(x), uintptr(y), 476, 222, mainHWND, 0, instance, 0)
	if hwnd == 0 {
		callMessage("Could not open shortcut settings: "+fmt.Sprint(err), MB_ICONWARNING)
		return
	}
	shortcutDialogWindow = hwnd
	shortcutCreateChild("STATIC", "Toggle 'Enable / Disable Word Tooltips' globally.", 0, 18, 15, 426, 23, 0)
	shortcutCreateChild("STATIC", "Type Ctrl+Alt+T or Ctrl+Shift+T (letters, 0-9, F1-F11).", 0, 18, 45, 432, 23, 0)
	val := currentShortcut.Display
	if val == "" {
		val = "Ctrl+Alt+T"
	}
	shortcutInputWindow = shortcutCreateChild("EDIT", val, shortcutWSBorder|shortcutWSTabStop|shortcutESAutoScroll, 18, 76, 434, 26, 0)
	shortcutCreateChild("BUTTON", "Save", shortcutWSTabStop, 18, 125, 115, 32, shortcutDialogSaveID)
	shortcutCreateChild("BUTTON", "Remove Shortcut", shortcutWSTabStop, 147, 125, 166, 32, shortcutDialogClearID)
	shortcutCreateChild("BUTTON", "Cancel", shortcutWSTabStop, 327, 125, 125, 32, shortcutDialogCloseID)
	pSetForeground.Call(shortcutDialogWindow)
	if shortcutInputWindow != 0 {
		pFocusShortcutWindow.Call(shortcutInputWindow)
	}
}

func shortcutEditText() string {
	if shortcutInputWindow == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	n, _, _ := pGetWindowTextW.Call(shortcutInputWindow, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return strings.TrimSpace(syscall.UTF16ToString(buf[:n]))
}

func handleShortcutWindowMessage(hwnd, message, wp, lp uintptr) (uintptr, bool) {
	if hwnd == 0 || hwnd != shortcutDialogWindow {
		return 0, false
	}
	switch message {
	case WM_COMMAND:
		id := wp & 0xFFFF
		switch id {
		case shortcutDialogSaveID:
			if err := saveToggleShortcut(shortcutEditText()); err != nil {
				callMessage("Shortcut not saved:\n"+err.Error(), MB_ICONWARNING)
				return 0, true
			}
			pDestroyShortcutWindow.Call(hwnd)
			return 0, true
		case shortcutDialogClearID:
			if err := clearToggleShortcut(); err != nil {
				callMessage("Shortcut not cleared:\n"+err.Error(), MB_ICONWARNING)
				return 0, true
			}
			pDestroyShortcutWindow.Call(hwnd)
			return 0, true
		case shortcutDialogCloseID:
			pDestroyShortcutWindow.Call(hwnd)
			return 0, true
		}
	case WM_CLOSE:
		pDestroyShortcutWindow.Call(hwnd)
		return 0, true
	case WM_DESTROY:
		shortcutDialogWindow = 0
		shortcutInputWindow = 0
		return 0, true
	}
	return 0, false
}
