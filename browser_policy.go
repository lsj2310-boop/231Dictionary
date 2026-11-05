package main

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Exclude browsers on ordinary web pages; a PDF document opened inside a
// browser is intentionally permitted.  No browser history is retained.
var knownBrowserExecutables = map[string]struct{}{
	"chrome.exe": {}, "chromium.exe": {}, "msedge.exe": {},
	"whale.exe": {}, "naverwhale.exe": {},
	"aside.exe": {}, "aside-browser.exe": {}, "asidebrowser.exe": {},
	"brave.exe": {}, "brave-browser.exe": {},
	"firefox.exe": {}, "waterfox.exe": {}, "zen.exe": {},
	"opera.exe": {}, "opera_gx.exe": {}, "opera_gx_browser.exe": {},
	"vivaldi.exe": {}, "arc.exe": {}, "duckduckgo.exe": {},
	"iexplore.exe": {}, "sider.exe": {}, "thorium.exe": {},
}

func isBrowserExecutable(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(path, `\`, `/`)))
	if normalized == "" {
		return false
	}
	_, ok := knownBrowserExecutables[strings.ToLower(filepath.Base(normalized))]
	return ok
}

func browserExclusionFromFile(data []byte) bool {
	switch strings.ToLower(strings.TrimSpace(string(data))) {
	case "off", "false", "0", "no":
		return false
	default:
		return true
	}
}

// A tab title with an actual .pdf filename is a fast, cheap indication, so
// browser PDF translation works immediately without waiting for UI Automation.
// "PDF tools" or "How to create a PDF" is NOT sufficient.
var pdfFileTitlePattern = regexp.MustCompile(`(?i)\.pdf(?:$|[\s\]\)\-—|])`)

func isPDFBrowserTitle(title string) bool {
	return pdfFileTitlePattern.MatchString(strings.TrimSpace(title))
}

// Used by the UI Automation browser address-bar probe. Parse the URL rather
// than searching for ".pdf" anywhere in the query string (search results are
// not PDF documents). URL-only links that hide their content type will require
// the browser PDF accessibility control to be available.
func isPDFDocumentURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	path, _ := url.PathUnescape(parsed.EscapedPath())
	if strings.HasSuffix(strings.ToLower(path), ".pdf") {
		return true
	}
	params := parsed.Query()
	for _, name := range []string{"file", "url", "document", "src", "pdf"} {
		for _, value := range params[name] {
			value, _ = url.QueryUnescape(value)
			v, e := url.Parse(value)
			if e == nil && v.Path != "" && strings.HasSuffix(strings.ToLower(v.Path), ".pdf") {
				return true
			}
			if strings.HasSuffix(strings.ToLower(strings.SplitN(value, "?", 2)[0]), ".pdf") {
				return true
			}
		}
	}
	return false
}
