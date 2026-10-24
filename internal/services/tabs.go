package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/freak/guitarists/internal/audio"
	"github.com/freak/guitarists/internal/platform"
	"github.com/freak/guitarists/internal/tabs"
)

// TabPlaybackState is emitted on the "tabs:playback" event.
// Line is -1 when every line is playing.
type TabPlaybackState struct {
	Playing bool `json:"playing"`
	Line    int  `json:"line"`
}

// TabService loads and saves tablature files in the documents directory.
type TabService struct {
	store *tabs.Store

	mu      sync.Mutex
	player  *audio.ClipPlayer
	playing bool
	line    int
	cancel  context.CancelFunc
	playID  uint64
}

// NewTabService creates the tab service and ensures the storage directory exists.
func NewTabService() (*TabService, error) {
	dir, err := platform.TabsDir()
	if err != nil {
		return nil, fmt.Errorf("preparing tabs storage: %w", err)
	}
	store, err := tabs.NewStore(dir)
	if err != nil {
		return nil, fmt.Errorf("preparing tabs storage: %w", err)
	}
	return &TabService{store: store, line: -1}, nil
}

// ServiceName implements application.ServiceName.
func (s *TabService) ServiceName() string { return "TabService" }

// Directory returns the absolute directory where tabs are stored.
func (s *TabService) Directory() string {
	return s.store.Dir()
}

// New returns an empty tab document with default settings.
func (s *TabService) New() tabs.Tab {
	return tabs.New()
}

// List returns stored tabs sorted by most recently modified.
func (s *TabService) List() ([]tabs.Summary, error) {
	return s.store.List()
}

// Load reads a tab by file name.
func (s *TabService) Load(name string) (tabs.Tab, error) {
	tab, err := s.store.Load(name)
	if err != nil {
		return tabs.Tab{}, err
	}
	return *tab, nil
}

// Save validates and writes a tab. The .xml extension is added when missing.
func (s *TabService) Save(name string, tab tabs.Tab) (tabs.Summary, error) {
	return s.store.Save(name, &tab)
}

// Delete removes a stored tab.
func (s *TabService) Delete(name string) error {
	return s.store.Delete(name)
}

// Play renders the supplied tab and starts playback in the background.
func (s *TabService) Play(tab tabs.Tab) error {
	return s.startPlayback(tab, -1)
}

// PlayLine plays a single zero-based line from the supplied tab.
func (s *TabService) PlayLine(tab tabs.Tab, line int) error {
	source := tab
	source.Normalize()
	if err := source.Validate(); err != nil {
		return err
	}
	if line < 0 || line >= len(source.Lines) {
		return fmt.Errorf("line must be between 1 and %d", len(source.Lines))
	}
	source.Lines = []tabs.Line{source.Lines[line]}
	return s.startPlayback(source, line)
}

func (s *TabService) startPlayback(tab tabs.Tab, line int) error {
	source := tab
	source.Normalize()
	if err := source.Validate(); err != nil {
		return err
	}
	notes, total, err := source.PlaybackNotes()
	if err != nil {
		return err
	}
	samples := tabs.RenderPlayback(notes, total, audio.SampleRate)

	s.mu.Lock()
	if s.player == nil {
		player, err := audio.NewClipPlayer()
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("starting tab playback: %w", err)
		}
		s.player = player
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.playID++
	playID := s.playID
	s.playing = true
	s.line = line
	player := s.player
	s.mu.Unlock()

	player.Play(samples)
	emit("tabs:playback", TabPlaybackState{Playing: true, Line: line})
	go s.watchPlayback(playID, ctx, total)
	return nil
}

// Stop stops tab playback.
func (s *TabService) Stop() error {
	s.mu.Lock()
	if !s.playing && s.cancel == nil {
		s.mu.Unlock()
		return nil
	}
	s.playing = false
	s.line = -1
	s.playID++
	cancel := s.cancel
	s.cancel = nil
	player := s.player
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if player != nil {
		player.Stop()
	}
	emit("tabs:playback", TabPlaybackState{Playing: false, Line: -1})
	return nil
}

// PlaybackState returns whether a tab is currently playing.
func (s *TabService) PlaybackState() TabPlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return TabPlaybackState{Playing: s.playing, Line: s.line}
}

// ServiceShutdown stops playback and releases the audio device.
func (s *TabService) ServiceShutdown() error {
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

func (s *TabService) watchPlayback(id uint64, ctx context.Context, total time.Duration) {
	timer := time.NewTimer(total)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	s.mu.Lock()
	if s.playID != id {
		s.mu.Unlock()
		return
	}
	s.playing = false
	s.line = -1
	s.cancel = nil
	s.mu.Unlock()
	emit("tabs:playback", TabPlaybackState{Playing: false, Line: -1})
}
