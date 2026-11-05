package main

import (
	"path/filepath"
	"strings"
)

// KakaoTalk windows are always excluded from translation regardless of the
// user's optional ordinary-web-page exclusion setting. Match the executable
// name rather than its window title or the window's visible content.
var kakaoTalkExecutables = map[string]struct{}{
	"kakaotalk.exe":         {},
	"kakaotalkbeta.exe":     {},
	"kakaotalkpc.exe":       {},
	"kakaotalk64.exe":       {},
	"kakaotalkportable.exe": {},
}

func isKakaoTalkExecutable(processPath string) bool {
	name := strings.ToLower(filepath.Base(strings.ReplaceAll(strings.TrimSpace(processPath), `\`, `/`)))
	_, ok := kakaoTalkExecutables[name]
	return ok
}

// Pure helper used by tests. A browser PDF is permitted, but KakaoTalk
// remains excluded even if the browser-exclusion preference is disabled.
func shouldSuppressApplication(processPath string, excludeBrowsers, isPDF bool) bool {
	if isKakaoTalkExecutable(processPath) {
		return true
	}
	return excludeBrowsers && isBrowserExecutable(processPath) && !isPDF
}
