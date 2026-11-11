package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// toggleShortcut is independent of the Windows UI so shortcuts can be
// validated in normal Go unit tests before they are registered system-wide.
type toggleShortcut struct {
	Modifiers uint32
	Key       uint32
	Display   string
}

const (
	shortcutModAlt     uint32 = 0x0001
	shortcutModControl uint32 = 0x0002
	shortcutModShift   uint32 = 0x0004
)

func parseToggleShortcut(input string) (toggleShortcut, error) {
	var spec toggleShortcut
	input = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input), " ", ""))
	parts := strings.Split(input, "+")
	if len(parts) < 3 || len(parts) > 4 {
		return spec, errors.New("use at least two modifiers, e.g. Ctrl+Alt+T or Ctrl+Shift+T")
	}
	var gotKey bool
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" {
			return spec, errors.New("shortcut contains an empty key")
		}
		switch part {
		case "CTRL", "CONTROL", "ALT", "SHIFT":
			canonical := part
			if canonical == "CONTROL" {
				canonical = "CTRL"
			}
			if seen[canonical] {
				return spec, fmt.Errorf("duplicate modifier: %s", canonical)
			}
			if gotKey {
				return spec, errors.New("put modifiers before the final key")
			}
			seen[canonical] = true
			switch canonical {
			case "CTRL":
				spec.Modifiers |= shortcutModControl
			case "ALT":
				spec.Modifiers |= shortcutModAlt
			case "SHIFT":
				spec.Modifiers |= shortcutModShift
			}
		default:
			if gotKey {
				return spec, errors.New("specify only one final key")
			}
			if len(part) == 1 {
				c := part[0]
				if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
					spec.Key = uint32(c)
				} else {
					return spec, errors.New("use a letter, number, or F1-F11")
				}
			} else if strings.HasPrefix(part, "F") {
				num, err := strconv.Atoi(part[1:])
				if err != nil || num < 1 || num > 11 {
					return spec, errors.New("function keys F1-F11 are supported (F12 is reserved by Windows)")
				}
				spec.Key = uint32(0x70 + num - 1)
			} else {
				return spec, errors.New("use a letter, number, or F1-F11")
			}
			gotKey = true
		}
	}
	if !gotKey || spec.Modifiers&shortcutModControl == 0 || spec.Modifiers&(shortcutModAlt|shortcutModShift) == 0 {
		return toggleShortcut{}, errors.New("a shortcut must contain Ctrl+Alt or Ctrl+Shift and one final key")
	}
	keys := []string{"Ctrl"}
	if spec.Modifiers&shortcutModAlt != 0 {
		keys = append(keys, "Alt")
	}
	if spec.Modifiers&shortcutModShift != 0 {
		keys = append(keys, "Shift")
	}
	if spec.Key >= 0x70 && spec.Key <= 0x7A {
		keys = append(keys, fmt.Sprintf("F%d", spec.Key-0x70+1))
	} else {
		keys = append(keys, string(rune(spec.Key)))
	}
	spec.Display = strings.Join(keys, "+")
	return spec, nil
}
