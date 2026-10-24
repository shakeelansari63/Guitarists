package tabs

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/freak/guitarists/internal/tuner"
)

const (
	// MaxPlaybackDuration bounds how long a tab may be played in one pass.
	MaxPlaybackDuration = 5 * time.Minute
	playbackTail        = 500 * time.Millisecond
	defaultPlaybackBPM  = 120
)

// PlaybackNote is one fretted note scheduled for audio playback.
type PlaybackNote struct {
	Start     time.Duration
	Duration  time.Duration
	Frequency float64
}

// PlaybackNotes converts the tab into timed frequencies using its tempo.
// Empty positions are silent beats; lines follow their configured pause.
func (t *Tab) PlaybackNotes() ([]PlaybackNote, time.Duration, error) {
	if t == nil {
		return nil, 0, errors.New("tab is nil")
	}
	copied := *t
	copied.Normalize()
	if err := copied.Validate(); err != nil {
		return nil, 0, err
	}

	bpm := copied.Tempo
	if bpm <= 0 {
		bpm = defaultPlaybackBPM
	}
	positionDuration := time.Minute / time.Duration(bpm)
	noteDuration := time.Duration(float64(positionDuration) * 0.9)
	if noteDuration < 50*time.Millisecond {
		noteDuration = 50 * time.Millisecond
	}

	notes := make([]PlaybackNote, 0)
	var offset time.Duration
	for lineIndex, line := range copied.Lines {
		for _, measure := range line.Measures {
			for _, beat := range measure.Beats {
				base, ok := tuner.StringByNumber(beat.String)
				if !ok {
					continue
				}
				frequency := base.Frequency * math.Pow(2, float64(beat.Fret)/12)
				if beat.SlideTo == nil {
					notes = append(notes, PlaybackNote{
						Start:     offset,
						Duration:  noteDuration,
						Frequency: frequency,
					})
				} else {
					part := noteDuration / 2
					notes = append(notes,
						PlaybackNote{Start: offset, Duration: part, Frequency: frequency},
						PlaybackNote{
							Start:     offset + part,
							Duration:  noteDuration - part,
							Frequency: base.Frequency * math.Pow(2, float64(*beat.SlideTo)/12),
						},
					)
				}
			}
			offset += positionDuration
		}
		if lineIndex < len(copied.Lines)-1 && line.Pause > 0 {
			offset += time.Duration(line.Pause) * positionDuration
		}
	}
	if len(notes) == 0 {
		return nil, 0, errors.New("tab has no notes to play")
	}

	total := offset + playbackTail
	if total > MaxPlaybackDuration {
		return nil, 0, fmt.Errorf("tab is too long to play (maximum %s)", MaxPlaybackDuration)
	}
	return notes, total, nil
}

// RenderPlayback synthesises a plucked-string mix for the scheduled notes.
func RenderPlayback(notes []PlaybackNote, total time.Duration, sampleRate int) []float32 {
	if sampleRate <= 0 || total <= 0 {
		return nil
	}
	count := int(math.Round(total.Seconds() * float64(sampleRate)))
	if count <= 0 {
		return nil
	}

	samples := make([]float32, count)
	const releaseSeconds = 0.012
	for _, note := range notes {
		if note.Frequency <= 0 || note.Duration <= 0 {
			continue
		}
		start := int(math.Round(note.Start.Seconds() * float64(sampleRate)))
		length := int(math.Round(note.Duration.Seconds() * float64(sampleRate)))
		if start < 0 || start >= len(samples) {
			continue
		}
		release := int(releaseSeconds * float64(sampleRate))
		if release > length {
			release = length
		}
		for i := 0; i < length; i++ {
			index := start + i
			if index >= len(samples) {
				break
			}
			t := float64(i) / float64(sampleRate)
			envelope := (1 - math.Exp(-t/0.003)) * math.Exp(-t/0.18)
			if release > 0 && i >= length-release {
				envelope *= float64(length-i) / float64(release)
			}
			phase := 2 * math.Pi * note.Frequency * t
			wave := math.Sin(phase) + 0.3*math.Sin(2*phase) + 0.12*math.Sin(3*phase)
			samples[index] += float32(0.12 * envelope * wave)
		}
	}

	for i, sample := range samples {
		if sample > 0.95 {
			samples[i] = 0.95
		} else if sample < -0.95 {
			samples[i] = -0.95
		}
	}
	return samples
}
