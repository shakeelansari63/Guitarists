package audio

import (
	"math"
	"testing"
)

func TestClickSamples(t *testing.T) {
	t.Parallel()

	for _, accent := range []bool{false, true} {
		samples := clickSamples(accent)
		if len(samples) == 0 {
			t.Fatalf("accent=%v: no samples", accent)
		}
		peak := 0.0
		for _, s := range samples {
			if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
				t.Fatalf("accent=%v: non-finite sample", accent)
			}
			if v := math.Abs(float64(s)); v > peak {
				peak = v
			}
		}
		if peak <= 0.1 || peak > 1 {
			t.Fatalf("accent=%v: unexpected peak %v", accent, peak)
		}
	}
}

func TestAccentIsLouder(t *testing.T) {
	t.Parallel()

	normal := clickSamples(false)
	accent := clickSamples(true)
	if len(normal) != len(accent) {
		t.Fatal("click durations differ")
	}

	peak := func(samples []float32) float64 {
		max := 0.0
		for _, s := range samples {
			if v := math.Abs(float64(s)); v > max {
				max = v
			}
		}
		return max
	}
	if peak(accent) <= peak(normal) {
		t.Fatalf("accent peak %v is not louder than normal peak %v", peak(accent), peak(normal))
	}
}
