//go:build windows

package main

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

var (
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pGlobalFree       = kernel32.NewProc("GlobalFree")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 2
)

var (
	popupDragging        bool
	popupEntered         bool
	popupShowAt          time.Time
	popupDragCursor      point
	popupDragBounds      rect
	popupEnglishCopyBody string
)

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
	for i := 0; i < 6; i++ {
		ok, _, _ := pOpenClipboard.Call(owner)
		if ok != 0 {
			opened = true
			break
		}
		time.Sleep(25 * time.Millisecond)
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

func currentEnglishTranslationForCopy() string {
	uiMu.Lock()
	mode := tipMode
	uiMu.Unlock()
	if !tipShown || !sentenceShown {
		return ""
	}
	if mode != "sentence" {
		return ""
	}
	return popupEnglishCopyBody
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
	if popupDragging {
		pReleaseCapture.Call()
	}
	popupDragging = false
	popupEntered = false
	popupShowAt = time.Time{}
	popupEnglishCopyBody = ""
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
	if popupDragging {
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
		return 0, true
	case WM_LBUTTONUP:
		if popupDragging {
			popupDragging = false
			pReleaseCapture.Call()
		}
		return 0, true
	case WM_RBUTTONUP:
		if value := currentEnglishTranslationForCopy(); value != "" {
			if err := copyUnicodeTextToClipboard(tipHWND, value); err != nil {
				appendDiagnostic("English translation copy error: " + err.Error())
			} else {
				appendDiagnostic("English translated sentence copied")
			}
		}
		return 0, true
	case WM_CONTEXTMENU:
		return 0, true
	}
	return 0, false
}
