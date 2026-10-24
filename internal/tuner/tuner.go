// Package tuner contains guitar tuning and pitch-detection logic.
// It has no dependency on Wails or any OS-specific audio API.
package tuner

import (
	"math"
)

// InTuneTolerance is the maximum absolute cents deviation considered in tune.
const InTuneTolerance = 5.0

// GuitarString describes one string of a standard-tuned guitar.
type GuitarString struct {
	Number    int     `json:"number"`
	Note      string  `json:"note"`
	Frequency float64 `json:"frequency"`
}

// StandardTuning lists the strings from high E (1) to low E (6).
var StandardTuning = []GuitarString{
	{Number: 1, Note: "E4", Frequency: 329.6276},
	{Number: 2, Note: "B3", Frequency: 246.9417},
	{Number: 3, Note: "G3", Frequency: 195.9977},
	{Number: 4, Note: "D3", Frequency: 146.8324},
	{Number: 5, Note: "A2", Frequency: 110.0000},
	{Number: 6, Note: "E2", Frequency: 82.4069},
}

// Detection is the raw result of pitch analysis.
type Detection struct {
	Frequency float64 `json:"frequency"`
	Clarity   float64 `json:"clarity"`
	RMS       float64 `json:"rms"`
}

// Result describes how a detected pitch relates to a target string.
type Result struct {
	Target     GuitarString `json:"target"`
	Cents      float64      `json:"cents"`
	InTune     bool         `json:"inTune"`
	Direction  string       `json:"direction"`
	DetectedHz float64      `json:"detectedHz"`
}

// StringByNumber returns the standard string with the given number.
func StringByNumber(number int) (GuitarString, bool) {
	for _, s := range StandardTuning {
		if s.Number == number {
			return s, true
		}
	}
	return GuitarString{}, false
}

// Cents returns the deviation in cents between two frequencies.
func Cents(frequency, target float64) float64 {
	if frequency <= 0 || target <= 0 {
		return 0
	}
	return 1200 * math.Log2(frequency/target)
}

// NearestString returns the standard string closest in pitch to frequency.
func NearestString(frequency float64) (GuitarString, bool) {
	if frequency <= 0 {
		return GuitarString{}, false
	}
	best := StandardTuning[0]
	bestDist := math.Abs(Cents(frequency, best.Frequency))
	for _, s := range StandardTuning[1:] {
		if d := math.Abs(Cents(frequency, s.Frequency)); d < bestDist {
			best, bestDist = s, d
		}
	}
	return best, true
}

// Analyze compares a detected frequency with the requested string.
// target 0 means auto-detect the closest string.
func Analyze(frequency float64, target int) (Result, bool) {
	if frequency <= 0 || math.IsNaN(frequency) || math.IsInf(frequency, 0) {
		return Result{}, false
	}

	var chosen GuitarString
	if target == 0 {
		var ok bool
		chosen, ok = NearestString(frequency)
		if !ok {
			return Result{}, false
		}
	} else {
		var ok bool
		chosen, ok = StringByNumber(target)
		if !ok {
			return Result{}, false
		}
	}

	cents := Cents(frequency, chosen.Frequency)
	inTune := math.Abs(cents) <= InTuneTolerance
	direction := "in-tune"
	if cents < -InTuneTolerance {
		direction = "flat"
	} else if cents > InTuneTolerance {
		direction = "sharp"
	}

	return Result{
		Target:     chosen,
		Cents:      cents,
		InTune:     inTune,
		Direction:  direction,
		DetectedHz: frequency,
	}, true
}
