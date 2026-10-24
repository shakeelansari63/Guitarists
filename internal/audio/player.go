package audio

import (
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

const maxQueuedSamples = SampleRate * 2

// Player plays short click sounds through the default output device.
type Player struct {
	ctx       *malgo.AllocatedContext
	device    *malgo.Device
	mu        sync.Mutex
	queue     []float32
	head      int
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

// NewPlayer opens the default playback device.
func NewPlayer() (*Player, error) {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, errors.New("initialising audio system: " + err.Error())
	}

	p := &Player{ctx: mctx}
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

func (p *Player) fill(output []byte, frames int) {
	p.mu.Lock()
	for i := 0; i < frames; i++ {
		var sample float32
		if p.head < len(p.queue) {
			sample = p.queue[p.head]
			p.head++
		}
		if p.head >= len(p.queue) {
			p.queue = p.queue[:0]
			p.head = 0
		}
		if (i+1)*4 <= len(output) {
			binary.LittleEndian.PutUint32(output[i*4:], math.Float32bits(sample))
		}
	}
	p.mu.Unlock()
}

// Play queues one click. Accented clicks are louder and higher pitched.
func (p *Player) Play(accent bool) {
	samples := clickSamples(accent)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	if len(p.queue)-p.head+len(samples) > maxQueuedSamples {
		return
	}
	if p.head > 0 {
		p.queue = append(p.queue[:0], p.queue[p.head:]...)
		p.head = 0
	}
	p.queue = append(p.queue, samples...)
}

// Close stops playback and releases the audio device.
func (p *Player) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.queue = nil
		p.head = 0
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

// clickSamples synthesises a short percussive click.
func clickSamples(accent bool) []float32 {
	duration := 0.045
	frequency := 1000.0
	amplitude := 0.5
	if accent {
		frequency = 1600.0
		amplitude = 0.8
	}
	count := int(float64(SampleRate) * duration)
	samples := make([]float32, count)
	decay := SampleRate * 0.009
	for i := range samples {
		t := float64(i) / SampleRate
		env := math.Exp(-float64(i) / decay)
		wave := math.Sin(2 * math.Pi * frequency * t)
		samples[i] = float32(amplitude * env * wave)
	}
	return samples
}
