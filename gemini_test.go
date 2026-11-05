package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testModelList = `{"models":[{"name":"models/gemini-2.5-flash-lite","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-3.8-flash","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-3.5-flash-lite","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-embedding-001","supportedGenerationMethods":["embedContent"]},{"name":"models/gemini-3.5-flash-lite-tts","supportedGenerationMethods":["generateContent"]}]}`
const testGeminiSuccess = `{"candidates":[{"content":{"parts":[{"text":"수동 펌프는 정상 작동하지만 전동 펌프가 유압을 형성하지 못합니다."}]}}]}`

func resetGeminiDiscovery() {
	geminiDiscoveryCache.Lock()
	geminiDiscoveryCache.keyHash = [32]byte{}
	geminiDiscoveryCache.models = nil
	geminiDiscoveryCache.expires = time.Time{}
	geminiDiscoveryCache.Unlock()
}
func TestGeminiResponseSegments(t *testing.T) {
	got, err := parseGeminiResponse([]byte(`{"candidates":[{"content":{"parts":[{"text":"캡 틸트 시스템이 "},{"text":"작동하지 않습니다."}]}}]}`))
	if err != nil || got != "캡 틸트 시스템이 작동하지 않습니다." {
		t.Fatalf("segments: %q %v", got, err)
	}
}
func TestGeminiDiscoverEligibleModels(t *testing.T) {
	resetGeminiDiscovery()
	old := webRequester
	defer func() { webRequester = old; resetGeminiDiscovery() }()
	count := 0
	webRequester = func(_ context.Context, method, endpoint string, h map[string]string, _ []byte) (webResponse, error) {
		count++
		if method != "GET" || endpoint != geminiModelListURL || h["x-goog-api-key"] != "TEST_KEY_ONLY" {
			t.Fatalf("bad list %s %s", method, endpoint)
		}
		return webResponse{Status: 200, Body: []byte(testModelList)}, nil
	}
	models, err := discoverGeminiModels(context.Background(), "TEST_KEY_ONLY")
	if err != nil || len(models) != 3 || models[0] != "gemini-3.5-flash-lite" || models[1] != "gemini-3.8-flash" {
		t.Fatalf("models=%v err=%v", models, err)
	}
	_, err = discoverGeminiModels(context.Background(), "TEST_KEY_ONLY")
	if err != nil || count != 1 {
		t.Fatalf("cache failed: %d %v", count, err)
	}
}
func TestGeminiWholeTextAndAuthHeader(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	src := "Cab tilt system malfunction. Hand pump works; electric pump fails to build hydraulic pressure."
	requests := 0
	webRequester = func(_ context.Context, method, endpoint string, h map[string]string, body []byte) (webResponse, error) {
		if h["x-goog-api-key"] != "TEST_KEY_ONLY" {
			t.Fatal("missing key header")
		}
		if method == "GET" {
			return webResponse{Status: 200, Body: []byte(testModelList)}, nil
		}
		requests++
		if endpoint != geminiAPIBase+"gemini-3.5-flash-lite:generateContent" {
			t.Fatalf("wrong model: %s", endpoint)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		parts := payload["contents"].([]any)[0].(map[string]any)["parts"].([]any)
		if !strings.Contains(parts[0].(map[string]any)["text"].(string), src) {
			t.Fatal("source sentence altered")
		}
		return webResponse{Status: http.StatusOK, Body: []byte(testGeminiSuccess)}, nil
	}
	result, provider, err := translateGemini(context.Background(), src)
	if err != nil || !strings.Contains(result, "수동 펌프") || provider != "Gemini (gemini-3.5-flash-lite)" || requests != 1 {
		t.Fatalf("result %q provider %q error %v calls %d", result, provider, err, requests)
	}
}
func TestGemini404FallsBackWithinAccountModels(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	tries := []string{}
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		if method == "GET" {
			return webResponse{Status: 200, Body: []byte(testModelList)}, nil
		}
		tries = append(tries, endpoint)
		if strings.Contains(endpoint, "3.5-flash-lite:") {
			return webResponse{Status: 404, Body: []byte(`{"error":{"code":404,"message":"not available"}}`)}, nil
		}
		if strings.Contains(endpoint, "3.8-flash:") {
			return webResponse{Status: 200, Body: []byte(testGeminiSuccess)}, nil
		}
		t.Fatalf("unexpected model %s", endpoint)
		return webResponse{}, nil
	}
	text, model, err := translateGemini(context.Background(), "Hand pump works; electric pump fails to build hydraulic pressure.")
	if err != nil || len(tries) != 2 || !strings.Contains(model, "3.8-flash") || text == "" {
		t.Fatalf("fallback %q %q %v %v", text, model, err, tries)
	}
}
func TestGemini503RetriesAndFallback(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK, oldPause := webRequester, geminiKeyReader, geminiRetryPause
	defer func() {
		webRequester = oldR
		geminiKeyReader = oldK
		geminiRetryPause = oldPause
		resetGeminiDiscovery()
	}()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	geminiRetryPause = func(_ context.Context, _ time.Duration) error { return nil }
	first, second := 0, 0
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		if method == "GET" {
			return webResponse{Status: 200, Body: []byte(testModelList)}, nil
		}
		if strings.Contains(endpoint, "3.5-flash-lite:") {
			first++
			return webResponse{Status: 503, Body: []byte(`{"error":{"code":503,"message":"high demand"}}`)}, nil
		}
		if strings.Contains(endpoint, "3.8-flash:") {
			second++
			return webResponse{Status: 200, Body: []byte(testGeminiSuccess)}, nil
		}
		t.Fatalf("unexpected model %s", endpoint)
		return webResponse{}, nil
	}
	result, _, err := translateGemini(context.Background(), "Cab tilt system malfunction.")
	if err != nil || result == "" || first != 2 || second != 1 {
		t.Fatalf("503 fallback %q %v counts=%d,%d", result, err, first, second)
	}
}
func TestGemini401AndInvalidKey(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	calls := 0
	webRequester = func(_ context.Context, _, _ string, _ map[string]string, _ []byte) (webResponse, error) {
		calls++
		return webResponse{Status: 401, Body: []byte(`{"error":{"code":401,"message":"Invalid key"}}`)}, nil
	}
	_, _, err := translateGemini(context.Background(), "Root cause analysis required.")
	if err == nil || !strings.Contains(err.Error(), "401") || calls != 1 {
		t.Fatalf("401 should stop: %v %d", err, calls)
	}
	geminiKeyReader = func() (string, error) { return "", nil }
	_, _, err = translateGemini(context.Background(), "Root cause analysis required.")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("missing key: %v", err)
	}
}
func TestGeminiModelsPaginationAndNoFallbackToUnavailable(t *testing.T) {
	resetGeminiDiscovery()
	old := webRequester
	defer func() { webRequester = old; resetGeminiDiscovery() }()
	calls := 0
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		calls++
		if method != "GET" {
			t.Fatal("unexpected POST")
		}
		if calls == 1 {
			return webResponse{Status: 200, Body: []byte(`{"models":[{"name":"models/gemini-2.5-flash-lite","supportedGenerationMethods":["embedContent"]}],"nextPageToken":"NEXT"}`)}, nil
		}
		if !strings.Contains(endpoint, "pageToken=NEXT") {
			t.Fatalf("missing page token: %s", endpoint)
		}
		return webResponse{Status: 200, Body: []byte(`{"models":[{"name":"models/gemini-3.6-flash","supportedGenerationMethods":["generateContent"]}]}`)}, nil
	}
	available, err := discoverGeminiModels(context.Background(), "TEST_KEY_ONLY")
	if err != nil || len(available) != 1 || available[0] != "gemini-3.6-flash" {
		t.Fatalf("pagination %v %v", available, err)
	}
}
func TestGeminiNetworkFailure(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	webRequester = func(_ context.Context, _, _ string, _ map[string]string, _ []byte) (webResponse, error) {
		return webResponse{}, errors.New("no route to host")
	}
	_, _, err := translateGemini(context.Background(), "An issue occurred.")
	if err == nil || !strings.Contains(err.Error(), "Could not connect") {
		t.Fatalf("network: %v", err)
	}
}
func TestGeminiWordTranslationOnlyKorean(t *testing.T) {
	resetGeminiDiscovery()
	oldR, oldK := webRequester, geminiKeyReader
	defer func() { webRequester = oldR; geminiKeyReader = oldK; resetGeminiDiscovery() }()
	geminiKeyReader = func() (string, error) { return "TEST_KEY_ONLY", nil }
	webRequester = func(_ context.Context, method, _ string, _ map[string]string, body []byte) (webResponse, error) {
		if method == "GET" {
			return webResponse{Status: 200, Body: []byte(testModelList)}, nil
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		prompt := req["systemInstruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"].(string)
		if !strings.Contains(prompt, "ONE English technical word") {
			t.Fatal("incorrect word prompt")
		}
		return webResponse{Status: 200, Body: []byte(`{"candidates":[{"content":{"parts":[{"text":"냉각 장치"}]}}]}`)}, nil
	}
	value, _, err := translateGeminiWord(context.Background(), "Condenser")
	if err != nil || value != "냉각 장치" {
		t.Fatalf("word %s %v", value, err)
	}
}
