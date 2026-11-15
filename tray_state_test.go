package main

import (
	"testing"
	"time"
)

func TestTranslationStateRoundTrip(t *testing.T) {
	for _, on := range []bool{true, false} {
		got, err := persistedTranslationState(serializeTranslationState(on))
		if err != nil || got != on {
			t.Fatalf("state=%v got=%v err=%v", on, got, err)
		}
	}
}
func TestTranslationStateCorruptionFailsOpen(t *testing.T) {
	got, err := persistedTranslationState("corrupted")
	if err == nil || !got {
		t.Fatalf("corrupt data should be reported, default enabled: got=%v err=%v", got, err)
	}
}

func TestTrayDoubleClickIsSingleToggle(t *testing.T) {
	first := time.Date(2026, 11, 16, 11, 0, 0, 0, time.UTC)
	if !trayClickCanToggle(first, 0) {
		t.Fatal("first click must toggle")
	}
	until := first.Add(800 * time.Millisecond).UnixNano()
	if trayClickCanToggle(first.Add(180*time.Millisecond), until) {
		t.Fatal("second click must not undo toggle")
	}
	if !trayClickCanToggle(first.Add(1*time.Second), until) {
		t.Fatal("subsequent click must work")
	}
}
