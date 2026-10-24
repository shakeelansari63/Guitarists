// Package audio provides microphone capture and metronome click playback.
// malgo/miniaudio abstracts the platform audio backends, so no OS-specific
// files are required while the device layer remains isolated here.
package audio

const (
	// SampleRate is the sample rate used by capture and playback.
	SampleRate = 44100
	// FrameSize is the number of samples analysed by the tuner at once.
	FrameSize = 4096
)
