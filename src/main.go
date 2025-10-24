package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/getlantern/systray"
	"github.com/stephanwesten/go-whisper/src/audio"
	"github.com/stephanwesten/go-whisper/src/whisper"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

var (
	recorder      *audio.Recorder
	transcriber   *whisper.Transcriber
	mStatus       *systray.MenuItem
	stopAnimation chan bool
)

func main() {
	mainthread.Init(fn)
}

func fn() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set the menu bar icon and title
	systray.SetTitle("◉")
	systray.SetTooltip("GoWhisper - Press Cmd+Shift+P to record")

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

	// Register global hotkey: Cmd+Shift+P
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeyP)
	if err := hk.Register(); err != nil {
		log.Printf("Failed to register hotkey: %v", err)
	} else {
		log.Println("Hotkey registered: Cmd+Shift+P")
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
		stopRecordingAnimation()
		systray.SetTitle("◉")
		mStatus.SetTitle("Processing...")

		samples, err := recorder.Stop()
		if err != nil {
			log.Printf("Error stopping recording: %v", err)
			mStatus.SetTitle("Error: Failed to stop recording")
			return
		}

		log.Printf("Recorded %d samples (%.2f seconds)", len(samples), float64(len(samples))/float64(audio.SampleRate))

		// Calculate audio volume/amplitude
		var maxAmplitude float32
		var sumSquared float64
		for _, sample := range samples {
			if abs := sample; abs < 0 {
				abs = -abs
			} else if abs > maxAmplitude {
				maxAmplitude = abs
			}
			sumSquared += float64(sample * sample)
		}
		rms := float32(0)
		if len(samples) > 0 {
			rms = float32(sumSquared / float64(len(samples)))
		}
		log.Printf("Audio levels - Max amplitude: %.4f, RMS: %.4f", maxAmplitude, rms)

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

			// Show user-friendly error dialog
			errorMsg := "GoWhisper needs Accessibility permissions to type text.\n\nPlease go to:\nSystem Settings → Privacy & Security → Accessibility\n\nAnd add your Terminal app to the allowed list."
			showErrorDialog("Accessibility Permission Required", errorMsg)
			return
		}

		log.Println("Successfully sent transcribed text")
		mStatus.SetTitle("Ready")

	} else {
		// Start recording
		log.Println("Starting recording...")
		startRecordingAnimation()
		mStatus.SetTitle("🎤 Recording...")

		if err := recorder.Start(); err != nil {
			log.Printf("Error starting recording: %v", err)
			stopRecordingAnimation()
			systray.SetTitle("◉")
			mStatus.SetTitle("Error: Failed to start")
			return
		}

		log.Println("Recording started - press Cmd+Shift+P again to stop")
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

// showErrorDialog displays an error dialog to the user
func showErrorDialog(title, message string) {
	// AppleScript to show a dialog
	script := `
		display dialog "` + message + `" with title "` + title + `" buttons {"OK"} default button "OK" with icon caution
	`

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to show error dialog: %v", err)
	}
}

// getIconReady returns icon for ready state (microphone)
func getIconReady() []byte {
	// Simple microphone icon - 18x18 template image (black/transparent)
	// This is a minimal microphone design that should render well
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x12, 0x00, 0x00, 0x00, 0x12,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x56, 0xce, 0x8e, 0x57, 0x00, 0x00, 0x00,
		0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x0e, 0xc4, 0x00, 0x00, 0x0e,
		0xc4, 0x01, 0x95, 0x2b, 0x0e, 0x1b, 0x00, 0x00, 0x00, 0x61, 0x49, 0x44,
		0x41, 0x54, 0x38, 0x8d, 0xed, 0xd3, 0x31, 0x0a, 0x80, 0x30, 0x0c, 0x04,
		0x50, 0xdf, 0xff, 0xd9, 0xa5, 0x76, 0xf2, 0x02, 0x5d, 0x08, 0x12, 0x5c,
		0x7a, 0x93, 0x96, 0x16, 0x04, 0x11, 0xa5, 0x14, 0x63, 0xcc, 0x6f, 0x02,
		0x00, 0xa0, 0x01, 0x3e, 0x80, 0x17, 0xf0, 0x03, 0x7e, 0xc0, 0x1f, 0xf8,
		0x07, 0xff, 0xe0, 0x0f, 0xfc, 0x83, 0x3f, 0xf0, 0x0f, 0xfe, 0xc0, 0x3f,
		0xf8, 0x03, 0xff, 0xe0, 0x1f, 0xfc, 0x81, 0x7f, 0xf0, 0x0f, 0xfe, 0xc0,
		0x1f, 0xf8, 0x07, 0x7f, 0xe0, 0x0f, 0xfc, 0x83, 0x3f, 0xf0, 0x0f, 0xfe,
		0xc0, 0x1f, 0xf8, 0x07, 0x6f, 0x60, 0x07, 0x76, 0x60, 0x0f, 0x0e, 0xe0,
		0x04, 0x00, 0x00, 0xff, 0xff, 0x78, 0x93, 0x1d, 0xa6, 0x88, 0x59, 0x7c,
		0xb9, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60,
		0x82,
	}
}

// getIconRecording returns icon for recording state (red microphone)
func getIconRecording() []byte {
	// Red microphone icon for menu bar (18x18)
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x12, 0x00, 0x00, 0x00, 0x12,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x56, 0xce, 0x8e, 0x57, 0x00, 0x00, 0x00,
		0xd7, 0x49, 0x44, 0x41, 0x54, 0x38, 0x8d, 0xed, 0x93, 0xb1, 0x0d, 0x02,
		0x31, 0x0c, 0x45, 0xdf, 0x24, 0x24, 0x00, 0x21, 0x14, 0x50, 0x80, 0x02,
		0x14, 0xa0, 0x84, 0x12, 0x4a, 0x08, 0xa1, 0x84, 0x10, 0x0a, 0x50, 0x40,
		0x09, 0x15, 0x94, 0x50, 0x41, 0x01, 0x4a, 0x18, 0x72, 0x24, 0x76, 0x12,
		0x3b, 0x89, 0x13, 0xf7, 0xfe, 0xee, 0xdd, 0xf3, 0x7c, 0xf7, 0x1c, 0x49,
		0x92, 0x24, 0x49, 0x92, 0xfc, 0x0b, 0xf0, 0x06, 0xb8, 0x02, 0xae, 0x80,
		0x4b, 0x00, 0x1f, 0xc0, 0x19, 0xc0, 0x0f, 0xe0, 0x05, 0x70, 0x0d, 0x9c,
		0x03, 0x17, 0xc0, 0x25, 0x70, 0x05, 0x9c, 0x02, 0xc7, 0xc0, 0x29, 0x70,
		0x0c, 0x5c, 0x01, 0x67, 0xc0, 0x05, 0x70, 0x0e, 0x9c, 0x02, 0xa7, 0xc0,
		0x19, 0x70, 0x06, 0x5c, 0x02, 0x17, 0xc0, 0x45, 0x70, 0x09, 0x9c, 0x03,
		0x97, 0xc0, 0x15, 0x70, 0x0d, 0xdc, 0x00, 0xd7, 0xc0, 0x2d, 0x70, 0x07,
		0xdc, 0x03, 0x0f, 0xc0, 0x23, 0xf0, 0x04, 0x3c, 0x03, 0x2f, 0xc0, 0x2b,
		0xf0, 0x06, 0xbc, 0x03, 0x1f, 0xc0, 0x27, 0xf0, 0x05, 0x7c, 0x03, 0x3f,
		0xc0, 0x1f, 0xf0, 0x0f, 0xfc, 0x01, 0xff, 0xc0, 0x3f, 0xf0, 0x00, 0x3c,
		0x02, 0x4f, 0xc0, 0x33, 0xf0, 0x02, 0xbc, 0x02, 0x6f, 0xc0, 0x3b, 0xf0,
		0x01, 0x7c, 0x02, 0x5f, 0xc0, 0x37, 0xf0, 0x03, 0xfc, 0x01, 0xff, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xcd, 0x3d, 0x31, 0x5c, 0x8b, 0xc5, 0x01, 0x84,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

// startRecordingAnimation starts a blinking animation in the menu bar
func startRecordingAnimation() {
	stopAnimation = make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(750 * time.Millisecond) // Blink every 750ms
		defer ticker.Stop()

		blinkState := false
		for {
			select {
			case <-stopAnimation:
				return
			case <-ticker.C:
				if blinkState {
					systray.SetTitle("🔴") // Filled red circle
				} else {
					systray.SetTitle("⭕") // Hollow red circle
				}
				blinkState = !blinkState
			}
		}
	}()
}

// stopRecordingAnimation stops the blinking animation
func stopRecordingAnimation() {
	if stopAnimation != nil {
		select {
		case stopAnimation <- true:
		default:
		}
	}
}
