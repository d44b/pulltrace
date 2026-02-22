package model

import (
	"sync"
	"time"
)

// RateCalculator tracks download rates using a sliding window.
type RateCalculator struct {
	mu      sync.Mutex
	samples []rateSample
	window  time.Duration
}

type rateSample struct {
	timestamp time.Time
	bytes     int64
}

func NewRateCalculator(window time.Duration) *RateCalculator {
	return &RateCalculator{
		window: window,
	}
}

func (r *RateCalculator) Add(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.samples = append(r.samples, rateSample{timestamp: now, bytes: bytes})
	r.prune(now)
}

func (r *RateCalculator) Rate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.prune(now)
	if len(r.samples) < 2 {
		return 0
	}
	first := r.samples[0]
	last := r.samples[len(r.samples)-1]
	elapsed := last.timestamp.Sub(first.timestamp).Seconds()
	if elapsed <= 0 {
		return 0
	}
	totalBytes := last.bytes - first.bytes
	return float64(totalBytes) / elapsed
}

func (r *RateCalculator) ETA(remaining int64) float64 {
	rate := r.Rate()
	if rate <= 0 {
		return 0
	}
	return float64(remaining) / rate
}

func (r *RateCalculator) prune(now time.Time) {
	cutoff := now.Add(-r.window)
	i := 0
	for i < len(r.samples) && r.samples[i].timestamp.Before(cutoff) {
		i++
	}
	if i > 0 && i < len(r.samples) {
		r.samples = r.samples[i-1:]
	}
}
