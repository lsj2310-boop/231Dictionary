package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSelectedLinesPreservesRows(t *testing.T) {
	in := "Quality  assurance.\r\n  Warranty   coverage.\r\n\r\nThe third line."
	expected := "Quality assurance.\nWarranty coverage.\n\nThe third line."
	if got := normalizeSelectedLines(in); got != expected {
		t.Fatalf("got %q, want %q", got, expected)
	}
}
func TestTranslateSourceLinesKeepsOrderAndBlankLines(t *testing.T) {
	input := "Hydraulic pressure is low.\n\nThe electric pump failed."
	got, _, err := translateSourceLines(context.Background(), input, func(ctx context.Context, s string) (string, string, error) {
		if strings.Contains(s, "Hydraulic") {
			time.Sleep(35 * time.Millisecond)
			return "유압이 낮습니다.", "Gemini", nil
		}
		return "전동 펌프가 고장났습니다.", "Gemini", nil
	})
	expected := "유압이 낮습니다.\n\n전동 펌프가 고장났습니다."
	if err != nil || got != expected {
		t.Fatalf("got %q err %v", got, err)
	}
}
