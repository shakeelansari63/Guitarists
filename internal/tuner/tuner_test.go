package tuner

import (
	"math"
	"testing"
)

func sine(frequency float64, sampleRate, count int, amplitude float64) []float64 {
	samples := make([]float64, count)
	for i := range samples {
		samples[i] = amplitude * math.Sin(2*math.Pi*frequency*float64(i)/float64(sampleRate))
	}
	return samples
}

func TestDetectStandardStrings(t *testing.T) {
	t.Parallel()

	const sampleRate = 44100
	detector := NewDetector()

	for _, s := range StandardTuning {
		s := s
		t.Run(s.Note, func(t *testing.T) {
			t.Parallel()
			samples := sine(s.Frequency, sampleRate, 8192, 0.5)
			got, ok := detector.Detect(samples, sampleRate)
			if !ok {
				t.Fatalf("expected detection for %s", s.Note)
			}
			if diff := math.Abs(got.Frequency - s.Frequency); diff > 1 {
				t.Fatalf("detected %.3f Hz, want %.3f Hz (diff %.3f)", got.Frequency, s.Frequency, diff)
			}
			if got.Clarity <= 0.5 {
				t.Fatalf("clarity too low: %.3f", got.Clarity)
			}
		})
	}
}

func TestDetectSilence(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	if _, ok := detector.Detect(make([]float64, 4096), 44100); ok {
		t.Fatal("expected no detection for silence")
	}
}

func TestDetectNoise(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	noise := make([]float64, 4096)
	for i := range noise {
		noise[i] = 0.2
		if i%2 == 0 {
			noise[i] = -0.2
		}
	}
	if _, ok := detector.Detect(noise, 44100); ok {
		t.Fatal("expected no confident detection for alternating signal")
	}
}

func TestCents(t *testing.T) {
	t.Parallel()

	if got := Cents(110, 110); got != 0 {
		t.Fatalf("unison cents = %v", got)
	}
	sharp := 110 * math.Pow(2, 10.0/1200)
	if got := Cents(sharp, 110); math.Abs(got-10) > 0.001 {
		t.Fatalf("expected 10 cents, got %v", got)
	}
	flat := 110 * math.Pow(2, -25.0/1200)
	if got := Cents(flat, 110); math.Abs(got+25) > 0.001 {
		t.Fatalf("expected -25 cents, got %v", got)
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	a2, _ := StringByNumber(5)

	got, ok := Analyze(a2.Frequency, 5)
	if !ok {
		t.Fatal("expected analysis")
	}
	if !got.InTune || got.Direction != "in-tune" {
		t.Fatalf("expected in tune, got %+v", got)
	}

	sharp := a2.Frequency * math.Pow(2, 12.0/1200)
	got, _ = Analyze(sharp, 5)
	if got.InTune || got.Direction != "sharp" {
		t.Fatalf("expected sharp, got %+v", got)
	}

	auto, ok := Analyze(a2.Frequency, 0)
	if !ok || auto.Target.Number != 5 {
		t.Fatalf("auto target = %+v, ok=%v", auto.Target, ok)
	}
}

func TestAnalyzeInvalid(t *testing.T) {
	t.Parallel()

	for _, freq := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, ok := Analyze(freq, 1); ok {
			t.Fatalf("expected invalid result for %v", freq)
		}
	}
	if _, ok := Analyze(110, 7); ok {
		t.Fatal("expected invalid result for unknown string")
	}
}

func TestNearestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		frequency float64
		want      int
	}{
		{82.4, 6},
		{110.1, 5},
		{147.0, 4},
		{196.5, 3},
		{247.0, 2},
		{330.0, 1},
	}
	for _, tt := range tests {
		got, ok := NearestString(tt.frequency)
		if !ok || got.Number != tt.want {
			t.Fatalf("NearestString(%v) = %d, ok=%v; want %d", tt.frequency, got.Number, ok, tt.want)
		}
	}
}
