package main

import "testing"

func TestToggleShortcutParsing(t *testing.T) {
	tests := []struct {
		input, display string
		modifiers, key uint32
	}{
		{"Ctrl+Alt+T", "Ctrl+Alt+T", 3, 'T'},
		{"  control + shift + f5 ", "Ctrl+Shift+F5", 6, 0x74},
		{"Ctrl+Alt+Shift+9", "Ctrl+Alt+Shift+9", 7, '9'},
		{"Shift+Ctrl+Z", "Ctrl+Shift+Z", 6, 'Z'},
	}
	for _, tc := range tests {
		actual, err := parseToggleShortcut(tc.input)
		if err != nil {
			t.Fatalf("%q: %v", tc.input, err)
		}
		if actual.Display != tc.display || actual.Modifiers != tc.modifiers || actual.Key != tc.key {
			t.Errorf("%q => %+v; expected %q mods=%d key=%d", tc.input, actual, tc.display, tc.modifiers, tc.key)
		}
		again, err := parseToggleShortcut(actual.Display)
		if err != nil || again != actual {
			t.Errorf("non-round-trip for %q: %+v %v", tc.input, again, err)
		}
	}
}

func TestRejectUnsafeOrInvalidToggleShortcuts(t *testing.T) {
	invalid := []string{"", "T", "Ctrl+T", "Alt+T", "Shift+T", "Ctrl+Alt", "Ctrl+Alt+F12", "Ctrl+Alt+F13", "Ctrl+Alt+Delete", "Ctrl+Alt+T+U", "Ctrl+Ctrl+Alt+T", "Ctrl+Alt+Ctrl+T", "Win+Alt+T", "Ctrl+Alt++T", "Ctrl+Alt+-"}
	for _, s := range invalid {
		if _, err := parseToggleShortcut(s); err == nil {
			t.Errorf("expected invalid shortcut: %q", s)
		}
	}
}
