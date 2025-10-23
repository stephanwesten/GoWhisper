package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test_whisper.go <model_path> <audio_path>")
		fmt.Println("Example: go run test_whisper.go ~/.go-whisper/models/ggml-small.en.bin /tmp/whisper.cpp/samples/jfk.wav")
		os.Exit(1)
	}

	modelPath := os.Args[1]
	audioPath := os.Args[2]

	// Load the model
	log.Printf("Loading model from: %s", modelPath)
	model, err := whisper.New(modelPath)
	if err != nil {
		log.Fatalf("Failed to load model: %v", err)
	}
	defer model.Close()

	log.Println("Model loaded successfully!")

	// Create processing context
	context, err := model.NewContext()
	if err != nil {
		log.Fatalf("Failed to create context: %v", err)
	}

	// Open the WAV file
	log.Printf("Loading audio file: %s", audioPath)
	fh, err := os.Open(audioPath)
	if err != nil {
		log.Fatalf("Failed to open audio file: %v", err)
	}
	defer fh.Close()

	// Decode the WAV file
	dec := wav.NewDecoder(fh)
	buf, err := dec.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Failed to decode WAV: %v", err)
	}

	if dec.SampleRate != whisper.SampleRate {
		log.Fatalf("Unsupported sample rate: %d (expected %d)", dec.SampleRate, whisper.SampleRate)
	}
	if dec.NumChans != 1 {
		log.Fatalf("Unsupported number of channels: %d (expected 1)", dec.NumChans)
	}

	data := buf.AsFloat32Buffer().Data
	log.Printf("Loaded %d samples at %dHz", len(data), dec.SampleRate)

	// Process the audio data
	log.Println("Processing audio...")
	if err := context.Process(data, nil, nil, nil); err != nil {
		log.Fatalf("Failed to process audio: %v", err)
	}

	// Get transcription
	log.Println("\n=== Transcription ===")
	for {
		segment, err := context.NextSegment()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error getting segment: %v", err)
		}
		fmt.Printf("[%6s->%6s] %s\n",
			segment.Start.String(),
			segment.End.String(),
			segment.Text)
	}

	log.Println("\n=== Test completed successfully! ===")
}
