package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseKoreanTranslationJSONAndGlossary(t *testing.T) {
	raw := `{"translation":"The electric pump cannot generate hydraulic pressure.","terms":[{"english":"electric pump","meaning_ko":"전동 펌프"},{"english":"hydraulic pressure","meaning_ko":"유압"},{"english":"invalid","meaning_ko":"임의 내용"}]}`
	translation, glossary, err := parseKoreanTranslationAndTerms(raw)
	if err != nil || !strings.Contains(translation, "hydraulic pressure") || !strings.Contains(glossary, "electric pump  ·  전동 펌프") || strings.Contains(glossary, "invalid") {
		t.Fatalf("translation %q glossary %q error %v", translation, glossary, err)
	}
	if got, _, err := parseKoreanTranslationAndTerms("```json\n" + raw + "\n```"); err != nil || !strings.Contains(got, "electric pump") {
		t.Fatalf("code fence parse %q %v", got, err)
	}
	if tr, gl, err := parseKoreanTranslationAndTerms("The cooling system has failed."); err != nil || tr == "" || gl != "" {
		t.Fatalf("plain text fallback %q %q %v", tr, gl, err)
	}
}

func TestGeminiKoreanWordAndSentencePrompts(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	seenSentence, seenWord := false, false
	webRequester = func(_ context.Context, method, endpoint string, h map[string]string, body []byte) (webResponse, error) {
		if h["x-goog-api-key"] != "TEST_KEY_ONLY" {
			t.Fatal("missing api key")
		}
		if method == "GET" {
			return webResponse{Status: 200, Body: []byte(testModelList)}, nil
		}
		var b struct {
			SystemInstruction struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"systemInstruction"`
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if err := json.Unmarshal(body, &b); err != nil {
			t.Fatal(err)
		}
		if len(b.Contents) == 0 || len(b.Contents[0].Parts) == 0 {
			t.Fatalf("missing source")
		}
		source := b.Contents[0].Parts[0].Text
		prompt := b.SystemInstruction.Parts[0].Text
		if strings.Contains(source, "유압을") {
			seenSentence = true
			if !strings.Contains(prompt, "Korean-to-English") || !strings.Contains(source, "유압을") {
				t.Fatalf("wrong korean sentence prompt %q", prompt)
			}
			result := `{"translation":"The electric pump cannot generate hydraulic pressure.","terms":[{"english":"electric pump","meaning_ko":"전동 펌프"},{"english":"hydraulic pressure","meaning_ko":"유압"}]}`
			data, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": result}}}}}})
			return webResponse{Status: 200, Body: data}, nil
		}
		seenWord = true
		if !strings.Contains(source, "품질보증") || !strings.Contains(prompt, "Korean word") {
			t.Fatalf("wrong korean hover prompt %q / %q", prompt, source)
		}
		return webResponse{Status: 200, Body: []byte(`{"candidates":[{"content":{"parts":[{"text":"quality assurance"}]}}]}`)}, nil
	}
	sentence := "전동 펌프가 유압을 형성하지 못합니다."
	tr, _, err := translateGemini(context.Background(), sentence)
	if err != nil || !strings.Contains(tr, "hydraulic pressure") {
		t.Fatalf("sentence %q: %v", tr, err)
	}
	glossary := selectionGlossary(sentence, tr)
	if !strings.Contains(glossary, "hydraulic pressure") || !strings.Contains(glossary, "유압") {
		t.Fatalf("glossary %q", glossary)
	}
	word, _, err := translateGeminiWord(context.Background(), "품질보증")
	if err != nil || word != "quality assurance" || !seenSentence || !seenWord {
		t.Fatalf("word %q err %v sentence %v word %v", word, err, seenSentence, seenWord)
	}
}

func TestLegacyEnglishGlossaryRemains(t *testing.T) {
	sentence := "The electric pump fails to build hydraulic pressure."
	glossary := selectionGlossary(sentence, "전동 펌프가 유압을 형성하지 못합니다.")
	if !strings.Contains(glossary, "hydraulic pressure") || !strings.Contains(glossary, "유압") {
		t.Fatalf("legacy english glossary %q", glossary)
	}
}
