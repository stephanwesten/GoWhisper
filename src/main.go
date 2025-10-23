package main

import (
	"log"
	"os/exec"

	"github.com/getlantern/systray"
	"github.com/stephanwesten/go-whisper/src/audio"
	"github.com/stephanwesten/go-whisper/src/whisper"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

var (
	recorder    *audio.Recorder
	transcriber *whisper.Transcriber
	mStatus     *systray.MenuItem
)

func main() {
	mainthread.Init(fn)
}

func fn() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set the menu bar icon
	systray.SetIcon(getIcon())
	systray.SetTitle("GoWhisper")
	systray.SetTooltip("Voice to Terminal")

	// Initialize audio recorder
	var err error
	recorder, err = audio.NewRecorder()
	if err != nil {
		log.Fatalf("Failed to initialize recorder: %v", err)
	}

	// Initialize Whisper transcriber
	transcriber, err = whisper.NewTranscriber("~/.go-whisper/models/ggml-small.en.bin")
	if err != nil {
		log.Fatalf("Failed to initialize transcriber: %v", err)
	}
	log.Println("Whisper model loaded successfully")

	// Add menu items
	mStatus = systray.AddMenuItem("Ready", "Current status")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Register global hotkey: Cmd+Shift+H
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeyH)
	if err := hk.Register(); err != nil {
		log.Printf("Failed to register hotkey: %v", err)
	} else {
		log.Println("Hotkey registered: Cmd+Shift+H")
	}

	// Handle hotkey with channel to process one at a time
	triggerCh := make(chan struct{}, 1)

	// Collect hotkey events (may fire multiple times)
	go func() {
		for {
			<-hk.Keydown()
			// Try to send, but don't block if channel is full
			select {
			case triggerCh <- struct{}{}:
			default:
			}
		}
	}()

	// Process triggers one at a time
	go func() {
		for range triggerCh {
			handleHotkey()
		}
	}()

	// Handle menu actions
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				log.Println("Quit clicked")
				hk.Unregister()
				systray.Quit()
			}
		}
	}()
}

func handleHotkey() {
	if recorder.IsRecording() {
		// Stop recording and transcribe
		log.Println("Stopping recording...")
		mStatus.SetTitle("Processing...")

		samples, err := recorder.Stop()
		if err != nil {
			log.Printf("Error stopping recording: %v", err)
			mStatus.SetTitle("Error: Failed to stop recording")
			return
		}

		log.Printf("Recorded %d samples (%.2f seconds)", len(samples), float64(len(samples))/float64(audio.SampleRate))

		if len(samples) < audio.SampleRate/2 { // Less than 0.5 seconds
			log.Println("Recording too short, ignoring")
			mStatus.SetTitle("Ready")
			return
		}

		// Transcribe
		log.Println("Transcribing...")
		mStatus.SetTitle("Transcribing...")

		text, err := transcriber.Transcribe(samples)
		if err != nil {
			log.Printf("Error transcribing: %v", err)
			mStatus.SetTitle("Error: Transcription failed")
			return
		}

		log.Printf("Transcription: %s", text)

		if text == "" {
			log.Println("No speech detected")
			mStatus.SetTitle("Ready")
			return
		}

		// Send to active window
		mStatus.SetTitle("Typing...")
		if err := sendTextToActiveWindow(text); err != nil {
			log.Printf("Error sending text: %v", err)
			mStatus.SetTitle("Error: Failed to type")
			return
		}

		log.Println("Successfully sent transcribed text")
		mStatus.SetTitle("Ready")

	} else {
		// Start recording
		log.Println("Starting recording...")
		mStatus.SetTitle("🎤 Recording...")

		if err := recorder.Start(); err != nil {
			log.Printf("Error starting recording: %v", err)
			mStatus.SetTitle("Error: Failed to start")
			return
		}

		log.Println("Recording started - press Cmd+Shift+H again to stop")
	}
}

func onExit() {
	// Cleanup when app exits
	log.Println("Cleaning up...")
	if recorder != nil {
		recorder.Close()
	}
	if transcriber != nil {
		transcriber.Close()
	}
	log.Println("GoWhisper menu bar app exiting")
}

// sendTextToActiveWindow sends text to the currently active window using AppleScript
func sendTextToActiveWindow(text string) error {
	// AppleScript to send keystrokes to the frontmost application
	script := `
		tell application "System Events"
			keystroke "` + text + `"
		end tell
	`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("AppleScript output: %s", string(output))
		return err
	}

	log.Printf("Successfully sent text: %s", text)
	return nil
}

// getIcon returns a simple icon for the menu bar
func getIcon() []byte {
	// Simple PNG icon data (black circle on transparent background)
	// TODO: Replace with proper microphone icon
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0xf3, 0xff, 0x61, 0x00, 0x00, 0x00,
		0x3e, 0x49, 0x44, 0x41, 0x54, 0x38, 0x8d, 0x63, 0x60, 0x18, 0x05, 0xa3,
		0x60, 0x14, 0x8c, 0x82, 0x51, 0x30, 0x0a, 0x46, 0xc1, 0x28, 0x18, 0x05,
		0xa3, 0x60, 0x14, 0x8c, 0x82, 0x51, 0x30, 0x0a, 0x46, 0xc1, 0x28, 0x18,
		0x05, 0xa3, 0x60, 0x14, 0x8c, 0x82, 0x51, 0x30, 0x0a, 0x46, 0xc1, 0x28,
		0x18, 0x05, 0xa3, 0x60, 0x14, 0x8c, 0x82, 0x51, 0x30, 0x0a, 0x00, 0x00,
		0x84, 0x1c, 0x00, 0x01, 0x1f, 0x9c, 0x44, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
