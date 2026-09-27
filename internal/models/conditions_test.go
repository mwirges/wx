package models

import "testing"

func TestComputeFeelsLike(t *testing.T) {
	wc := -5.0
	hi := 35.0

	// Wind chill takes priority
	got := ComputeFeelsLike(&wc, &hi)
	if got == nil || *got != -5.0 {
		t.Errorf("expected -5.0, got %v", got)
	}

	// Heat index when wind chill is nil
	got = ComputeFeelsLike(nil, &hi)
	if got == nil || *got != 35.0 {
		t.Errorf("expected 35.0, got %v", got)
	}

	// Nil when neither is present
	got = ComputeFeelsLike(nil, nil)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}
