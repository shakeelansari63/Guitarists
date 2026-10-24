package tabs

import (
	"math"
	"testing"
	"time"
)

func TestPlaybackNotes(t *testing.T) {
	t.Parallel()

	tab := Tab{
		Version: CurrentVersion,
		Tempo:   120,
		Lines: []Line{
			{Measures: []Measure{
				{Beats: []Beat{{String: 2, Fret: 1}}},
				{Beats: []Beat{{String: 6, Fret: 0}}},
				{},
			}},
			{Measures: []Measure{
				{Beats: []Beat{{String: 1, Fret: 12}}},
			}},
		},
	}

	notes, total, err := tab.PlaybackNotes()
	if err != nil {
		t.Fatalf("playback notes: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}
	if want := 2500 * time.Millisecond; total != want {
		t.Fatalf("total = %s, want %s (including decay tail)", total, want)
	}
	if notes[0].Start != 0 {
		t.Fatalf("first start = %s, want 0", notes[0].Start)
	}
	if notes[1].Start != 500*time.Millisecond {
		t.Fatalf("second start = %s, want 500ms", notes[1].Start)
	}
	if notes[2].Start != 1500*time.Millisecond {
		t.Fatalf("third start = %s, want 1.5s", notes[2].Start)
	}
	if notes[0].Duration != 450*time.Millisecond {
		t.Fatalf("duration = %s, want 450ms", notes[0].Duration)
	}

	const c4 = 261.6256
	if math.Abs(notes[0].Frequency-c4) > 0.01 {
		t.Fatalf("frequency = %f, want %f", notes[0].Frequency, c4)
	}
	const highE12 = 659.2551
	if math.Abs(notes[2].Frequency-highE12) > 0.01 {
		t.Fatalf("frequency = %f, want %f", notes[2].Frequency, highE12)
	}
}

func TestPlaybackLinePause(t *testing.T) {
	t.Parallel()

	tab := Tab{
		Version: CurrentVersion,
		Tempo:   120,
		Lines: []Line{
			{Pause: 2, Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 0}}}}},
			{Measures: []Measure{{Beats: []Beat{{String: 1, Fret: 0}}}}},
		},
	}

	notes, total, err := tab.PlaybackNotes()
	if err != nil {
		t.Fatalf("playback notes: %v", err)
	}
	if want := 1500 * time.Millisecond; notes[1].Start != want {
		t.Fatalf("second line start = %s, want %s", notes[1].Start, want)
	}
	if want := 2500 * time.Millisecond; total != want {
		t.Fatalf("total = %s, want %s", total, want)
	}
}

func TestPlaybackSlide(t *testing.T) {
	t.Parallel()

	tab := Tab{
		Version: CurrentVersion,
		Tempo:   120,
		Lines: []Line{{Measures: []Measure{{
			Beats: []Beat{{String: 1, Fret: 5, SlideTo: intPtr(7)}},
		}}}},
	}

	notes, _, err := tab.PlaybackNotes()
	if err != nil {
		t.Fatalf("playback notes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2 for a slide", len(notes))
	}
	if notes[1].Start != 225*time.Millisecond {
		t.Fatalf("slide start = %s, want 225ms", notes[1].Start)
	}
	const a4 = 440.0
	const b4 = 493.883
	if math.Abs(notes[0].Frequency-a4) > 0.01 {
		t.Fatalf("first frequency = %f, want %f", notes[0].Frequency, a4)
	}
	if math.Abs(notes[1].Frequency-b4) > 0.01 {
		t.Fatalf("slide frequency = %f, want %f", notes[1].Frequency, b4)
	}
}

func TestPlaybackNotesEmpty(t *testing.T) {
	t.Parallel()

	tab := New()
	_, _, err := tab.PlaybackNotes()
	if err == nil {
		t.Fatal("expected empty tab to fail")
	}
}

func TestPlaybackNotesTooLong(t *testing.T) {
	t.Parallel()

	measures := make([]Measure, MaxMeasures)
	measures[0] = Measure{Beats: []Beat{{String: 1, Fret: 0}}}
	tab := Tab{Version: CurrentVersion, Tempo: 120, Lines: []Line{{Measures: measures}}}
	if _, _, err := tab.PlaybackNotes(); err == nil {
		t.Fatal("expected long tab to fail")
	}
}

func TestRenderPlayback(t *testing.T) {
	t.Parallel()

	notes := []PlaybackNote{{Start: 0, Duration: 100 * time.Millisecond, Frequency: 440}}
	samples := RenderPlayback(notes, 100*time.Millisecond, 1000)
	if len(samples) != 100 {
		t.Fatalf("got %d samples, want 100", len(samples))
	}
	peak := 0.0
	for _, sample := range samples {
		if value := math.Abs(float64(sample)); value > peak {
			peak = value
		}
	}
	if peak == 0 {
		t.Fatal("expected non-zero audio")
	}
	if peak > 0.95 {
		t.Fatalf("peak = %f, want <= 0.95", peak)
	}
}
