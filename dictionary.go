package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed dictionary_ko.tsv
var localDictionaryFS embed.FS

// translationResult separates a valid translation from a networking failure.
// v1.2 incorrectly treated "뜻 조회 실패" as a successful translation.
type translationResult struct {
	Meaning string
	Source  string
	Err     string
}

var (
	dictOnce      sync.Once
	dictMu        sync.RWMutex
	builtinDict   = make(map[string]string)
	customDict    = make(map[string]string)
	savedDict     = make(map[string]string)
	remoteStateMu sync.Mutex
	remoteRetryAt time.Time
	remoteError   string
)

func dataDir() string {
	if p := os.Getenv("LOCALAPPDATA"); p != "" {
		return filepath.Join(p, "231Dictionary")
	}
	if p, err := os.UserConfigDir(); err == nil {
		return filepath.Join(p, "231Dictionary")
	}
	return filepath.Join(os.TempDir(), "231Dictionary")
}
func userDictionaryPath() string { return filepath.Join(dataDir(), "user_dictionary.tsv") }
func networkCachePath() string   { return filepath.Join(dataDir(), "translation_cache.tsv") }

// Simple UTF-8 TSV dictionary. '#' lines and malformed lines are skipped.
func parseDictionary(data []byte) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 512), 64*1024)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		pair := strings.SplitN(s, "\t", 2)
		if len(pair) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(pair[0]))
		v := strings.TrimSpace(pair[1])
		if k == "" || v == "" || len(k) > 64 || len([]rune(v)) > 150 {
			continue
		}
		out[k] = v
	}
	return out
}
func prepareDictionary() {
	dictOnce.Do(func() {
		data, err := localDictionaryFS.ReadFile("dictionary_ko.tsv")
		if err == nil {
			builtinDict = parseDictionary(data)
		}
		_ = os.MkdirAll(dataDir(), 0755)
		if _, err = os.Stat(userDictionaryPath()); errors.Is(err, os.ErrNotExist) {
			// User glossary has highest priority and never gets overwritten on updates.
			_ = os.WriteFile(userDictionaryPath(), []byte("# EnglishWord[TAB]Korean meaning (UTF-8)\n# Edit the file and reload the custom dictionary from the tray menu\nTachograph\t운행기록계, 디지털 운행기록장치(DTG)\nWarranty\t품질보증, 보증기간\n"), 0644)
		}
		if data, e := os.ReadFile(userDictionaryPath()); e == nil {
			customDict = parseDictionary(data)
		}
		if data, e := os.ReadFile(networkCachePath()); e == nil {
			savedDict = parseDictionary(data)
		}
	})
}
func reloadCustomDictionary() (int, error) {
	prepareDictionary()
	b, e := os.ReadFile(userDictionaryPath())
	if e != nil {
		return 0, e
	}
	d := parseDictionary(b)
	dictMu.Lock()
	customDict = d
	dictMu.Unlock()
	return len(d), nil
}
func offlineLookup(word string) (translationResult, bool) {
	prepareDictionary()
	key := strings.ToLower(strings.TrimSpace(word))
	dictMu.RLock()
	defer dictMu.RUnlock()
	if v, ok := customDict[key]; ok {
		return translationResult{v, "Custom dictionary", ""}, true
	}
	if v, ok := builtinDict[key]; ok {
		return translationResult{v, "Built-in dictionary", ""}, true
	}
	if v, ok := savedDict[key]; ok {
		return translationResult{v, "Cached translation", ""}, true
	}
	// Limited, unambiguous morphological fallback (e.g., warranties -> warranty).
	variants := []string{}
	if strings.HasSuffix(key, "ies") && len(key) > 4 {
		variants = append(variants, strings.TrimSuffix(key, "ies")+"y")
	}
	if strings.HasSuffix(key, "es") && len(key) > 4 {
		variants = append(variants, strings.TrimSuffix(key, "es"))
	}
	if strings.HasSuffix(key, "s") && len(key) > 3 {
		variants = append(variants, strings.TrimSuffix(key, "s"))
	}
	if strings.HasSuffix(key, "ed") && len(key) > 4 {
		variants = append(variants, strings.TrimSuffix(key, "ed"))
	}
	for _, alt := range variants {
		if v, ok := customDict[alt]; ok {
			return translationResult{v, "Custom dictionary", ""}, true
		}
		if v, ok := builtinDict[alt]; ok {
			return translationResult{v, "Built-in dictionary", ""}, true
		}
		if v, ok := savedDict[alt]; ok {
			return translationResult{v, "Cached translation", ""}, true
		}
	}
	return translationResult{}, false
}
func remoteStatus() string {
	remoteStateMu.Lock()
	defer remoteStateMu.Unlock()
	if !remoteRetryAt.IsZero() && time.Now().Before(remoteRetryAt) {
		return fmt.Sprintf("Online translation paused (%ds until retry); last error: %s", int(time.Until(remoteRetryAt).Seconds()+1), remoteError)
	}
	if remoteError != "" {
		return "Last online translation error: " + remoteError
	}
	return "Online translation connection has not been tested yet"
}

// Every hovered word is translated by Gemini. Cached entries are only results
// returned from Gemini in this session; legacy offline/Google meanings are ignored.
var geminiWordMemory sync.Map

func translateDetailed(word string) translationResult {
	key := strings.ToLower(strings.TrimSpace(word))
	if v, ok := geminiWordMemory.Load(key); ok {
		return translationResult{Meaning: v.(string), Source: "Gemini cache"}
	}
	if !onlineWordLookupAllowed() {
		return translationResult{Meaning: "Online word translation is disabled", Source: "Settings"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()
	meaning, err := remoteTranslate(ctx, word)
	if err != nil {
		return translationResult{Meaning: "Gemini translation failed", Source: "Gemini", Err: err.Error()}
	}
	geminiWordMemory.Store(key, meaning)
	return translationResult{Meaning: meaning, Source: "Gemini"}
}

func saveTranslation(word, meaning string) {
	// Only English words (not document screenshots, complete sentences) are persisted locally.
	if word == "" || meaning == "" || strings.ContainsAny(word, "\r\n\t") || strings.ContainsAny(meaning, "\r\n\t") {
		return
	}
	dictMu.Lock()
	defer dictMu.Unlock()
	k := strings.ToLower(word)
	if _, ok := savedDict[k]; ok {
		return
	}
	savedDict[k] = meaning
	f, err := os.OpenFile(networkCachePath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s\t%s\n", k, meaning)
}

// Online word lookups use the MouseTooltipTranslator-inspired Google web methods.
// Only unknown word text is sent; images and entire documents are not uploaded.
func remoteTranslate(ctx context.Context, word string) (string, error) {
	if len(word) < 2 || len(word) > 60 {
		return "", errors.New("Invalid English word length")
	}
	result, _, err := translateGeminiWord(ctx, word)
	return result, err
}
