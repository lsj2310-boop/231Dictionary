package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestBrowserPDFWindowTitle(t *testing.T) {
	positives := []string{
		"Report.pdf - Google Chrome", "Quality_Issues.PDF - Microsoft Edge",
		"meeting.pdf - NAVER Whale", "filename.pdf (1) - Brave", "C:\\docs\\manual.pdf — Aside",
	}
	for _, title := range positives {
		if !isPDFBrowserTitle(title) {
			t.Errorf("PDF tab wrongly excluded: %q", title)
		}
	}
	negatives := []string{"How to read PDF files - Chrome", "PDF converter - Chrome", "Quality issues - Google Chrome", "Spreadsheet.xlsx - Chrome"}
	for _, title := range negatives {
		if isPDFBrowserTitle(title) {
			t.Errorf("ordinary tab wrongly allowed: %q", title)
		}
	}
}
func TestBrowserPDFAddressPolicy(t *testing.T) {
	positives := []string{
		"https://example.org/report.pdf", "https://example.org/report.PDF?token=123",
		"file:///C:/Users/A/Documents/report.pdf", "https://example.org/view?file=report.pdf",
		"https://example.org/view?url=https%3A%2F%2Fexample.org%2Fsafe.pdf",
	}
	negatives := []string{
		"https://example.org/search?q=report.pdf", "https://example.org/pdf-tools",
		"https://example.org/articles/how-to-read-pdf", "https://example.org/report.pdf.html",
	}
	for _, u := range positives {
		if !isPDFDocumentURL(u) {
			t.Errorf("PDF address wrongly excluded: %s", u)
		}
	}
	for _, u := range negatives {
		if isPDFDocumentURL(u) {
			t.Errorf("non-PDF address wrongly allowed: %s", u)
		}
	}
}
func TestDragTranslationOwnsPopup(t *testing.T) {
	for _, tc := range []struct{ drag, selectBusy, shown, want bool }{
		{true, false, false, true}, {false, true, false, true}, {false, false, true, true}, {false, false, false, false},
	} {
		if got := sentencePriorityActive(tc.drag, tc.selectBusy, tc.shown); got != tc.want {
			t.Errorf("priority %#v got %v", tc, got)
		}
	}
	now := time.Now()
	if !keepSentenceVisible(now, now.Add(25*time.Second)) {
		t.Error("sentence vanished during pointer move")
	}
	if keepSentenceVisible(now, now.Add(-time.Second)) {
		t.Error("sentence timeout not respected")
	}
}
func TestBrowserPDFProbeIncludedAndUsed(t *testing.T) {
	main, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("browser_pdf_probe.ps1")
	if err != nil {
		t.Fatal(err)
	}
	src := string(main)
	if !strings.Contains(src, "//go:embed ocr_worker.ps1 test_ocr.png web_proxy.ps1 browser_pdf_probe.ps1") || !strings.Contains(src, "browserPDFProbeScriptPath()") {
		t.Fatal("PDF accessibility probe not packed in installer")
	}
	if !strings.Contains(string(script), "AutomationElement") || !strings.Contains(string(script), ".pdf$") {
		t.Fatal("PDF probe does not inspect accessibility metadata")
	}
}
