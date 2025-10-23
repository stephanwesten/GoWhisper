package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate = 16000 // Whisper requires 16kHz
	channels   = 1     // Mono
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test_audio_record.go <duration_seconds> <output.wav>")
		fmt.Println("Example: go run test_audio_record.go 5 test.wav")
		os.Exit(1)
	}

	var duration int
	outputFile := os.Args[2]
	fmt.Sscanf(os.Args[1], "%d", &duration)

	log.Printf("Recording for %d seconds to %s...", duration, outputFile)
	log.Println("Speak into your microphone!")

	// Initialize PortAudio
	if err := portaudio.Initialize(); err != nil {
		log.Fatalf("Failed to initialize PortAudio: %v", err)
	}
	defer portaudio.Terminate()

	// Buffer to store audio samples
	buffer := make([]float32, 0)

	// Create input stream
	stream, err := portaudio.OpenDefaultStream(channels, 0, float64(sampleRate), 0, func(in []float32) {
		// Copy samples to buffer
		buffer = append(buffer, in...)
	})
	if err != nil {
		log.Fatalf("Failed to open stream: %v", err)
	}
	defer stream.Close()

	// Start recording
	if err := stream.Start(); err != nil {
		log.Fatalf("Failed to start stream: %v", err)
	}

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Record for specified duration or until interrupt
	ticker := time.NewTicker(time.Duration(duration) * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		log.Println("Recording time completed")
	case <-sigChan:
		log.Println("\nRecording interrupted")
	}

	// Stop recording
	if err := stream.Stop(); err != nil {
		log.Fatalf("Failed to stop stream: %v", err)
	}

	log.Printf("Recorded %d samples (%.2f seconds)", len(buffer), float64(len(buffer))/float64(sampleRate))

	// Save to WAV file
	if err := saveWAV(outputFile, buffer, sampleRate, channels); err != nil {
		log.Fatalf("Failed to save WAV: %v", err)
	}

	log.Printf("Audio saved to %s", outputFile)
	log.Println("✅ Test completed successfully!")
}

// saveWAV saves audio samples to a WAV file
func saveWAV(filename string, samples []float32, sampleRate, channels int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Convert float32 samples to int16
	int16Samples := make([]int16, len(samples))
	for i, sample := range samples {
		// Clamp and convert to int16
		val := sample * 32767.0
		if val > 32767 {
			val = 32767
		} else if val < -32768 {
			val = -32768
		}
		int16Samples[i] = int16(val)
	}

	// Calculate sizes
	dataSize := len(int16Samples) * 2 // 2 bytes per sample
	fileSize := 36 + dataSize

	// Write WAV header
	// RIFF chunk
	f.WriteString("RIFF")
	binary.Write(f, binary.LittleEndian, uint32(fileSize))
	f.WriteString("WAVE")

	// fmt chunk
	f.WriteString("fmt ")
	binary.Write(f, binary.LittleEndian, uint32(16))                      // Subchunk1Size
	binary.Write(f, binary.LittleEndian, uint16(1))                       // AudioFormat (PCM)
	binary.Write(f, binary.LittleEndian, uint16(channels))                // NumChannels
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))              // SampleRate
	binary.Write(f, binary.LittleEndian, uint32(sampleRate*channels*2))   // ByteRate
	binary.Write(f, binary.LittleEndian, uint16(channels*2))              // BlockAlign
	binary.Write(f, binary.LittleEndian, uint16(16))                      // BitsPerSample

	// data chunk
	f.WriteString("data")
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	// Write samples
	return binary.Write(f, binary.LittleEndian, int16Samples)
}
