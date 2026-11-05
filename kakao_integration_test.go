package main

import (
	"os"
	"strings"
	"testing"
)

func TestKakaoTalkSuppressionUsedAtAllTranslationEntrypoints(t *testing.T) {
	for _, filename := range []string{"main.go", "sentence.go"} {
		raw, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		source := string(raw)
		if strings.Contains(source, "foregroundOrPointIsBrowser(") {
			t.Errorf("%s still calls browser-only filter", filename)
		}
		if !strings.Contains(source, "foregroundOrPointIsSuppressed(") {
			t.Errorf("%s does not use all-app suppression", filename)
		}
	}
	data, err := os.ReadFile("browser_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if !strings.Contains(source, "if isKakaoTalkExecutable(name) {") {
		t.Fatal("KakaoTalk process-name exclusion missing")
	}
	if !strings.Contains(source, "if browserExclusion && isBrowserExecutable(name)") {
		t.Fatal("PDF-aware browser preference should be independent from KakaoTalk")
	}
}
