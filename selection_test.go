package main

import (
	"os"
	"strings"
	"testing"
)

func TestSentenceSelectionDoesNotCaptureImage(t *testing.T) {
	raw, err := os.ReadFile("sentence.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	start := strings.Index(s, "func inspectSelection(")
	end := strings.Index(s, "func normalizeSentence(")
	if start < 0 || end < start {
		t.Fatal("cannot locate sentence selection method")
	}
	function := s[start:end]
	if strings.Contains(function, "captureArea(") || strings.Contains(function, ".png") || strings.Contains(function, "Read-WindowsOCR") {
		t.Fatal("sentence still uses screenshot OCR")
	}
	if !strings.Contains(function, "readSelection(b, \"\", foreground)") {
		t.Fatal("sentence text pathway missing")
	}
	if strings.Contains(s, "showSentenceTip(t.English") {
		t.Fatal("sentence still displays English source")
	}
}

func TestSelectionWorkerUsesTextAndPreservesClipboard(t *testing.T) {
	raw, err := os.ReadFile("ocr_worker.ps1")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	start := strings.Index(s, "if ([string]$req.mode -eq 'selection') {")
	if start < 0 {
		t.Fatal("selection mode missing")
	}
	end := start + strings.Index(s[start:], "$cx = [double]$req.x")
	if end < start {
		t.Fatal("can't locate mode boundary")
	}
	selection := s[start:end]
	for _, snippet := range []string{"Read-SelectedText ", "Read-SelectedTextViaCopy", "GetForegroundWindow"} {
		if !strings.Contains(selection, snippet) {
			t.Errorf("selection missing %s", snippet)
		}
	}
	if strings.Contains(selection, "Read-WindowsOCRText ") || strings.Contains(selection, "Read-TesseractText ") {
		t.Fatal("OCR fallback still active")
	}
	if !strings.Contains(s, "OleGetClipboard") || !strings.Contains(s, "OleSetClipboard") {
		t.Fatal("clipboard preservation not implemented")
	}
}
