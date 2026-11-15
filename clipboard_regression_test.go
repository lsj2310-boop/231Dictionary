package main

import (
	"os"
	"strings"
	"testing"
)

func TestPowerShellClipboardFallbackRejectsStaleRestore(t *testing.T) {
	data, err := os.ReadFile("ocr_worker.ps1")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	required := []string{
		"Local\\231DictionaryClipboardTransaction",
		"[SelectionWin32]::InputHeld()",
		"[SelectionWin32]::GetClipboardSequenceNumber() -ne $before",
		"if ($current -eq $copySeq)",
		"[SelectionWin32]::OleSetClipboard($old)",
		"Skipped clipboard restore because another copy occurred.",
	}
	for _, token := range required {
		if !strings.Contains(s, token) {
			t.Errorf("clipboard safeguard missing: %q", token)
		}
	}
	restore := strings.Index(s, "[SelectionWin32]::OleSetClipboard($old)")
	guard := strings.Index(s, "if ($current -eq $copySeq)")
	if restore < guard {
		t.Fatal("unconditional clipboard restore returned")
	}
}

func TestPopupCopyCoordinatedAndNonblocking(t *testing.T) {
	data, err := os.ReadFile("popup_interaction_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, token := range []string{
		"Local\\231DictionaryClipboardTransaction",
		"func copyPopupTranslationNow(text string)",
		"withClipboardTransaction(func() error",
		"copyPopupTranslationNow(currentTranslationForCopy())",
		"case WM_RBUTTONDOWN:",
		"pSetCapture.Call(tipHWND)",
		"for i := 0; i < 30; i++",
		"copyPopupTranslationNow(currentTranslationForCopy())",
	} {
		if !strings.Contains(s, token) {
			t.Errorf("popup clipboard safeguard missing: %q", token)
		}
	}
}

func TestPopupClipboardWriteIsSynchronous(t *testing.T) {
	data, err := os.ReadFile("popup_interaction_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	from := strings.Index(s, "func copyPopupTranslationNow(text string)")
	to := strings.Index(s, "func copyUnicodeTextToClipboard(")
	if from < 0 || to <= from {
		t.Fatal("copy implementation not found")
	}
	implementation := s[from:to]
	if strings.Contains(implementation, "go func") {
		t.Fatal("right-click copy is still asynchronous")
	}
	if !strings.Contains(implementation, "withClipboardTransaction(func() error") {
		t.Fatal("copy lacks transaction")
	}
}
