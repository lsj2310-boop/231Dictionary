package main

import (
	"errors"
	"strings"
	"time"
)

// persistedTranslationState returns a deliberately strict setting; unknown or
// corrupted settings should not silently disable the application on startup.
func persistedTranslationState(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "enabled", "true", "1":
		return true, nil
	case "off", "disabled", "false", "0":
		return false, nil
	default:
		return true, errors.New("unknown saved translation toggle state")
	}
}

func serializeTranslationState(enabled bool) string {
	if enabled {
		return "enabled\n"
	}
	return "disabled\n"
}

// trayClickCanToggle ignores the mouse-up emitted after WM_LBUTTONDBLCLK.
// The first click has already toggled, so the second one must not undo it.
func trayClickCanToggle(now time.Time, ignoreUntilNanos int64) bool {
	return now.UnixNano() >= ignoreUntilNanos
}
