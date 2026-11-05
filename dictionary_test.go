package main

import (
	"os"
	"strings"
	"testing"
)

func TestBuiltInAndOffline(t *testing.T) {
	prepareDictionary()
	if len(builtinDict) < 1000 {
		t.Fatalf("glossary too small: %d", len(builtinDict))
	}
	examples := map[string]string{
		"Tachograph": "운행기록계", "Warranty": "품질보증", "Quality": "품질",
		"Software": "소프트웨어", "Update": "업데이트", "Chassis": "차대",
		"Leakage": "누수", "Powertrain": "동력계",
	}
	for k, v := range examples {
		got, ok := offlineLookup(k)
		if !ok || !strings.Contains(got.Meaning, v) {
			t.Errorf("%s offline=%v %q expected %q", k, ok, got.Meaning, v)
		}
		if got.Err != "" {
			t.Errorf("unexpected network error for %s: %s", k, got.Err)
		}
		if strings.Contains(got.Meaning, "인터넷") {
			t.Errorf("should not network fail %s", k)
		}
	}
	// translateDetailed intentionally bypasses the offline dictionary in v2.9.
	// Gemini word translation is separately tested with a mocked HTTP provider.
	if _, err := os.Stat(userDictionaryPath()); err != nil {
		t.Error("user dictionary not created", err)
	}
}
func TestCustomParsing(t *testing.T) {
	x := parseDictionary([]byte("#comment\nTachograph\t운행기록계\nBAD\nA\t\n"))
	if len(x) != 1 || x["tachograph"] != "운행기록계" {
		t.Fatal(x)
	}
}
