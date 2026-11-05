package main

import (
	"os"
	"strings"
	"testing"
)

func TestTranslatedSentenceHasGlossaryAndNoEnglishParagraph(t *testing.T) {
	b, e := os.ReadFile("sentence.go")
	if e != nil {
		t.Fatal(e)
	}
	src := string(b)
	i := strings.Index(src, "func handleSentenceTranslation()")
	j := strings.Index(src[i:], "func translateSentence(")
	if i < 0 || j < 0 {
		t.Fatal("sentence translation handler missing")
	}
	handler := src[i : i+j]
	if !strings.Contains(handler, "selectionGlossary(t.English, t.Korean)") || !strings.Contains(handler, `showSentenceTip("", t.Korean,`) {
		t.Fatal("translated sentence does not pass glossary to popup")
	}
	if strings.Contains(handler, `showSentenceTip(t.English,`) || strings.Contains(handler, "t.Provider") {
		t.Fatal("English paragraph or model label visible")
	}
	if !strings.Contains(src, `heading := "Key Words & Expressions"`) || !strings.Contains(src, "tipVocabulary = terms") {
		t.Fatal("glossary section not rendered")
	}
}
