package main

import (
	"os"
	"strings"
	"testing"
)

func TestCopyOnlyTranslatedEnglishWithoutGlossary(t *testing.T) {
	src := "전동 펌프가 충분한 유압을 형성하지 못합니다."
	translated := "The electric pump cannot generate sufficient hydraulic pressure."
	glossary := "electric pump — 전동 펌프\nhydraulic pressure — 유압"
	got := eligibleEnglishCopy(src, translated, true)
	if got != translated {
		t.Fatalf("translation content mismatch: %q", got)
	}
	if strings.Contains(got, glossary) || strings.Contains(got, "유압") {
		t.Fatalf("glossary/source leaked into copy: %q", got)
	}
}
func TestEnglishCopyNotAllowedForEnglishSourceOrPendingTranslation(t *testing.T) {
	for _, tc := range []struct {
		src, result string
		ready       bool
	}{
		{"The warranty period was extended.", "보증기간이 연장되었습니다.", true},
		{"보증기간이 연장되었습니다.", "Translating…", false},
		{"보증기간이 연장되었습니다.", "Translation failed: timeout", true},
		{"보증기간이 연장되었습니다.", "The warranty period was extended.", false},
	} {
		if r := eligibleEnglishCopy(tc.src, tc.result, tc.ready); r != "" {
			t.Fatalf("unexpected clipboard content: %q", r)
		}
	}
}
func TestPDFCraftSelectionFallbackAllowed(t *testing.T) {
	source, err := os.ReadFile("ocr_worker.ps1")
	if err != nil {
		t.Fatal(err)
	}
	s := string(source)
	for _, part := range []string{"'pdfcraft'", "'pdfcraftpro'", "$isPdfCraft", "Read-SelectedTextViaCopy", "Selected text copy is unavailable"} {
		if !strings.Contains(s, part) {
			t.Fatalf("missing PDF Craft support: %q", part)
		}
	}
}
