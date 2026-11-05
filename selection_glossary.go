package main

import "strings"

// English-to-Korean: explain English terms found in the original.
// Korean-to-English: explain important English terms from the *translated*
// sentence. The Gemini-provided contextual glossary is preferred, with the
// local quality/automotive glossary as a fallback when JSON is unavailable.
func selectionGlossary(source, translated string) string {
	if directionForText(source) == koreanToEnglish {
		if v, ok := koreanSentenceGlossary.Load(source); ok {
			if glossary, ok := v.(string); ok && glossary != "" {
				return glossary
			}
		}
		return extractKeyTerms(translated, 5)
	}
	return extractKeyTerms(source, 5)
}

// When the source spans several rows, Gemini provides a vocabulary list for
// each Korean source row. Collect at most five nonduplicated terms under the
// full selection, so cached multiline translations retain their glossary.
func rememberCombinedKoreanGlossary(source, translated string) {
	if directionForText(source) != koreanToEnglish {
		return
	}
	sources := strings.Split(normalizeSelectedLines(source), "\n")
	translations := strings.Split(normalizeSelectedLines(translated), "\n")
	all := make([]string, 0, 5)
	seen := make(map[string]bool)
	for i, row := range sources {
		if strings.TrimSpace(row) == "" {
			continue
		}
		glossary := ""
		if val, ok := koreanSentenceGlossary.Load(row); ok {
			glossary, _ = val.(string)
		}
		if glossary == "" && i < len(translations) {
			glossary = extractKeyTerms(translations[i], 5)
		}
		for _, item := range strings.Split(glossary, "\n") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(strings.SplitN(item, "·", 2)[0]))
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, item)
			if len(all) >= 5 {
				break
			}
		}
		if len(all) >= 5 {
			break
		}
	}
	if len(all) == 0 {
		glossary := extractKeyTerms(translated, 5)
		if glossary != "" {
			koreanSentenceGlossary.Store(source, glossary)
		}
		return
	}
	koreanSentenceGlossary.Store(source, strings.Join(all, "\n"))
}
