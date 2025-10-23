package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	SampleRate = 16000 // Whisper requires 16kHz
	Channels   = 1     // Mono
)

// Recorder handles audio recording from microphone
type Recorder struct {
	stream   *portaudio.Stream
	buffer   []float32
	mu       sync.Mutex
	isActive bool
}

// NewRecorder creates a new audio recorder
func NewRecorder() (*Recorder, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize PortAudio: %w", err)
	}

	return &Recorder{
		buffer: make([]float32, 0),
	}, nil
}

// Start begins recording audio
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isActive {
		return fmt.Errorf("already recording")
	}

	// Clear previous buffer
	r.buffer = make([]float32, 0)

	// Create input stream
	stream, err := portaudio.OpenDefaultStream(Channels, 0, float64(SampleRate), 0, func(in []float32) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.buffer = append(r.buffer, in...)
	})
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start stream: %w", err)
	}

	r.stream = stream
	r.isActive = true
	return nil
}

// Stop stops recording and returns the audio buffer
func (r *Recorder) Stop() ([]float32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isActive {
		return nil, fmt.Errorf("not recording")
	}

	if err := r.stream.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop stream: %w", err)
	}

	if err := r.stream.Close(); err != nil {
		return nil, fmt.Errorf("failed to close stream: %w", err)
	}

	r.stream = nil
	r.isActive = false

	// Return copy of buffer
	result := make([]float32, len(r.buffer))
	copy(result, r.buffer)
	return result, nil
}

// IsRecording returns true if currently recording
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isActive
}

// Close cleans up the recorder
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stream != nil {
		if r.isActive {
			r.stream.Stop()
		}
		r.stream.Close()
	}

	return portaudio.Terminate()
}
