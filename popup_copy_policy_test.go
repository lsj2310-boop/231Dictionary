package main

import "testing"

func TestCopyTranslatedBodyInBothDirectionsWithoutSourceOrGlossary(t *testing.T) {
	for _, tc := range []struct{ source, translated, terms string }{
		{"전동 펌프가 충분한 유압을 형성하지 못합니다.", "The electric pump cannot generate sufficient hydraulic pressure.", "electric pump — 전동 펌프\nhydraulic pressure — 유압"},
		{"The warranty period was extended.", "보증기간이 연장되었습니다.", "warranty period — 보증기간"},
		{"개선이 완료되었습니다.\n추가 확인이 필요합니다.", "The improvement has been completed.\nAdditional verification is required.", "improvement — 개선"},
	} {
		got := eligibleTranslationCopy(tc.source, tc.translated, true)
		if got != tc.translated {
			t.Fatalf("want translated body %q, got %q", tc.translated, got)
		}
		if got == tc.source || got == tc.terms {
			t.Fatalf("copied source or glossary: %q", got)
		}
		if got := displayedTranslationForCopy(tc.translated, got, true, true, "sentence"); got != tc.translated {
			t.Fatalf("right click copies something other than visible translation: %q", got)
		}
	}
}
func TestCopyOnlyWhenCompletedVisibleAndMatching(t *testing.T) {
	source := "The warranty period was extended."
	result := "보증기간이 연장되었습니다."
	for _, tc := range []struct {
		source, result string
		completed      bool
	}{
		{source, "Translating…", false},
		{source, "Translation failed: timeout", true},
		{source, "Online translation is disabled.", true},
		{source, "Could not read selected text.", true},
		{"", result, true},
	} {
		if got := eligibleTranslationCopy(tc.source, tc.result, tc.completed); got != "" {
			t.Fatalf("unexpected copy for pending/error case: %q", got)
		}
	}
	eligible := eligibleTranslationCopy(source, result, true)
	for _, tc := range []struct {
		visible         string
		shown, sentence bool
		mode            string
	}{
		{result, false, true, "sentence"},
		{result, true, false, "sentence"},
		{result, true, true, "word"},
		{"The warranty period was extended.", true, true, "sentence"},
		{"Another translation", true, true, "sentence"},
	} {
		if got := displayedTranslationForCopy(tc.visible, eligible, tc.shown, tc.sentence, tc.mode); got != "" {
			t.Fatalf("incorrectly copied stale or inactive translation: %q", got)
		}
	}
}
