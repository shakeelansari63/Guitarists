// Package metronome contains tempo, timing, and beat-scheduling logic.
// It has no dependency on Wails or audio device APIs.
package metronome

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	MinBPM             = 20
	MaxBPM             = 400
	MinBeatsPerMeasure = 1
	MaxBeatsPerMeasure = 16
	MinSubdivision     = 1
	MaxSubdivision     = 8
)

// Config controls metronome timing.
type Config struct {
	BPM             int  `json:"bpm"`
	BeatsPerMeasure int  `json:"beatsPerMeasure"`
	Subdivision     int  `json:"subdivision"`
	Accent          bool `json:"accent"`
}

// DefaultConfig returns a 4/4, 120 BPM configuration with accented downbeats.
func DefaultConfig() Config {
	return Config{BPM: 120, BeatsPerMeasure: 4, Subdivision: 1, Accent: true}
}

// Validate checks that all timing values are within supported ranges.
func (c Config) Validate() error {
	if c.BPM < MinBPM || c.BPM > MaxBPM {
		return fmt.Errorf("bpm must be between %d and %d", MinBPM, MaxBPM)
	}
	if c.BeatsPerMeasure < MinBeatsPerMeasure || c.BeatsPerMeasure > MaxBeatsPerMeasure {
		return fmt.Errorf("beats per measure must be between %d and %d", MinBeatsPerMeasure, MaxBeatsPerMeasure)
	}
	if c.Subdivision < MinSubdivision || c.Subdivision > MaxSubdivision {
		return fmt.Errorf("subdivision must be between %d and %d", MinSubdivision, MaxSubdivision)
	}
	return nil
}

// Interval returns the duration between subdivision clicks.
func (c Config) Interval() time.Duration {
	if c.BPM <= 0 || c.Subdivision <= 0 {
		return 0
	}
	nanos := float64(time.Minute) / float64(c.BPM*c.Subdivision)
	return time.Duration(math.Round(nanos))
}

// Beat describes one scheduled click.
type Beat struct {
	Index       int  `json:"index"`
	Measure     int  `json:"measure"`
	Beat        int  `json:"beat"`
	SubIndex    int  `json:"subIndex"`
	Accent      bool `json:"accent"`
	BPM         int  `json:"bpm"`
	Subdivision int  `json:"subdivision"`
}

// BeatForIndex computes the musical position of a click sequence index.
func (c Config) BeatForIndex(index int) Beat {
	if index < 0 {
		index = 0
	}
	if c.BeatsPerMeasure <= 0 {
		c.BeatsPerMeasure = 4
	}
	if c.Subdivision <= 0 {
		c.Subdivision = 1
	}
	perMeasure := c.BeatsPerMeasure * c.Subdivision
	measure := index/perMeasure + 1
	withinMeasure := index % perMeasure
	return Beat{
		Index:       index,
		Measure:     measure,
		Beat:        withinMeasure/c.Subdivision + 1,
		SubIndex:    withinMeasure % c.Subdivision,
		Accent:      c.Accent && withinMeasure == 0,
		BPM:         c.BPM,
		Subdivision: c.Subdivision,
	}
}

// TapTempo derives a BPM from a sequence of taps.
type TapTempo struct {
	taps       []time.Time
	timeout    time.Duration
	maxSamples int
}

// NewTapTempo creates an empty tap tempo tracker.
func NewTapTempo() *TapTempo {
	return &TapTempo{timeout: 2 * time.Second, maxSamples: 8}
}

// Reset clears the tap history.
func (t *TapTempo) Reset() {
	t.taps = nil
}

// Tap records a tap at now and returns the derived BPM once two taps exist.
func (t *TapTempo) Tap(now time.Time) (int, bool) {
	if len(t.taps) > 0 {
		gap := now.Sub(t.taps[len(t.taps)-1])
		if gap <= 0 || gap > t.timeout {
			t.taps = nil
		}
	}
	t.taps = append(t.taps, now)
	if len(t.taps) > t.maxSamples {
		t.taps = t.taps[len(t.taps)-t.maxSamples:]
	}
	if len(t.taps) < 2 {
		return 0, false
	}

	total := t.taps[len(t.taps)-1].Sub(t.taps[0])
	intervals := len(t.taps) - 1
	bpm := int(math.Round(60.0 / (total.Seconds() / float64(intervals))))
	if bpm < MinBPM {
		bpm = MinBPM
	}
	if bpm > MaxBPM {
		bpm = MaxBPM
	}
	return bpm, true
}

// ErrNotEnoughTaps is returned when tap tempo is requested before two taps.
var ErrNotEnoughTaps = errors.New("tap at least twice to set the tempo")
