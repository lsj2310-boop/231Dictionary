package main

import "strings"

// Only a completed Korean-to-English sentence translation can be copied.
// The glossary is kept in a separate field and is never concatenated here.
func eligibleEnglishCopy(source, translated string, completed bool) string {
	if !completed || directionForText(source) != koreanToEnglish {
		return ""
	}
	value := strings.TrimSpace(translated)
	if value == "" || strings.HasPrefix(value, "Translating") || strings.HasPrefix(value, "Translation failed") || strings.HasPrefix(value, "Online translation is disabled") || strings.HasPrefix(value, "Could not read selected text") || strings.HasPrefix(value, "Selection text access error") {
		return ""
	}
	return value
}
