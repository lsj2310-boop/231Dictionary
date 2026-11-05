//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"

	"os"

	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Windows-only selection detection. No clipboard operations or keystroke injection:
// avoids replacing the user's clipboard or inadvertently copying PPT objects.
var pGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
var sentenceOnlineAllowed bool
var dragDown, dragQualified, selectionBusy, sentenceShown bool
var dragStart, dragEnd point
var dragStartAt time.Time
var sentenceUntil time.Time
var selectionSequence uint64
var pendingSelection selectedSentence
var pendingSentenceTranslation finishedSentence
var tipMode, tipOriginal, tipVocabulary string
var sentenceMemory sync.Map     // in-memory only; private document sentences are NOT written to disk
var lastSelectedSentence string // in-memory only; no document text saved to disk

const (
	WM_SELECTION           = WM_APP + 14
	WM_SENTENCE_TRANSLATED = WM_APP + 15
	CMD_SENTENCE_TEST      = 214
	CMD_SENTENCE_ONLINE    = 215
	sentenceMaxBytes       = 14000 // cap request size to limit cost and latency
)

type selectedSentence struct {
	ID         uint64
	Start, End point
	Text       string
	Source     string
	Err        string
}
type finishedSentence struct {
	ID                     uint64
	English, Korean, Error string
	Provider               string
}

func settingPath() string { return filepath.Join(dirPath(), "sentence_online_consent_web_v24.txt") }
func loadSentenceConsent() {
	b, e := os.ReadFile(settingPath())
	sentenceOnlineAllowed = e == nil && strings.TrimSpace(string(b)) == "yes"
}
func setSentenceConsent(allow bool) {
	sentenceOnlineAllowed = allow
	b := "no\n"
	if allow {
		b = "yes\n"
	}
	_ = os.MkdirAll(dirPath(), 0755)
	_ = os.WriteFile(settingPath(), []byte(b), 0600)
}
func requestConsent() bool {
	r := callMessage("Enable two-way Gemini sentence translation?\n\nSelected English/Korean text and hovered English words may be sent to the Google Gemini API.\nDo not select confidential company documents, customer information or sensitive data.\n\nYes: Enable online translation\nNo: Disable online translation\n\nYou can change this setting from the system tray.", MB_YESNO|MB_ICONQUESTION)
	setSentenceConsent(r == IDYES)
	return r == IDYES
}
func buttonDown() bool { state, _, _ := pGetAsyncKeyState.Call(0x01); return state&0x8000 != 0 }
func dragEnough(a, b point) bool {
	dx := int64(a.X) - int64(b.X)
	dy := int64(a.Y) - int64(b.Y)
	return dx*dx+dy*dy >= 32*32
}

// Called on the main GUI thread from a 60 ms timer tick.
func observeDrag(pos point, now time.Time) bool {
	down := buttonDown()
	if down && !dragDown {
		// A deliberate click/drag starts a new interaction. Invalidate any
		// in-flight word OCR/Gemini result so it cannot replace a sentence.
		hoverGeneration++
		dragDown = true
		dragQualified = false
		dragStart = pos
		dragEnd = pos
		dragStartAt = now
		if sentenceShown {
			hide231Dictionary()
			sentenceShown = false
		}
		return true
	}
	if down && dragDown {
		dragEnd = pos
		if dragEnough(dragStart, pos) {
			dragQualified = true
		}
		return true
	}
	if !down && dragDown {
		dragDown = false
		end := pos
		enough := dragQualified && dragEnough(dragStart, end) && now.Sub(dragStartAt) > 100*time.Millisecond
		dragQualified = false
		if enough && !selectionBusy {
			selectionSequence++
			id := selectionSequence
			selectionBusy = true
			hide231Dictionary()
			lastMoved = now
			lastCheckAt = now
			from := dragStart
			fg, _, _ := pGetForegroundWindow.Call()
			go inspectSelection(id, from, end, fg)
			return true
		}
		lastMoved = now
		return true
	}
	if selectionBusy {
		return true
	}
	if sentenceShown {
		// Ignore pointer movement after a completed drag. The selected
		// sentence (including an asynchronous Gemini result) owns the popup
		// until timeout or the next deliberate mouse-button press.
		if now.Before(sentenceUntil) {
			return true
		}
		sentenceShown = false
		hide231Dictionary()
		lastMoved = now
		return true // Do not immediately fire word OCR on the same tick.
	}
	return false
}

func inspectSelection(id uint64, a, b point, foreground uintptr) {
	// v3.2: NEVER make a screenshot or OCR the selected sentence.
	// Ask the Windows accessibility selection provider for the selected text;
	// compatible PPT/PDF editors may also supply actual text via Ctrl+C.
	selection := selectedSentence{ID: id, Start: a, End: b}
	answer, err := ocrEngine.readSelection(b, "", foreground)
	if err != nil {
		selection.Err = "Selection text access error: " + err.Error()
	} else {
		selection.Text = normalizeSentence(answer.Sentence)
		selection.Source = answer.Backend
		selection.Err = answer.Error
	}
	uiMu.Lock()
	pendingSelection = selection
	uiMu.Unlock()
	pPostMessage.Call(mainHWND, WM_SELECTION, 0, 0)
}

func normalizeSentence(s string) string {
	// Preserve the user's explicit source rows instead of flattening them.
	return normalizeSelectedLines(s)
}
func mostlyEnglish(s string) bool {
	// Keep the legacy entry point while recognizing Korean drag selections.
	return selectedLanguageSupported(s)
}
func handleSelection() {
	if foregroundOrPointIsSuppressed(mouse()) {
		selectionBusy = false
		return
	}
	uiMu.Lock()
	s := pendingSelection
	uiMu.Unlock()
	if s.ID != selectionSequence {
		return
	}
	selectionBusy = false
	if !mostlyEnglish(s.Text) {
		appendDiagnostic(fmt.Sprintf("selection: no text, backend=%s err=%s", s.Source, s.Err))
		// No OCR fallback: refuse inaccurate approximate translation.
		detail := "Could not read selected text. In PowerPoint, select text in edit mode; in a PDF, use the text-selection tool."
		if s.Err != "" {
			detail = s.Err
		}
		showSentenceTip("", detail, "", s.End)
		return
	}
	lastSelectedSentence = s.Text
	appendDiagnostic(fmt.Sprintf("selected text via %s; length=%d", s.Source, len([]rune(s.Text))))
	meaning := "Translating…"
	if !sentenceOnlineAllowed {
		meaning = "Online translation is disabled."
	}
	if cached, ok := sentenceMemory.Load(s.Text); ok {
		meaning = cached.(string)
	}
	// Show glossary only alongside a completed translation (including cache hits).
	terms := ""
	if meaning != "Translating…" && sentenceOnlineAllowed {
		terms = selectionGlossary(s.Text, meaning)
	}
	showSentenceTip("", meaning, terms, s.End)
	if !sentenceOnlineAllowed || meaning != "Translating…" {
		return
	}
	go func(req selectedSentence) {
		meaning, provider, err := translateSentence(req.Text)
		response := finishedSentence{ID: req.ID, English: req.Text, Korean: meaning, Provider: provider}
		if err != nil {
			response.Error = err.Error()
		}
		uiMu.Lock()
		pendingSentenceTranslation = response
		uiMu.Unlock()
		pPostMessage.Call(mainHWND, WM_SENTENCE_TRANSLATED, 0, 0)
	}(s)
}
func handleSentenceTranslation() {
	if foregroundOrPointIsSuppressed(mouse()) {
		return
	}
	uiMu.Lock()
	t := pendingSentenceTranslation
	uiMu.Unlock()
	if t.ID != selectionSequence || !active {
		return
	}
	if t.Error != "" {
		appendDiagnostic("sentence API error: " + t.Error)
		msg := "Translation failed: " + t.Error
		if len([]rune(msg)) > 180 {
			msg = string([]rune(msg)[:180]) + "…"
		}
		showSentenceTip("", msg, "", dragEnd)
		return
	}
	sentenceMemory.Store(t.English, t.Korean)
	// Korean sentence first, followed by important English terms and idioms.
	// Do not display the original sentence, a model name, or a "문장 번역" label.
	showSentenceTip("", t.Korean, selectionGlossary(t.English, t.Korean), dragEnd)
}

// No text gets stored on disk and only text explicitly drag-selected by the user
// is sent to third-party translation after the user opted in at install.
func translateSentence(s string) (string, string, error) {
	if v, ok := sentenceMemory.Load(s); ok {
		return v.(string), "Cache", nil
	}
	s = normalizeSentence(s)
	if len([]byte(s)) > sentenceMaxBytes {
		return "", "", errors.New("The selected text is too long (maximum 14,000 bytes). Select shorter passages.")
	}
	if len(s) == 0 {
		return "", "", errors.New("Empty selection")
	}
	// Translate one source row per Gemini request, then reconstruct the
	// output using the *original row indexes*, including blank rows.
	// This prevents Gemini from merging separate source lines.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	translated, provider, err := translateSourceLines(ctx, s, translateGemini)
	if err != nil {
		return "", "", err
	}
	if directionForText(s) == koreanToEnglish {
		rememberCombinedKoreanGlossary(s, translated)
	}
	return translated, provider, nil
}
func showSentenceTip(original, translation, terms string, at point) {
	if !active || foregroundOrPointIsSuppressed(mouse()) {
		return
	}
	uiMu.Lock()
	tipMode = "sentence"
	tipOriginal = original
	tipMean = translation
	tipVocabulary = terms
	// Fit both translated Korean and the glossary; original English sentence is hidden.
	runeCount := len([]rune(translation))
	tipWidth = 210 + runeCount*8
	for _, line := range strings.Split(terms, "\n") {
		// Approximate the width of short bilingual glossary entries; wrap long lines.
		n := len([]rune(line))
		if n > 24 && tipWidth < 430 {
			tipWidth = 430
		}
	}
	if tipWidth > 600 {
		tipWidth = 600
	}
	if tipWidth < 210 {
		tipWidth = 210
	}
	// Fit smaller laptop displays while keeping long sentences visible.
	ww, _, _ := pGetSystemMetrics.Call(SM_CXVIRTUALSCREEN)
	hh, _, _ := pGetSystemMetrics.Call(SM_CYVIRTUALSCREEN)
	if int(ww) > 0 && tipWidth > int(ww)-20 {
		tipWidth = int(ww) - 20
	}
	if tipWidth < 260 {
		tipWidth = 260
	}
	tipHeight = sentenceHeight(original, translation, terms, tipWidth)
	if int(hh) > 0 && tipHeight > int(hh)-20 {
		tipHeight = int(hh) - 20
	}
	w, h := tipWidth, tipHeight
	uiMu.Unlock()
	// Under the selected text, but clamp to the virtual monitor bounds.
	x, y := int(at.X)-80, int(at.Y)+23
	left, _, _ := pGetSystemMetrics.Call(SM_XVIRTUALSCREEN)
	top, _, _ := pGetSystemMetrics.Call(SM_YVIRTUALSCREEN)
	minX, minY := int(int32(left)), int(int32(top))
	if x+w > minX+int(ww) {
		x = minX + int(ww) - w - 5
	}
	if y+h > minY+int(hh) {
		y = int(at.Y) - h - 25
	}
	if x < minX {
		x = minX + 4
	}
	if y < minY {
		y = minY + 4
	}
	pSetWindowPos.Call(tipHWND, HWND_TOPMOST, uintptr(x), uintptr(y), uintptr(w), uintptr(h), SWP_NOACTIVATE|SWP_SHOWWINDOW)
	pInvalidate.Call(tipHWND, 0, 1)
	tipShown = true
	sentenceShown = true
	sentenceUntil = time.Now().Add(32 * time.Second)
}

func sentenceHeight(orig, meaning, terms string, width int) int {
	hdc, _, _ := pGetDC.Call(0)
	if hdc == 0 {
		return 72
	}
	defer pReleaseDC.Call(0, hdc)
	// Reserve enough height for a separate vocabulary section below translation.
	total := 16 + measureWrappedHeight(hdc, fontMeaning, meaning, width-20, 320)
	if terms != "" {
		total += 13 // separator and spacing
		total += measureWrappedHeight(hdc, fontMeaning, "Key Words & Expressions", width-20, 30) + 4
		total += measureWrappedHeight(hdc, fontMeaning, terms, width-20, 220)
	}
	if total < 38 {
		total = 38
	}
	if total > 580 {
		total = 580
	}
	return total
}

func measureWrappedHeight(hdc, font uintptr, s string, width, maxHeight int) int {
	old, _, _ := pSelectObject.Call(hdc, font)
	defer pSelectObject.Call(hdc, old)
	r := rect{0, 0, int32(width), int32(maxHeight)}
	u := syscall.StringToUTF16(s)
	if len(u) < 2 {
		return fontPixelHeight + 5
	}
	h, _, _ := pDrawText.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), DT_WORDBREAK|DT_NOPREFIX|DT_CALCRECT)
	if h == 0 {
		return fontPixelHeight + 6
	}
	if int(h) > maxHeight {
		return maxHeight
	}
	return int(h)
}
func drawSentenceTip(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	uiMu.Lock()
	width, height := tipWidth, tipHeight
	translation, terms := tipMean, tipVocabulary
	uiMu.Unlock()
	r := rect{0, 0, int32(width), int32(height)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brushBG)
	pFrameRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brushBorder)
	pSetBkMode.Call(hdc, TRANSPARENT)
	pSetTextColor.Call(hdc, color(28, 90, 65))
	pSelectObject.Call(hdc, fontMeaning)
	y := int32(8)
	translationHeight := int32(measureWrappedHeight(hdc, fontMeaning, translation, width-20, 320))
	textRect := rect{9, y, int32(width) - 9, y + translationHeight}
	if textRect.Bottom > int32(height)-5 {
		textRect.Bottom = int32(height) - 5
	}
	if u := syscall.StringToUTF16(translation); len(u) > 1 {
		pDrawText.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&textRect)), DT_WORDBREAK|DT_NOPREFIX|DT_END_ELLIPSIS)
	}
	y += translationHeight + 4
	if terms != "" && y+30 < int32(height) {
		// Subtle divider ensures the Korean translation reads as the primary result.
		divider := rect{9, y + 1, int32(width) - 9, y + 2}
		pFillRect.Call(hdc, uintptr(unsafe.Pointer(&divider)), brushBorder)
		y += 8
		pSetTextColor.Call(hdc, color(74, 80, 90))
		heading := "Key Words & Expressions"
		headingRect := rect{9, y, int32(width) - 9, y + 22}
		if u := syscall.StringToUTF16(heading); len(u) > 1 {
			pDrawText.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&headingRect)), DT_WORDBREAK|DT_NOPREFIX)
		}
		y += int32(measureWrappedHeight(hdc, fontMeaning, heading, width-20, 30)) + 4
		termsRect := rect{9, y, int32(width) - 9, int32(height) - 5}
		if u := syscall.StringToUTF16(terms); len(u) > 1 {
			pDrawText.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&termsRect)), DT_WORDBREAK|DT_NOPREFIX|DT_END_ELLIPSIS)
		}
	}
	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}
