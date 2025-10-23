package whisper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	whispergo "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// Transcriber handles audio transcription using Whisper
type Transcriber struct {
	model   whispergo.Model
	context whispergo.Context
}

// NewTranscriber creates a new transcriber with the specified model
func NewTranscriber(modelPath string) (*Transcriber, error) {
	// Expand home directory if needed
	if strings.HasPrefix(modelPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		modelPath = filepath.Join(home, modelPath[2:])
	}

	// Load the model
	model, err := whispergo.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create context
	context, err := model.NewContext()
	if err != nil {
		model.Close()
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	return &Transcriber{
		model:   model,
		context: context,
	}, nil
}

// Transcribe converts audio samples to text
func (t *Transcriber) Transcribe(samples []float32) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("no audio samples provided")
	}

	// Process the audio data
	if err := t.context.Process(samples, nil, nil, nil); err != nil {
		return "", fmt.Errorf("failed to process audio: %w", err)
	}

	// Collect all segments into a single string
	var result strings.Builder
	for {
		segment, err := t.context.NextSegment()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", fmt.Errorf("error getting segment: %w", err)
		}

		// Trim whitespace and add to result
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(text)
		}
	}

	return result.String(), nil
}

// Close cleans up the transcriber
func (t *Transcriber) Close() error {
	if t.model != nil {
		t.model.Close()
	}
	return nil
}
