package main

import (
	"strings"
	"testing"
)

func TestKeyTermsAndIdioms(t *testing.T) {
	prepareDictionary()
	got := extractKeyTerms("Root cause analysis and corrective action are required before the software update.", 6)
	for _, expect := range []string{"root cause analysis", "근본 원인 분석", "corrective action", "시정 조치", "software update", "소프트웨어 업데이트"} {
		if !strings.Contains(got, expect) {
			t.Fatalf("missing %q in %q", expect, got)
		}
	}
}
func TestUserSentenceTerms(t *testing.T) {
	got := extractKeyTerms("The warranty claim was made due to water leakage.", 6)
	for _, expect := range []string{"warranty claim", "water leakage", "due to"} {
		if !strings.Contains(got, expect) {
			t.Errorf("missing %q: %s", expect, got)
		}
	}
	if len(strings.Split(got, "\n")) > 6 {
		t.Errorf("more than six terms: %s", got)
	}
}
func TestNoTerms(t *testing.T) {
	if extractKeyTerms("The a or an", 5) != "" {
		t.Fatal("stopwords should not be shown")
	}
}

func TestAutomotiveContextGlossary(t *testing.T) {
	prepareDictionary()
	got := extractKeyTerms("Cab tilt system malfunction. Hand pump works; electric pump fails to build hydraulic pressure.", 5)
	for _, want := range []string{"cab tilt system", "캡 틸트 시스템", "hand pump", "수동 펌프", "electric pump", "전동 펌프", "hydraulic pressure", "유압"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in glossary: %s", want, got)
		}
	}
	if len(strings.Split(got, "\n")) > 5 {
		t.Fatalf("too many terms: %s", got)
	}
}

func TestSpecificPhraseNotRepeated(t *testing.T) {
	prepareDictionary()
	got := extractKeyTerms("Root cause analysis was performed.", 5)
	if strings.Contains(got, "root cause  ·") {
		t.Fatalf("nested phrase duplicated: %s", got)
	}
	if !strings.Contains(got, "root cause analysis") {
		t.Fatalf("missing specific phrase: %s", got)
	}
}
