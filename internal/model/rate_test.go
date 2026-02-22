package model

import (
	"testing"
	"time"
)

func TestRateCalculator_Empty(t *testing.T) {
	rc := NewRateCalculator(10 * time.Second)
	if rate := rc.Rate(); rate != 0 {
		t.Errorf("expected 0 rate for empty calculator, got %f", rate)
	}
}

func TestRateCalculator_ETA(t *testing.T) {
	rc := NewRateCalculator(10 * time.Second)
	if eta := rc.ETA(1000); eta != 0 {
		t.Errorf("expected 0 ETA for empty calculator, got %f", eta)
	}
}
