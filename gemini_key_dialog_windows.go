//go:build windows

package main

// Uses the OS-provided Windows credential dialog rather than a hand-written
// WNDCLASSEX window with a nested message loop. This avoids the v2.6 failure
// where the menu item activated but no key-entry window became visible.
// Credentials are NOT stored in Windows Credential Manager; the application
// encrypts the key with per-user Windows DPAPI in gemini_key_windows.go.

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type credUIInfo struct {
	Size    uint32
	Parent  uintptr
	Message *uint16
	Caption *uint16
	Banner  uintptr
}

const (
	credUIGenericCredentials  = 0x00040000
	credUIDoNotPersist        = 0x00000002
	credUIAlwaysShow          = 0x00000080
	credUIExcludeCertificates = 0x00000008
	credUIPasswordOnlyOK      = 0x00000200
	winErrorCancelled         = 1223
	winErrorInvalidFlags      = 1004
	winErrorInvalidParameter  = 87
)

var (
	credUIDLL            = syscall.NewLazyDLL("credui.dll")
	promptForCredentials = credUIDLL.NewProc("CredUIPromptForCredentialsW")
)

func promptAndSaveGeminiKey() bool {
	title := syscall.StringToUTF16Ptr("231Dictionary - Set Gemini API Key")
	explanation := syscall.StringToUTF16Ptr(
		"Paste your Gemini API key into the Password field and select OK.\n" +
			"You may leave the User name as 231Dictionary.\n" +
			"Your key will be encrypted for your Windows user account on this PC.")
	target := syscall.StringToUTF16Ptr("231Dictionary.GeminiAPI")
	ui := credUIInfo{
		Size:    uint32(unsafe.Sizeof(credUIInfo{})),
		Parent:  0, // hidden tray windows are not suitable modal parents
		Message: explanation,
		Caption: title,
	}
	username := make([]uint16, 256)
	copy(username, syscall.StringToUTF16("231Dictionary"))
	password := make([]uint16, 256)
	defer func() {
		for i := range username {
			username[i] = 0
		}
		for i := range password {
			password[i] = 0
		}
	}()
	var save int32
	// Try password-only first. If this flag is rejected by a Windows policy
	// or version, retry the standard native username/password form.
	baseFlags := uintptr(credUIGenericCredentials | credUIDoNotPersist | credUIAlwaysShow | credUIExcludeCertificates)
	invoke := func(flags uintptr) uintptr {
		result, _, _ := promptForCredentials.Call(
			uintptr(unsafe.Pointer(&ui)), uintptr(unsafe.Pointer(target)), 0, 0,
			uintptr(unsafe.Pointer(&username[0])), uintptr(len(username)),
			uintptr(unsafe.Pointer(&password[0])), uintptr(len(password)),
			uintptr(unsafe.Pointer(&save)), flags,
		)
		return result
	}
	status := invoke(baseFlags | credUIPasswordOnlyOK)
	if status == winErrorInvalidFlags || status == winErrorInvalidParameter {
		status = invoke(baseFlags)
	}
	if status == winErrorCancelled {
		appendDiagnostic("Gemini API key entry canceled")
		return false
	}
	if status != 0 {
		appendDiagnostic(fmt.Sprintf("Gemini Windows Credential UI failure: %d", status))
		callMessage(fmt.Sprintf("Could not open the Windows API key dialog.\nError code: %d\n\nTry again or check your Windows security policy.", status), MB_ICONWARNING)
		return false
	}
	key := strings.TrimSpace(syscall.UTF16ToString(password))
	if err := saveGeminiKey(key); err != nil {
		appendDiagnostic("Gemini key DPAPI save failed: " + err.Error())
		callMessage("Failed to save Gemini API key: "+err.Error(), MB_ICONWARNING)
		return false
	}
	appendDiagnostic("Gemini API key saved with Windows DPAPI encryption")
	return true
}

func openGeminiKeyDialog() {
	if !promptAndSaveGeminiKey() {
		return
	}
	callMessage("Your Gemini API key has been encrypted and saved on this PC.\n\nThe app will test the connection with a sample sentence.", MB_ICONINFORMATION)
	go verifyGeminiConnection()
}

func runGeminiKeyRegistrationStandalone() {
	if !promptAndSaveGeminiKey() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	translated, _, err := translateGemini(ctx, "Cab tilt system malfunction.")
	if err != nil {
		appendDiagnostic("Gemini standalone connection test failure: " + err.Error())
		callMessage("API key saved, but the translation server connection failed.\n\n"+err.Error(), MB_ICONWARNING)
		return
	}
	callMessage("Gemini connection successful!\n\nCab tilt system malfunction.\n→ "+translated, MB_ICONINFORMATION)
}

func verifyGeminiConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	translated, _, err := translateGemini(ctx, "Cab tilt system malfunction.")
	uiMu.Lock()
	if err != nil {
		testText = "API key saved; Gemini translation verification failed\n\n" + err.Error() + "\n\nA 503 error indicates temporary server overload. You do not need to register your API key again."
	} else {
		testText = "Gemini connection and sample translation successful!\n\nCab tilt system malfunction.\n→ " + translated
	}
	uiMu.Unlock()
	if err != nil {
		appendDiagnostic("Gemini verification failure: " + err.Error())
	} else {
		appendDiagnostic("Gemini verification succeeded")
	}
	pPostMessage.Call(mainHWND, WM_TEST_RESULT, 0, 0)
}
