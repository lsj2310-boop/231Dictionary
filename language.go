package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Language is inferred from the selected text only; no extra screen content is
// sent to the translation provider. English acronyms in Korean paragraphs are
// intentionally permitted (e.g. "브레이크 ECU 점검 요청").
type translationDirection string

const (
	englishToKorean translationDirection = "en-ko"
	koreanToEnglish translationDirection = "ko-en"
)

var validKoreanHover = regexp.MustCompile(`^[가-힣]{1,32}$`)
var validEnglishHover = regexp.MustCompile(`^[A-Za-z][A-Za-z'-]{1,38}$`)

func isValidHoverWord(word string) bool {
	return validEnglishHover.MatchString(word) || validKoreanHover.MatchString(word)
}

func directionForText(text string) translationDirection {
	korean, latin := 0, 0
	for _, r := range text {
		switch {
		case r >= '가' && r <= '힣':
			korean++
		case r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z':
			latin++
		}
	}
	// Technical acronyms can be longer than Hangul words. Prefer Korean when
	// Hangul clearly forms substantial text, even if Latin acronyms occur.
	if korean > 0 && (latin == 0 || korean >= 3 && korean*2 >= latin || korean >= 7 && korean >= latin/3) {
		return koreanToEnglish
	}
	return englishToKorean
}

func selectedLanguageSupported(text string) bool {
	s := strings.TrimSpace(text)
	if utf8.RuneCountInString(s) < 2 {
		return false
	}
	if directionForText(s) == koreanToEnglish {
		count := 0
		for _, r := range s {
			if r >= '가' && r <= '힣' {
				count++
			}
		}
		return count >= 2
	}
	return utf8.RuneCountInString(s) >= 7 && len(wordToken.FindAllString(s, -1)) >= 2
}
