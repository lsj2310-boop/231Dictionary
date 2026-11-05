package main

import "testing"

func TestDirectionSelection(t *testing.T) {
	cases := []struct {
		in   string
		want translationDirection
	}{
		{"품질보증", koreanToEnglish},
		{"브레이크 ECU 점검을 진행했습니다.", koreanToEnglish},
		{"손상된 부품을 교체하십시오.", koreanToEnglish},
		{"Customer quality assurance and warranty.", englishToKorean},
		{"engine", englishToKorean},
		{"Cab tilt system malfunction. 조치 완료", englishToKorean},
	}
	for _, c := range cases {
		if got := directionForText(c.in); got != c.want {
			t.Errorf("%q: got %s want %s", c.in, got, c.want)
		}
	}
}
func TestHoverWordLanguage(t *testing.T) {
	for _, w := range []string{"품질보증", "누수", "고장", "Quality", "Warranty"} {
		if !isValidHoverWord(w) {
			t.Errorf("expected valid %q", w)
		}
	}
	for _, w := range []string{"두 단어", "two words", "@@", "A/C", ""} {
		if isValidHoverWord(w) {
			t.Errorf("expected invalid %q", w)
		}
	}
}
func TestSelectedKoreanAndEnglish(t *testing.T) {
	for _, s := range []string{"품질보증 조건을 검토합니다.", "선택사항", "The electric pump does not build hydraulic pressure."} {
		if !selectedLanguageSupported(s) {
			t.Fatalf("should accept %q", s)
		}
	}
	if selectedLanguageSupported("-") {
		t.Fatal("invalid selection")
	}
}
