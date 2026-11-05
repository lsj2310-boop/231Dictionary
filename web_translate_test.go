package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGoogleMTTJoinSegments(t *testing.T) {
	translated, err := parseGoogleResponse([]byte(`{"sentences":[{"trans":"캡 틸트 시스템 작동 불량. "},{"trans":"수동 펌프는 작동합니다."}],"src":"en"}`))
	if err != nil || translated != "캡 틸트 시스템 작동 불량. 수동 펌프는 작동합니다." {
		t.Fatalf("Google join: %q %v", translated, err)
	}
}
func TestGoogleLegacyResponse(t *testing.T) {
	translated, err := parseGoogleResponse([]byte(`[[["품질 ","quality"],["보증","warranty"]],null,"en"]`))
	if err != nil || translated != "품질 보증" {
		t.Fatalf("Google legacy: %q %v", translated, err)
	}
}
func TestBingResponse(t *testing.T) {
	text, err := parseBingResponse([]byte(`[{"detectedLanguage":{"language":"en"},"translations":[{"text":"전동 펌프가 유압을 형성하지 못합니다.","to":"ko"}]}]`))
	if err != nil || text != "전동 펌프가 유압을 형성하지 못합니다." {
		t.Fatalf("Bing response: %q %v", text, err)
	}
}
func TestBingToken(t *testing.T) {
	cred, err := parseBingCred([]byte(`<html>IG:"E4C2A12E" <span data-iid="translator.5021"> params_AbusePreventionHelper = [1762200000000,"abcd1234",3600000]</html>`))
	if err != nil || cred.Token != "abcd1234" || cred.Key != "1762200000000" || cred.IG != "E4C2A12E" {
		t.Fatalf("Bing token: %+v %v", cred, err)
	}
}
func TestMTTGoogleFirstWithFullText(t *testing.T) {
	old := webRequester
	defer func() { webRequester = old }()
	src := "Cab tilt system malfunction. Hand pump works; electric pump fails to build hydraulic pressure."
	n := 0
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		n++
		if method != "GET" || !strings.HasPrefix(endpoint, googleGTXURL) {
			t.Fatalf("unexpected req %s %s", method, endpoint)
		}
		u, _ := url.Parse(endpoint)
		if u.Query().Get("q") != src || u.Query().Get("tl") != "ko" {
			t.Fatalf("bad English text: %v", u.Query())
		}
		return webResponse{Status: http.StatusOK, Body: []byte(`{"sentences":[{"trans":"캡 틸트 시스템이 고장났습니다."}]}`)}, nil
	}
	translated, source, e := translateWebFallback(context.Background(), src)
	if e != nil || translated != "캡 틸트 시스템이 고장났습니다." || source != "Google" || n != 1 {
		t.Fatalf("got: %q %q %v %d", translated, source, e, n)
	}
}
func TestMTTGoogleAlternateHost(t *testing.T) {
	old := webRequester
	defer func() { webRequester = old }()
	seenPrimary := false
	seenAlt := false
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		switch {
		case strings.HasPrefix(endpoint, googleGTXURL+"?"):
			seenPrimary = true
			return webResponse{Status: 403}, nil
		case strings.HasPrefix(endpoint, googleGTXURLAlt+"?"):
			seenAlt = true
			return webResponse{Status: 200, Body: []byte(`{"sentences":[{"trans":"품질보증이 필요합니다."}]}`)}, nil
		}
		return webResponse{}, errors.New("unknown URL")
	}
	got, provider, err := translateWebFallback(context.Background(), "Quality assurance is required.")
	if err != nil || got != "품질보증이 필요합니다." || provider != "Google" || !seenPrimary || !seenAlt {
		t.Fatalf("alternate host fallback: %q %s %v, seen primary=%v alt=%v", got, provider, err, seenPrimary, seenAlt)
	}
}
func TestMTTBothFail(t *testing.T) {
	old := webRequester
	defer func() { webRequester = old }()
	webRequester = func(_ context.Context, method, endpoint string, _ map[string]string, _ []byte) (webResponse, error) {
		return webResponse{Status: 503}, nil
	}
	got, provider, err := translateWebFallback(context.Background(), "Cab tilt system malfunction.")
	if err == nil || got != "" || provider != "" || !strings.Contains(err.Error(), "Google translation failed") {
		t.Fatalf("failure should be diagnostic: %q %q %v", got, provider, err)
	}
}
