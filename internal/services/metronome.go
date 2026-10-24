package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/freak/guitarists/internal/audio"
	"github.com/freak/guitarists/internal/metronome"
)

// MetronomeService schedules clicks and exposes tempo controls.
type MetronomeService struct {
	mu      sync.RWMutex
	config  metronome.Config
	player  *audio.Player
	cancel  context.CancelFunc
	running bool
	change  chan struct{}
	taps    *metronome.TapTempo
}

// MetronomeState describes the current metronome configuration.
type MetronomeState struct {
	Config  metronome.Config `json:"config"`
	Running bool             `json:"running"`
}

// BeatEvent is emitted on the "metronome:beat" event.
type BeatEvent struct {
	metronome.Beat
	Timestamp int64 `json:"timestamp"`
}

// NewMetronomeService creates a metronome service with default settings.
func NewMetronomeService() *MetronomeService {
	return &MetronomeService{
		config: metronome.DefaultConfig(),
		change: make(chan struct{}, 1),
		taps:   metronome.NewTapTempo(),
	}
}

// ServiceName implements application.ServiceName.
func (s *MetronomeService) ServiceName() string { return "MetronomeService" }

// State returns the current configuration and running state.
func (s *MetronomeService) State() MetronomeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return MetronomeState{Config: s.config, Running: s.running}
}

// SetConfig validates and applies a new configuration. Running metronomes
// adopt the new tempo without restarting.
func (s *MetronomeService) SetConfig(config metronome.Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	s.config = config
	s.mu.Unlock()
	s.signalChange()
	emit("metronome:state", s.State())
	return nil
}

// Start begins the metronome.
func (s *MetronomeService) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	if err := s.config.Validate(); err != nil {
		s.mu.Unlock()
		return err
	}
	if s.player == nil {
		player, err := audio.NewPlayer()
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("starting metronome: %w", err)
		}
		s.player = player
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	select {
	case <-s.change:
	default:
	}

	go s.loop(ctx)
	s.emitState()
	return nil
}

// Stop stops the metronome.
func (s *MetronomeService) Stop() error {
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

// Tap records a tap and updates the tempo once two taps have been received.
func (s *MetronomeService) Tap() (int, error) {
	bpm, ok := s.taps.Tap(time.Now())
	if !ok {
		return 0, metronome.ErrNotEnoughTaps
	}

	s.mu.Lock()
	config := s.config
	config.BPM = bpm
	s.config = config
	s.mu.Unlock()

	s.signalChange()
	emit("metronome:state", s.State())
	return bpm, nil
}

// ServiceShutdown stops the metronome and releases the audio device.
func (s *MetronomeService) ServiceShutdown() error {
	_ = s.Stop()
	s.mu.Lock()
	player := s.player
	s.player = nil
	s.mu.Unlock()
	if player != nil {
		return player.Close()
	}
	return nil
}

func (s *MetronomeService) emitState() {
	emit("metronome:state", s.State())
}

func (s *MetronomeService) signalChange() {
	select {
	case s.change <- struct{}{}:
	default:
	}
}

func (s *MetronomeService) loop(ctx context.Context) {
	s.mu.RLock()
	config := s.config
	player := s.player
	s.mu.RUnlock()

	interval := config.Interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	index := 0
	s.fire(index, config, player)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.change:
			s.mu.RLock()
			config = s.config
			player = s.player
			s.mu.RUnlock()
			if next := config.Interval(); next > 0 {
				interval = next
				ticker.Reset(interval)
			}
		case <-ticker.C:
			index++
			s.mu.RLock()
			config = s.config
			player = s.player
			s.mu.RUnlock()
			s.fire(index, config, player)
		}
	}
}

func (s *MetronomeService) fire(index int, config metronome.Config, player *audio.Player) {
	beat := config.BeatForIndex(index)
	if player != nil {
		player.Play(beat.Accent)
	}
	emit("metronome:beat", BeatEvent{Beat: beat, Timestamp: time.Now().UnixMilli()})
}
