package audio

import (
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

// ClipPlayer streams one mono float32 clip through the default output device.
type ClipPlayer struct {
	ctx       *malgo.AllocatedContext
	device    *malgo.Device
	mu        sync.Mutex
	samples   []float32
	position  int
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

// NewClipPlayer opens the default playback device.
func NewClipPlayer() (*ClipPlayer, error) {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, errors.New("initialising audio system: " + err.Error())
	}

	p := &ClipPlayer{ctx: mctx}
	config := malgo.DefaultDeviceConfig(malgo.Playback)
	config.Playback.Format = malgo.FormatF32
	config.Playback.Channels = 1
	config.SampleRate = SampleRate
	config.Alsa.NoMMap = 1

	callbacks := malgo.DeviceCallbacks{
		Data: func(output, _ []byte, frameCount uint32) {
			p.fill(output, int(frameCount))
		},
	}

	device, err := malgo.InitDevice(mctx.Context, config, callbacks)
	if err != nil {
		mctx.Uninit()
		mctx.Free()
		return nil, errors.New("opening audio output: " + err.Error())
	}
	p.device = device

	if err := device.Start(); err != nil {
		device.Uninit()
		mctx.Uninit()
		mctx.Free()
		return nil, errors.New("starting audio output: " + err.Error())
	}
	return p, nil
}

func (p *ClipPlayer) fill(output []byte, frames int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := 0; i < frames; i++ {
		var sample float32
		if p.position < len(p.samples) {
			sample = p.samples[p.position]
			p.position++
		}
		if (i+1)*4 <= len(output) {
			binary.LittleEndian.PutUint32(output[i*4:], math.Float32bits(sample))
		}
	}
}

// Play replaces the current clip and starts from its beginning.
func (p *ClipPlayer) Play(samples []float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.samples = samples
	p.position = 0
}

// Stop silences playback and clears the current clip.
func (p *ClipPlayer) Stop() {
	p.mu.Lock()
	p.samples = nil
	p.position = 0
	p.mu.Unlock()
}

// Close stops playback and releases the audio device.
func (p *ClipPlayer) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.samples = nil
		p.position = 0
		p.mu.Unlock()

		if p.device != nil {
			if err := p.device.Stop(); err != nil {
				p.closeErr = err
			}
			p.device.Uninit()
		}
		if p.ctx != nil {
			if err := p.ctx.Uninit(); err != nil && p.closeErr == nil {
				p.closeErr = err
			}
			p.ctx.Free()
		}
	})
	return p.closeErr
}
