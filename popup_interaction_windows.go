//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	pOpenClipboard       = user32.NewProc("OpenClipboard")
	pEmptyClipboard      = user32.NewProc("EmptyClipboard")
	pSetClipboardData    = user32.NewProc("SetClipboardData")
	pCloseClipboard      = user32.NewProc("CloseClipboard")
	pWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	pReleaseMutex        = kernel32.NewProc("ReleaseMutex")
	pGlobalAlloc         = kernel32.NewProc("GlobalAlloc")
	pGlobalLock          = kernel32.NewProc("GlobalLock")
	pGlobalUnlock        = kernel32.NewProc("GlobalUnlock")
	pGlobalFree          = kernel32.NewProc("GlobalFree")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 2
)

var (
	popupDragging            bool
	popupRightCaptured       bool
	popupEntered             bool
	popupShowAt              time.Time
	popupDragCursor          point
	popupDragBounds          rect
	popupTranslationCopyBody string
)

// Use the same named mutex as the selection worker. The PowerShell fallback
// must finish restoring its temporary clipboard before the user's explicit
// right-click copy takes ownership of the clipboard.
func withClipboardTransaction(task func() error) error {
	// Win32 named mutexes and OpenClipboard are bound to the calling OS thread.
	// A Go goroutine can migrate unless we pin it until ReleaseMutex/CloseClipboard.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	name, err := syscall.UTF16PtrFromString(`Local\231DictionaryClipboardTransaction`)
	if err != nil {
		return err
	}
	handle, _, _ := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return errors.New("failed to create clipboard mutex")
	}
	defer pCloseHandle.Call(handle)
	state, _, _ := pWaitForSingleObject.Call(handle, 3000)
	if state != 0 && state != 0x80 { // WAIT_OBJECT_0 or WAIT_ABANDONED
		return fmt.Errorf("clipboard transaction wait failed: 0x%x", state)
	}
	defer pReleaseMutex.Call(handle)
	return task()
}

// Complete the clipboard write *before* returning from the right-click.
// The previous asynchronous implementation let Ctrl+V race the copy, so users
// sometimes pasted the last selected source sentence instead of the translated
// sentence. The translation text is snapshotted before acquiring the lock.
func copyPopupTranslationNow(text string) {
	if text == "" {
		appendDiagnostic("Translation copy skipped: no completed translation is displayed")
		return
	}
	err := withClipboardTransaction(func() error { return copyUnicodeTextToClipboard(tipHWND, text) })
	if err != nil {
		appendDiagnostic("Translated sentence copy failed: " + err.Error())
		callMessage("Could not copy the translation. The clipboard is busy.\nPlease right-click the translation again.\n\n"+err.Error(), MB_ICONWARNING)
		return
	}
	appendDiagnostic("Displayed translated sentence copied to clipboard")
}

func copyUnicodeTextToClipboard(owner uintptr, text string) error {
	if text == "" {
		return errors.New("no translation to copy")
	}
	utf16 := syscall.StringToUTF16(text)
	if len(utf16) == 0 {
		return errors.New("empty translation")
	}
	sizeBytes := uintptr(len(utf16) * 2)
	global, _, _ := pGlobalAlloc.Call(GMEM_MOVEABLE, sizeBytes)
	if global == 0 {
		return errors.New("GlobalAlloc failed")
	}
	// Ownership of this handle passes to the clipboard ONLY when
	// SetClipboardData succeeds.
	owned := true
	defer func() {
		if owned {
			pGlobalFree.Call(global)
		}
	}()
	buf, _, _ := pGlobalLock.Call(global)
	if buf == 0 {
		return errors.New("GlobalLock failed")
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(buf)), len(utf16))
	copy(dst, utf16)
	pGlobalUnlock.Call(global)
	opened := false
	for i := 0; i < 30; i++ {
		ok, _, _ := pOpenClipboard.Call(owner)
		if ok != 0 {
			opened = true
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if !opened {
		return errors.New("clipboard is busy")
	}
	defer pCloseClipboard.Call()
	ok, _, _ := pEmptyClipboard.Call()
	if ok == 0 {
		return errors.New("EmptyClipboard failed")
	}
	ok, _, _ = pSetClipboardData.Call(CF_UNICODETEXT, global)
	if ok == 0 {
		return errors.New("SetClipboardData failed")
	}
	owned = false
	return nil
}

func currentTranslationForCopy() string {
	uiMu.Lock()
	mode, visible := tipMode, tipMean
	uiMu.Unlock()
	return displayedTranslationForCopy(visible, popupTranslationCopyBody, tipShown, sentenceShown, mode)
}

func popupJustShown() {
	// An async response can change the popup while the mouse is in it;
	// retain the entered state only when the pointer is still inside.
	p := mouse()
	inside := pointIsInPopup(p)
	popupEntered = inside
	popupShowAt = time.Now()
}
func resetPopupPointerState() {
	if popupDragging || popupRightCaptured {
		pReleaseCapture.Call()
	}
	popupDragging = false
	popupRightCaptured = false
	popupEntered = false
	popupShowAt = time.Time{}
	popupTranslationCopyBody = ""
}
func pointIsInPopup(p point) bool {
	if !tipShown || tipHWND == 0 {
		return false
	}
	var r rect
	ok, _, _ := pGetWindowRect.Call(tipHWND, uintptr(unsafe.Pointer(&r)))
	return ok != 0 && p.X >= r.Left && p.Y >= r.Top && p.X < r.Right && p.Y < r.Bottom
}
func nearPopup(p point, margin int32) bool {
	var r rect
	ok, _, _ := pGetWindowRect.Call(tipHWND, uintptr(unsafe.Pointer(&r)))
	return ok != 0 && p.X >= r.Left-margin && p.Y >= r.Top-margin && p.X < r.Right+margin && p.Y < r.Bottom+margin
}

// Keep popup alive while the cursor is inside, and dismiss only after it has
// entered then left. Users also have time to move from the selection to the
// tooltip without the old 4-pixel movement dismissal.
func handlePopupPointerTimer(p point, now time.Time) bool {
	if !tipShown {
		return false
	}
	if popupDragging || popupRightCaptured {
		return true
	}
	if pointIsInPopup(p) {
		popupEntered = true
		sentenceUntil = now.Add(32 * time.Second)
		return true
	}
	if popupEntered {
		sentenceShown = false
		hide231Dictionary()
		lastMoved = now
		return false
	}
	if !popupShowAt.IsZero() && now.Sub(popupShowAt) < 2500*time.Millisecond && nearPopup(p, 35) {
		return true
	}
	return false
}

func handlePopupMouseMessage(message, wp, lp uintptr) (uintptr, bool) {
	switch message {
	case WM_LBUTTONDOWN:
		popupEntered = true
		popupDragging = true
		popupDragCursor = mouse()
		ok, _, _ := pGetWindowRect.Call(tipHWND, uintptr(unsafe.Pointer(&popupDragBounds)))
		if ok == 0 {
			popupDragging = false
			return 0, true
		}
		pSetCapture.Call(tipHWND)
		return 0, true
	case WM_RBUTTONDOWN:
		popupEntered = true
		popupRightCaptured = true
		pSetCapture.Call(tipHWND)
		return 0, true
	case WM_MOUSEMOVE:
		if popupDragging {
			p := mouse()
			dx, dy := int32(p.X-popupDragCursor.X), int32(p.Y-popupDragCursor.Y)
			x, y := popupDragBounds.Left+dx, popupDragBounds.Top+dy
			pSetWindowPos.Call(tipHWND, HWND_TOPMOST, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW)
		}
		return 0, true
	case WM_CAPTURECHANGED:
		popupDragging = false
		popupRightCaptured = false
		return 0, true
	case WM_LBUTTONUP:
		if popupDragging {
			popupDragging = false
			pReleaseCapture.Call()
		}
		return 0, true
	case WM_RBUTTONUP:
		// Capture means the mouse-up is delivered even if the pointer moves
		// slightly beyond the popup after pressing the right button.
		if popupRightCaptured {
			popupRightCaptured = false
			pReleaseCapture.Call()
		}
		copyPopupTranslationNow(currentTranslationForCopy())
		return 0, true
	case WM_CONTEXTMENU:
		// Keyboard context-menu invocation (Shift+F10/menu key) has -1/-1
		// coordinates. Mouse right-click is already handled by WM_RBUTTONUP.
		if int16(lp&0xffff) == -1 && int16((lp>>16)&0xffff) == -1 {
			copyPopupTranslationNow(currentTranslationForCopy())
		}
		return 0, true
	}
	return 0, false
}
