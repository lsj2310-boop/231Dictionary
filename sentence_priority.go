package main

import "time"

// A completed or in-progress drag selection always outranks word-hover OCR,
// independently of how far the mouse has moved since button release.
func sentencePriorityActive(dragging, selecting, showing bool) bool {
	return dragging || selecting || showing
}
func keepSentenceVisible(now, until time.Time) bool {
	return now.Before(until)
}
