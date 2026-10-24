package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

// Capture streams mono float32 microphone samples.
type Capture struct {
	ctx       *malgo.AllocatedContext
	device    *malgo.Device
	frames    chan []float32
	closeOnce sync.Once
	closeErr  error
}

// StartCapture opens the default microphone and begins streaming samples.
// Closing ctx stops the capture.
func StartCapture(ctx context.Context) (*Capture, error) {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, errors.New("initialising audio system: " + err.Error())
	}

	c := &Capture{
		ctx:    mctx,
		frames: make(chan []float32, 32),
	}

	config := malgo.DefaultDeviceConfig(malgo.Capture)
	config.Capture.Format = malgo.FormatF32
	config.Capture.Channels = 1
	config.SampleRate = SampleRate
	config.Alsa.NoMMap = 1

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frameCount uint32) {
			if len(input) == 0 || frameCount == 0 {
				return
			}
			n := int(frameCount)
			if max := len(input) / 4; n > max {
				n = max
			}
			samples := make([]float32, n)
			for i := 0; i < n; i++ {
				bits := binary.LittleEndian.Uint32(input[i*4:])
				samples[i] = math.Float32frombits(bits)
			}
			select {
			case c.frames <- samples:
			default:
			}
		},
	}

	device, err := malgo.InitDevice(mctx.Context, config, callbacks)
	if err != nil {
		mctx.Uninit()
		mctx.Free()
		return nil, errors.New("opening microphone: " + err.Error())
	}
	c.device = device

	if err := device.Start(); err != nil {
		device.Uninit()
		mctx.Uninit()
		mctx.Free()
		return nil, errors.New("starting microphone: " + err.Error())
	}

	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	return c, nil
}

// Frames returns the channel of captured mono sample blocks.
func (c *Capture) Frames() <-chan []float32 {
	return c.frames
}

// Close stops the microphone and releases its resources.
func (c *Capture) Close() error {
	c.closeOnce.Do(func() {
		if c.device != nil {
			if err := c.device.Stop(); err != nil {
				c.closeErr = err
			}
			c.device.Uninit()
		}
		if c.ctx != nil {
			if err := c.ctx.Uninit(); err != nil && c.closeErr == nil {
				c.closeErr = err
			}
			c.ctx.Free()
		}
	})
	return c.closeErr
}
