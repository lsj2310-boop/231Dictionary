package main

import (
	"context"
	"strings"
	"testing"
)

func TestV331KoreanRowsAndBlankRows(t *testing.T) {
	input := "전동 펌프가 작동하지 않습니다.\r\n\r\n유압이 부족합니다."
	expectedSrc := "전동 펌프가 작동하지 않습니다.\n\n유압이 부족합니다."
	if got := normalizeSelectedLines(input); got != expectedSrc {
		t.Fatalf("normalize %q", got)
	}
	translated, _, err := translateSourceLines(context.Background(), input, func(_ context.Context, row string) (string, string, error) {
		switch row {
		case "전동 펌프가 작동하지 않습니다.":
			return "The electric pump is not working.", "Gemini", nil
		case "유압이 부족합니다.":
			return "Hydraulic pressure is insufficient.", "Gemini", nil
		}
		t.Fatalf("unexpected row %q", row)
		return "", "", nil
	})
	want := "The electric pump is not working.\n\nHydraulic pressure is insufficient."
	if err != nil || translated != want {
		t.Fatalf("got %q err %v", translated, err)
	}
}

func TestV331CombinedContextualGlossary(t *testing.T) {
	source := "전동 펌프가 작동하지 않습니다.\n유압이 부족합니다."
	koreanSentenceGlossary.Store("전동 펌프가 작동하지 않습니다.", "electric pump  ·  전동 펌프")
	koreanSentenceGlossary.Store("유압이 부족합니다.", "hydraulic pressure  ·  유압")
	translated := "The electric pump is not working.\nHydraulic pressure is insufficient."
	rememberCombinedKoreanGlossary(source, translated)
	glossary := selectionGlossary(source, translated)
	for _, term := range []string{"electric pump  ·  전동 펌프", "hydraulic pressure  ·  유압"} {
		if !strings.Contains(glossary, term) {
			t.Fatalf("missing %q in %q", term, glossary)
		}
	}
}

func TestV331EnglishGlossaryRegression(t *testing.T) {
	source := "The electric pump is faulty.\nHydraulic pressure is insufficient."
	glossary := selectionGlossary(source, "전동 펌프가 고장났습니다.\n유압이 부족합니다.")
	if !strings.Contains(glossary, "hydraulic pressure") {
		t.Fatalf("english glossary missing: %q", glossary)
	}
}
