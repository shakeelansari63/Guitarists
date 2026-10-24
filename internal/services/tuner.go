package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/freak/guitarists/internal/audio"
	"github.com/freak/guitarists/internal/tuner"
)

// TunerService exposes microphone pitch detection to the frontend.
type TunerService struct {
	mu       sync.RWMutex
	detector *tuner.Detector
	cancel   context.CancelFunc
	running  bool
	target   int
}

// TunerState describes the current tuner configuration.
type TunerState struct {
	Running      bool `json:"running"`
	TargetString int  `json:"targetString"`
}

// TunerUpdate is emitted on the "tuner:update" event.
type TunerUpdate struct {
	HasSignal    bool    `json:"hasSignal"`
	Frequency    float64 `json:"frequency"`
	Clarity      float64 `json:"clarity"`
	RMS          float64 `json:"rms"`
	StringNumber int     `json:"stringNumber"`
	Note         string  `json:"note"`
	TargetHz     float64 `json:"targetHz"`
	Cents        float64 `json:"cents"`
	InTune       bool    `json:"inTune"`
	Direction    string  `json:"direction"`
	AutoDetect   bool    `json:"autoDetect"`
}

// NewTunerService creates a tuner service.
func NewTunerService() *TunerService {
	return &TunerService{detector: tuner.NewDetector()}
}

// ServiceName implements application.ServiceName.
func (s *TunerService) ServiceName() string { return "TunerService" }

// Strings returns the supported standard tuning.
func (s *TunerService) Strings() []tuner.GuitarString {
	out := make([]tuner.GuitarString, len(tuner.StandardTuning))
	copy(out, tuner.StandardTuning)
	return out
}

// State returns the current running state and target string.
func (s *TunerService) State() TunerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return TunerState{Running: s.running, TargetString: s.target}
}

// SetTargetString selects a string (1-6) or 0 for auto-detect.
func (s *TunerService) SetTargetString(number int) error {
	if number < 0 || number > 6 {
		return errors.New("string must be between 0 (auto) and 6")
	}
	s.mu.Lock()
	s.target = number
	s.mu.Unlock()
	s.emitState()
	return nil
}

// Start begins capturing audio from the default microphone.
func (s *TunerService) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	capture, err := audio.StartCapture(ctx)
	if err != nil {
		cancel()
		s.mu.Lock()
		s.running = false
		s.cancel = nil
		s.mu.Unlock()
		s.emitState()
		return fmt.Errorf("starting tuner: %w", err)
	}

	go s.loop(ctx, capture)
	s.emitState()
	return nil
}

// Stop stops microphone capture.
func (s *TunerService) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.running = false
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.emitState()
	return nil
}

// ServiceShutdown stops the tuner when the application exits.
func (s *TunerService) ServiceShutdown() error {
	return s.Stop()
}

func (s *TunerService) loop(ctx context.Context, capture *audio.Capture) {
	defer capture.Close()

	buffer := make([]float64, 0, audio.FrameSize)
	for {
		select {
		case <-ctx.Done():
			return
		case block, ok := <-capture.Frames():
			if !ok {
				return
			}
			for _, sample := range block {
				buffer = append(buffer, float64(sample))
			}
			for len(buffer) >= audio.FrameSize {
				frame := make([]float64, audio.FrameSize)
				copy(frame, buffer[:audio.FrameSize])
				buffer = append(buffer[:0], buffer[audio.FrameSize:]...)
				s.analyze(frame)
			}
		}
	}
}

func (s *TunerService) analyze(frame []float64) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()

	detection, ok := s.detector.Detect(frame, audio.SampleRate)
	update := TunerUpdate{
		HasSignal:  ok,
		Frequency:  detection.Frequency,
		Clarity:    detection.Clarity,
		RMS:        detection.RMS,
		AutoDetect: target == 0,
	}
	if ok {
		result, valid := tuner.Analyze(detection.Frequency, target)
		if valid {
			update.StringNumber = result.Target.Number
			update.Note = result.Target.Note
			update.TargetHz = result.Target.Frequency
			update.Cents = result.Cents
			update.InTune = result.InTune
			update.Direction = result.Direction
		}
	}
	emit("tuner:update", update)
}

func (s *TunerService) emitState() {
	emit("tuner:state", s.State())
}
