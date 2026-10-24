package metronome

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := []Config{
		DefaultConfig(),
		{BPM: MinBPM, BeatsPerMeasure: 1, Subdivision: 1},
		{BPM: MaxBPM, BeatsPerMeasure: 16, Subdivision: 8},
	}
	for _, c := range valid {
		if err := c.Validate(); err != nil {
			t.Fatalf("expected %+v to be valid: %v", c, err)
		}
	}

	invalid := []Config{
		{BPM: MinBPM - 1, BeatsPerMeasure: 4, Subdivision: 1},
		{BPM: MaxBPM + 1, BeatsPerMeasure: 4, Subdivision: 1},
		{BPM: 120, BeatsPerMeasure: 0, Subdivision: 1},
		{BPM: 120, BeatsPerMeasure: 17, Subdivision: 1},
		{BPM: 120, BeatsPerMeasure: 4, Subdivision: 0},
		{BPM: 120, BeatsPerMeasure: 4, Subdivision: 9},
	}
	for _, c := range invalid {
		if err := c.Validate(); err == nil {
			t.Fatalf("expected %+v to be invalid", c)
		}
	}
}

func TestInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bpm         int
		subdivision int
		want        time.Duration
	}{
		{120, 1, 500 * time.Millisecond},
		{120, 2, 250 * time.Millisecond},
		{120, 4, 125 * time.Millisecond},
		{60, 1, time.Second},
		{240, 1, 250 * time.Millisecond},
		{0, 1, 0},
	}
	for _, tt := range tests {
		c := Config{BPM: tt.bpm, BeatsPerMeasure: 4, Subdivision: tt.subdivision}
		if got := c.Interval(); got != tt.want {
			t.Fatalf("BPM %d sub %d: got %v, want %v", tt.bpm, tt.subdivision, got, tt.want)
		}
	}
}

func TestBeatForIndex(t *testing.T) {
	t.Parallel()

	c := Config{BPM: 100, BeatsPerMeasure: 3, Subdivision: 2, Accent: true}
	tests := []struct {
		index      int
		measure    int
		beat       int
		subIndex   int
		wantAccent bool
	}{
		{0, 1, 1, 0, true},
		{1, 1, 1, 1, false},
		{2, 1, 2, 0, false},
		{3, 1, 2, 1, false},
		{4, 1, 3, 0, false},
		{5, 1, 3, 1, false},
		{6, 2, 1, 0, true},
		{12, 3, 1, 0, true},
	}
	for _, tt := range tests {
		got := c.BeatForIndex(tt.index)
		if got.Measure != tt.measure || got.Beat != tt.beat || got.SubIndex != tt.subIndex {
			t.Fatalf("index %d: got %+v", tt.index, got)
		}
		if got.Accent != tt.wantAccent {
			t.Fatalf("index %d: accent = %v, want %v", tt.index, got.Accent, tt.wantAccent)
		}
	}
}

func TestBeatForIndexNoAccent(t *testing.T) {
	t.Parallel()

	c := Config{BPM: 120, BeatsPerMeasure: 4, Subdivision: 1, Accent: false}
	if c.BeatForIndex(0).Accent {
		t.Fatal("accent should be disabled")
	}
}

func TestTapTempo(t *testing.T) {
	t.Parallel()

	tracker := NewTapTempo()
	start := time.Unix(0, 0)

	if _, ok := tracker.Tap(start); ok {
		t.Fatal("one tap should not produce a tempo")
	}
	if bpm, ok := tracker.Tap(start.Add(500 * time.Millisecond)); !ok || bpm != 120 {
		t.Fatalf("got bpm=%d ok=%v, want 120", bpm, ok)
	}
	if bpm, ok := tracker.Tap(start.Add(time.Second)); !ok || bpm != 120 {
		t.Fatalf("got bpm=%d ok=%v, want 120", bpm, ok)
	}
}

func TestTapTempoResetsAfterPause(t *testing.T) {
	t.Parallel()

	tracker := NewTapTempo()
	start := time.Unix(0, 0)
	tracker.Tap(start)
	tracker.Tap(start.Add(500 * time.Millisecond))

	if _, ok := tracker.Tap(start.Add(5 * time.Second)); ok {
		t.Fatal("expected tap history to reset after a long pause")
	}
}

func TestTapTempoClamps(t *testing.T) {
	t.Parallel()

	tracker := NewTapTempo()
	start := time.Unix(0, 0)
	tracker.Tap(start)
	if bpm, ok := tracker.Tap(start.Add(10 * time.Millisecond)); !ok || bpm != MaxBPM {
		t.Fatalf("got bpm=%d ok=%v, want clamp to %d", bpm, ok, MaxBPM)
	}
}
