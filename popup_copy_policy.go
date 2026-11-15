package main

import "strings"

// eligibleTranslationCopy returns the completed translation body for either
// English->Korean or Korean->English. The selected source and the glossary
// are deliberately never copied. Pending/error text is also excluded.
func eligibleTranslationCopy(source, translated string, completed bool) string {
	if !completed || !selectedLanguageSupported(source) {
		return ""
	}
	value := strings.TrimSpace(translated)
	if value == "" {
		return ""
	}
	for _, message := range []string{
		"Translating", "Translation failed", "Online translation is disabled",
		"Could not read selected text", "Selection text access error",
	} {
		if strings.HasPrefix(value, message) {
			return ""
		}
	}
	return value
}

// The copied string must be the translation text currently displayed in the
// popup, never the selected text in PowerPoint/PDF or the glossary below it.
func displayedTranslationForCopy(visible, eligible string, popupShown, sentenceShown bool, mode string) string {
	if !popupShown || !sentenceShown || mode != "sentence" || eligible == "" {
		return ""
	}
	if strings.TrimSpace(visible) != eligible {
		return "" // The popup has changed while an old result was pending.
	}
	return eligible
}
