package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// normalizeSelectedLines preserves hard line breaks from UI Automation / clipboard.
// Only horizontal whitespace inside each line is collapsed.
func normalizeSelectedLines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\u2028", "\n")
	text = strings.ReplaceAll(text, "\u2029", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = strings.ReplaceAll(line, "\u200b", "")
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

type translatorLine func(context.Context, string) (string, string, error)

// Each visible source line is translated independently. This is intentional:
// LLMs can otherwise merge separate source lines despite a formatting prompt.
// Results are reassembled in original order, preserving blank lines.
func translateSourceLines(ctx context.Context, source string, fn translatorLine) (string, string, error) {
	lines := strings.Split(normalizeSelectedLines(source), "\n")
	if len(lines) == 1 {
		return fn(ctx, lines[0])
	}
	translated := make([]string, len(lines))
	providers := make([]string, len(lines))
	errorsByLine := make([]error, len(lines))
	limiter := make(chan struct{}, 2)
	var wg sync.WaitGroup
	for i, line := range lines {
		if line == "" {
			continue
		}
		wg.Add(1)
		go func(index int, sourceLine string) {
			defer wg.Done()
			select {
			case limiter <- struct{}{}:
			case <-ctx.Done():
				errorsByLine[index] = ctx.Err()
				return
			}
			defer func() { <-limiter }()
			translated[index], providers[index], errorsByLine[index] = fn(ctx, sourceLine)
		}(i, line)
	}
	wg.Wait()
	provider := "Gemini"
	for i, e := range errorsByLine {
		if e != nil {
			return "", "", fmt.Errorf("line %d: %w", i+1, e)
		}
		if translated[i] == "" && lines[i] != "" {
			return "", "", fmt.Errorf("line %d: empty translation", i+1)
		}
		if providers[i] != "" {
			provider = providers[i]
		}
	}
	return strings.Join(translated, "\n"), provider, nil
}
