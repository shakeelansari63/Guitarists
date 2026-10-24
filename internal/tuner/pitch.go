package tuner

import "math"

// Detector performs YIN pitch detection on monophonic audio.
type Detector struct {
	MinFrequency float64
	MaxFrequency float64
	Threshold    float64
	SilenceRMS   float64
}

// NewDetector returns a detector tuned for the guitar's frequency range.
func NewDetector() *Detector {
	return &Detector{
		MinFrequency: 60,
		MaxFrequency: 1200,
		Threshold:    0.15,
		SilenceRMS:   0.0015,
	}
}

// Detect analyses samples and returns a detection when a confident pitch is found.
func (d *Detector) Detect(samples []float64, sampleRate int) (Detection, bool) {
	if d == nil {
		d = NewDetector()
	}
	if sampleRate <= 0 || len(samples) < 64 {
		return Detection{}, false
	}

	mean := 0.0
	for _, v := range samples {
		mean += v
	}
	mean /= float64(len(samples))

	rms := 0.0
	x := make([]float64, len(samples))
	for i, v := range samples {
		v -= mean
		x[i] = v
		rms += v * v
	}
	rms = math.Sqrt(rms / float64(len(samples)))
	if rms < d.SilenceRMS {
		return Detection{RMS: rms}, false
	}

	maxLag := int(float64(sampleRate) / d.MinFrequency)
	if maxLag >= len(x)/2 {
		maxLag = len(x)/2 - 1
	}
	minLag := int(float64(sampleRate) / d.MaxFrequency)
	if minLag < 2 {
		minLag = 2
	}
	if maxLag <= minLag+2 {
		return Detection{RMS: rms}, false
	}

	diff := make([]float64, maxLag+1)
	for tau := 1; tau <= maxLag; tau++ {
		sum := 0.0
		limit := len(x) - tau
		for i := 0; i < limit; i++ {
			delta := x[i] - x[i+tau]
			sum += delta * delta
		}
		diff[tau] = sum
	}

	cmnd := make([]float64, maxLag+1)
	cmnd[0] = 1
	running := 0.0
	for tau := 1; tau <= maxLag; tau++ {
		running += diff[tau]
		if running == 0 {
			cmnd[tau] = 1
		} else {
			cmnd[tau] = diff[tau] * float64(tau) / running
		}
	}

	threshold := d.Threshold
	if threshold <= 0 {
		threshold = 0.15
	}

	bestTau := -1
	for tau := minLag; tau <= maxLag; tau++ {
		if cmnd[tau] < threshold {
			for tau+1 <= maxLag && cmnd[tau+1] < cmnd[tau] {
				tau++
			}
			bestTau = tau
			break
		}
	}
	if bestTau == -1 {
		minValue := math.Inf(1)
		for tau := minLag; tau <= maxLag; tau++ {
			if cmnd[tau] < minValue {
				minValue = cmnd[tau]
				bestTau = tau
			}
		}
	}
	if bestTau < 1 {
		return Detection{RMS: rms}, false
	}

	refined := float64(bestTau)
	if bestTau > 0 && bestTau < maxLag {
		y0 := cmnd[bestTau-1]
		y1 := cmnd[bestTau]
		y2 := cmnd[bestTau+1]
		denom := y0 - 2*y1 + y2
		if denom != 0 {
			shift := 0.5 * (y0 - y2) / denom
			if math.Abs(shift) < 1 {
				refined += shift
			}
		}
	}
	if refined <= 0 {
		return Detection{RMS: rms}, false
	}

	frequency := float64(sampleRate) / refined
	if frequency < d.MinFrequency || frequency > d.MaxFrequency {
		return Detection{RMS: rms}, false
	}

	clarity := 1 - cmnd[bestTau]
	if clarity < 0 {
		clarity = 0
	}
	if clarity > 1 {
		clarity = 1
	}
	if clarity < 0.5 {
		return Detection{Frequency: frequency, Clarity: clarity, RMS: rms}, false
	}

	return Detection{Frequency: frequency, Clarity: clarity, RMS: rms}, true
}
