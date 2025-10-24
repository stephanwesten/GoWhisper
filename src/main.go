package main

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/getlantern/systray"
	"github.com/stephanwesten/go-whisper/src/audio"
	"github.com/stephanwesten/go-whisper/src/whisper"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

const (
	recordingIndicator  = "Recording"
	processingIndicator = "Processing"
)

var (
	recorder      *audio.Recorder
	transcriber   *whisper.Transcriber
	mStatus       *systray.MenuItem
	stopAnimation chan bool
	isProcessing  bool // Prevent re-entrant hotkey handling
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
	mHotkey := systray.AddMenuItem("⌘⇧P - Start/Stop Recording", "Press Cmd+Shift+P to toggle recording")
	mHotkey.Disable()
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
	// Ignore hotkey presses while already processing
	if isProcessing {
		log.Println("Already processing, ignoring hotkey")
		return
	}

	if recorder.IsRecording() {
		// Stop recording and transcribe
		isProcessing = true
		log.Println("Stopping recording...")
		stopRecordingAnimation()
		systray.SetTitle("◉")
		mStatus.SetTitle("Processing...")
		log.Println("⏳ Processing transcription...")

		// Add delay before sending processing indicator to ensure the hotkey (Cmd+Shift+P)
		// is fully released before AppleScript types. Without this delay, the modifier keys
		// may still be pressed when keystroke injection occurs, causing incorrect characters.
		time.Sleep(100 * time.Millisecond)

		// Delete the "Recording" text (9 characters) before showing "Processing"
		if err := sendBackspaces(len(recordingIndicator)); err != nil {
			log.Printf("Error deleting recording indicator: %v", err)
		}

		if err := sendTextToActiveWindow(processingIndicator); err != nil {
			log.Printf("Error sending processing indicator: %v", err)
		}

		samples, err := recorder.Stop()
		if err != nil {
			log.Printf("Error stopping recording: %v", err)
			mStatus.SetTitle("Error: Failed to stop recording")
			isProcessing = false
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
			isProcessing = false
			return
		}

		// Transcribe
		log.Println("Transcribing...")
		mStatus.SetTitle("Transcribing...")

		text, err := transcriber.Transcribe(samples)
		if err != nil {
			log.Printf("Error transcribing: %v", err)
			mStatus.SetTitle("Error: Transcription failed")
			log.Println("✗ Transcription failed")
			isProcessing = false
			return
		}

		log.Printf("✓ Transcription: %s", text)

		if text == "" {
			log.Println("No speech detected")
			mStatus.SetTitle("Ready")
			isProcessing = false
			return
		}

		// Send transcribed text to active window
		mStatus.SetTitle("Typing...")

		// Delete the "Processing" text before typing the transcription
		if err := sendBackspaces(len(processingIndicator)); err != nil {
			log.Printf("Error deleting processing indicator: %v", err)
		}

		if err := sendTextToActiveWindow(text); err != nil {
			log.Printf("Error sending text: %v", err)
			mStatus.SetTitle("Error: Failed to type")

			// Show user-friendly error dialog
			errorMsg := "GoWhisper needs Accessibility permissions to type text.\n\nPlease go to:\nSystem Settings → Privacy & Security → Accessibility\n\nAnd add your Terminal app to the allowed list."
			showErrorDialog("Accessibility Permission Required", errorMsg)
			isProcessing = false
			return
		}

		log.Println("Successfully sent transcribed text")
		mStatus.SetTitle("Ready")
		isProcessing = false

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

		// Add delay before sending indicator text to ensure the hotkey (Cmd+Shift+P)
		// is fully released before AppleScript types. Without this delay, the modifier keys
		// may still be pressed when keystroke injection occurs, causing incorrect characters.
		time.Sleep(100 * time.Millisecond)
		if err := sendTextToActiveWindow(recordingIndicator); err != nil {
			log.Printf("Error sending recording indicator: %v", err)
		}
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

// sendBackspaces sends the specified number of backspace key presses to delete text
func sendBackspaces(count int) error {
	if count <= 0 {
		return nil
	}

	// AppleScript to send backspace keys (key code 51 is delete/backspace)
	script := `
		tell application "System Events"
			repeat ` + fmt.Sprintf("%d", count) + ` times
				key code 51
			end repeat
		end tell
	`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("AppleScript output: %s", string(output))
		return err
	}

	log.Printf("Successfully sent %d backspaces", count)
	return nil
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
