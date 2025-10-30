package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
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

// AppState represents the current state of the application
type AppState int

const (
	StateIdle AppState = iota
	StateRecording
	StateProcessing
)

func (s AppState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRecording:
		return "Recording"
	case StateProcessing:
		return "Processing"
	default:
		// Log before panic to ensure it's captured
		log.Printf("FATAL: Unknown state detected: %d (valid states: Idle=%d, Recording=%d, Processing=%d)",
			s, StateIdle, StateRecording, StateProcessing)
		panic(fmt.Sprintf("Unknown AppState: %d - this should never happen, indicates memory corruption or invalid cast", s))
	}
}

var (
	recorder      *audio.Recorder
	transcriber   *whisper.Transcriber
	mStatus       *systray.MenuItem
	mHotkey       *systray.MenuItem
	mToggleHotkey *systray.MenuItem
	stopAnimation chan bool
	hk            *hotkey.Hotkey

	// State machine with mutex protection
	stateMu      sync.Mutex
	currentState AppState = StateIdle

	// Hotkey enable/disable state
	enabledMu sync.Mutex
	isEnabled bool = true
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
	mHotkey = systray.AddMenuItem("⌘⇧P - Start Recording", "Click to start recording")
	systray.AddSeparator()
	mToggleHotkey = systray.AddMenuItem("Disable Hotkey", "Temporarily disable the global hotkey")
	systray.AddSeparator()
	mStatus = systray.AddMenuItem("", "Current operation status")
	mStatus.Hide() // Hidden by default, shown during operations
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Register global hotkey: Cmd+Shift+P
	hk = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeyP)
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
			case <-mHotkey.ClickedCh:
				log.Println("Start/Stop Recording menu item clicked")
				handleHotkey()
			case <-mToggleHotkey.ClickedCh:
				toggleHotkey()
			case <-mQuit.ClickedCh:
				log.Println("Quit clicked")
				hk.Unregister()
				systray.Quit()
			}
		}
	}()
}

// isHotkeyEnabled returns whether the hotkey is enabled (thread-safe)
func isHotkeyEnabled() bool {
	enabledMu.Lock()
	defer enabledMu.Unlock()
	return isEnabled
}

// setHotkeyEnabled sets the hotkey enabled state (thread-safe)
func setHotkeyEnabled(enabled bool) {
	enabledMu.Lock()
	defer enabledMu.Unlock()
	isEnabled = enabled
}

// getState returns the current application state (thread-safe)
func getState() AppState {
	stateMu.Lock()
	defer stateMu.Unlock()
	return currentState
}

// setState transitions to a new state (thread-safe)
func setState(newState AppState) {
	stateMu.Lock()
	defer stateMu.Unlock()
	oldState := currentState
	currentState = newState
	log.Printf("State transition: %s -> %s", oldState, newState)
}

// tryTransitionState attempts to transition from expectedState to newState
// Returns true if successful, false if current state doesn't match expectedState
func tryTransitionState(expectedState, newState AppState) bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	if currentState != expectedState {
		log.Printf("State transition rejected: expected %s, but current is %s", expectedState, currentState)
		return false
	}
	oldState := currentState
	currentState = newState
	log.Printf("State transition: %s -> %s", oldState, newState)
	return true
}

// toggleHotkey enables or disables the global hotkey
func toggleHotkey() {
	enabled := isHotkeyEnabled()

	if enabled {
		// Disabling hotkey
		log.Println("Disabling hotkey...")

		// If currently recording, stop and discard
		state := getState()
		if state == StateRecording {
			log.Println("Stopping recording due to hotkey disable")
			stopRecordingAnimation()
			systray.SetTitle("○") // Hollow circle for disabled

			// Stop recording and discard samples
			_, err := recorder.Stop()
			if err != nil {
				log.Printf("Error stopping recording: %v", err)
			}

			// Delete the "Recording" indicator text
			if err := sendBackspaces(len(recordingIndicator)); err != nil {
				log.Printf("Error deleting recording indicator: %v", err)
			}

			setState(StateIdle)
			mStatus.Hide()
		} else {
			systray.SetTitle("○") // Hollow circle for disabled
			mStatus.Hide()
		}

		// Set disabled state BEFORE unregistering to prevent race condition
		setHotkeyEnabled(false)
		mToggleHotkey.SetTitle("Enable Hotkey")

		// Unregister hotkey
		if err := hk.Unregister(); err != nil {
			log.Printf("Failed to unregister hotkey: %v", err)
		} else {
			log.Println("Hotkey unregistered successfully")
		}

	} else {
		// Enabling hotkey
		log.Println("Enabling hotkey...")

		// Register hotkey
		if err := hk.Register(); err != nil {
			log.Printf("Failed to register hotkey: %v", err)
			mStatus.SetTitle("Error: Failed to enable hotkey")
			return
		}

		log.Println("Hotkey registered successfully")
		setHotkeyEnabled(true)
		systray.SetTitle("◉") // Remove disabled overlay
		mStatus.Hide()
		mToggleHotkey.SetTitle("Disable Hotkey")
	}
}

func handleHotkey() {
	// CRITICAL: Check if hotkey is enabled first
	if !isHotkeyEnabled() {
		log.Println("Hotkey is disabled, ignoring")
		return
	}

	state := getState()

	// Ignore hotkey presses while processing
	if state == StateProcessing {
		log.Println("Already processing, ignoring hotkey")
		return
	}

	if state == StateRecording {
		// Transition to processing state
		if !tryTransitionState(StateRecording, StateProcessing) {
			log.Println("Failed to transition to Processing state")
			return
		}

		// Stop recording and transcribe
		log.Println("Stopping recording...")
		stopRecordingAnimation()
		systray.SetTitle("◉")
		mStatus.SetTitle("Processing...")
		mStatus.Show()
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
			mHotkey.SetTitle("⌘⇧P - Start Recording")
			mStatus.SetTitle("Error: Failed to stop recording")
			setState(StateIdle)
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
			mHotkey.SetTitle("⌘⇧P - Start Recording")
			mStatus.Hide()
			setState(StateIdle)
			return
		}

		// Transcribe
		log.Println("Transcribing...")
		mStatus.SetTitle("Transcribing...")

		text, err := transcriber.Transcribe(samples)
		if err != nil {
			log.Printf("Error transcribing: %v", err)
			mHotkey.SetTitle("⌘⇧P - Start Recording")
			mStatus.SetTitle("Error: Transcription failed")
			log.Println("✗ Transcription failed")
			setState(StateIdle)
			return
		}

		log.Printf("✓ Transcription: %s", text)

		if text == "" {
			log.Println("No speech detected")
			mHotkey.SetTitle("⌘⇧P - Start Recording")
			mStatus.Hide()
			setState(StateIdle)
			return
		}

		// Detect keywords in transcription
		hasClaude := containsClaude(text)
		hasClipboard := containsClipboardKeyword(text)

		log.Printf("Keyword detection - Claude: %v, Clipboard: %v", hasClaude, hasClipboard)

		// Determine output text and action based on keywords
		var outputText string
		var shouldCopyToClipboard bool
		var shouldRephrase bool

		if hasClaude && hasClipboard {
			// Both keywords: Remove both, rephrase with Claude, copy to clipboard
			outputText = removeCombinedKeywords(text)
			shouldRephrase = true
			shouldCopyToClipboard = true
			log.Printf("Both keywords detected. Will rephrase and copy: %s", outputText)
		} else if hasClaude {
			// Only Claude: Remove keyword, rephrase, type to window
			outputText = removeCombinedKeywords(text)
			shouldRephrase = true
			shouldCopyToClipboard = false
			log.Printf("Claude keyword detected. Will rephrase and type: %s", outputText)
		} else if hasClipboard {
			// Only Clipboard: Remove keyword, copy to clipboard
			outputText = removeClipboardPrefix(text)
			shouldRephrase = false
			shouldCopyToClipboard = true
			log.Printf("Clipboard keyword detected. Will copy: %s", outputText)
		} else {
			// No keywords: Type original text
			outputText = text
			shouldRephrase = false
			shouldCopyToClipboard = false
		}

		// Delete the "Processing" text first
		if err := sendBackspaces(len(processingIndicator)); err != nil {
			log.Printf("Error deleting processing indicator: %v", err)
		}

		// Rephrase with Claude if needed
		if shouldRephrase {
			const claudeIndicator = "Asking Claude"
			mStatus.SetTitle("Asking Claude...")

			// Show "Asking Claude" text in the window
			if err := sendTextToActiveWindow(claudeIndicator); err != nil {
				log.Printf("Error sending Claude indicator: %v", err)
			}

			rephrased, err := rephraseWithClaude(outputText)

			// Delete the "Asking Claude" text
			if err := sendBackspaces(len(claudeIndicator)); err != nil {
				log.Printf("Error deleting Claude indicator: %v", err)
			}

			if err != nil {
				log.Printf("Error rephrasing with Claude: %v", err)
				mHotkey.SetTitle("⌘⇧P - Start Recording")
				mStatus.SetTitle("Error: Claude rephrasing failed")
				mStatus.Show()
				setState(StateIdle)
				return
			}
			outputText = rephrased
			log.Printf("Successfully rephrased: %s", outputText)
		}

		if shouldCopyToClipboard {
			// Copy to clipboard
			mStatus.SetTitle("Copying to clipboard...")
			if err := clipboard.WriteAll(outputText); err != nil {
				log.Printf("Error copying to clipboard: %v", err)
				mHotkey.SetTitle("⌘⇧P - Start Recording")
				mStatus.SetTitle("Error: Failed to copy")
				mStatus.Show()
				setState(StateIdle)
				return
			}
			log.Printf("Successfully copied to clipboard: %s", outputText)
		} else {
			// Send transcribed text to active window
			mStatus.SetTitle("Typing...")
			if err := sendTextToActiveWindow(outputText); err != nil {
				log.Printf("Error sending text: %v", err)
				mHotkey.SetTitle("⌘⇧P - Start Recording")
				mStatus.SetTitle("Error: Failed to type")

				// Show user-friendly error dialog
				errorMsg := "GoWhisper needs Accessibility permissions to type text.\n\nPlease go to:\nSystem Settings → Privacy & Security → Accessibility\n\nAnd add your Terminal app to the allowed list."
				showErrorDialog("Accessibility Permission Required", errorMsg)
				setState(StateIdle)
				return
			}
			log.Println("Successfully sent transcribed text")
		}

		mHotkey.SetTitle("⌘⇧P - Start Recording")
		mStatus.Hide()
		setState(StateIdle)

	} else if state == StateIdle {
		// Transition to recording state
		if !tryTransitionState(StateIdle, StateRecording) {
			log.Println("Failed to transition to Recording state")
			return
		}

		// Start recording
		log.Println("Starting recording...")
		startRecordingAnimation()
		mHotkey.SetTitle("⌘⇧P - Stop Recording")
		mStatus.SetTitle("🎤 Recording...")
		mStatus.Show()

		if err := recorder.Start(); err != nil {
			log.Printf("Error starting recording: %v", err)
			stopRecordingAnimation()
			systray.SetTitle("◉")
			mHotkey.SetTitle("⌘⇧P - Start Recording")
			mStatus.SetTitle("Error: Failed to start")
			mStatus.Show()
			setState(StateIdle)
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
	} else {
		log.Printf("Unexpected state in handleHotkey: %s", state)
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
	// For complex text (multiline, special chars), use clipboard + paste instead of keystroke
	// This avoids AppleScript escaping issues and permission dialogs

	// Save current clipboard content
	originalClipboard, err := clipboard.ReadAll()
	if err != nil {
		log.Printf("Warning: Could not read clipboard: %v", err)
		originalClipboard = ""
	}

	// Put text in clipboard
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to write to clipboard: %v", err)
	}

	// Use AppleScript to paste (Cmd+V)
	script := `
		tell application "System Events"
			keystroke "v" using command down
		end tell
	`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("AppleScript output: %s", string(output))
		// Try to restore clipboard even if paste failed
		clipboard.WriteAll(originalClipboard)
		return err
	}

	// Restore original clipboard content after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		clipboard.WriteAll(originalClipboard)
	}()

	log.Printf("Successfully sent text: %s", text)
	return nil
}

// startsWithClipboard checks if text starts with "clipboard" (case-insensitive)
func startsWithClipboard(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.HasPrefix(lower, "clipboard")
}

// removeClipboardPrefix removes "clipboard" prefix and returns the remaining text
func removeClipboardPrefix(text string) string {
	trimmed := strings.TrimSpace(text)
	// Find where "clipboard" ends (case-insensitive)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "clipboard") {
		// Remove "clipboard" and any following whitespace
		remaining := trimmed[len("clipboard"):]
		return strings.TrimSpace(remaining)
	}
	return trimmed
}

// stripPunctuation removes common punctuation from a word
func stripPunctuation(word string) string {
	return strings.Trim(word, ".,!?;:\"'()[]{}")
}

// containsClaude checks if text starts with "claude" keyword (case-insensitive)
func containsClaude(text string) bool {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return false
	}
	// Check first TWO words for "claude" to allow "clipboard claude" combinations
	// but avoid matching "When Claude is running" in natural speech
	limit := 2
	if len(words) < limit {
		limit = len(words)
	}
	for i := 0; i < limit; i++ {
		cleaned := strings.ToLower(stripPunctuation(words[i]))
		if cleaned == "claude" {
			return true
		}
	}
	return false
}

// containsClipboardKeyword checks if text starts with "clipboard" keyword (case-insensitive)
func containsClipboardKeyword(text string) bool {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return false
	}
	// Check first TWO words for "clipboard" to allow "claude clipboard" combinations
	// but avoid matching "The clipboard contains" in natural speech
	limit := 2
	if len(words) < limit {
		limit = len(words)
	}
	for i := 0; i < limit; i++ {
		cleaned := strings.ToLower(stripPunctuation(words[i]))
		if cleaned == "clipboard" {
			return true
		}
	}
	return false
}

// removeCombinedKeywords removes both "claude" and "clipboard" from text (any order)
func removeCombinedKeywords(text string) string {
	words := strings.Fields(strings.TrimSpace(text))
	var filtered []string

	for _, word := range words {
		cleaned := strings.ToLower(stripPunctuation(word))
		if cleaned != "claude" && cleaned != "clipboard" {
			filtered = append(filtered, word)
		}
	}

	return strings.TrimSpace(strings.Join(filtered, " "))
}

// rephraseWithClaude sends text to Claude for rephrasing
func rephraseWithClaude(text string) (string, error) {
	systemPrompt := "You are a text refinement assistant. When given text, output ONLY the refined version without any explanation, formatting, or commentary. Just return the improved text directly."

	// Use claude CLI with --print flag and system prompt
	cmd := exec.Command("claude", "--print", "--system-prompt", systemPrompt, "-p", text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Claude CLI error: %v, output: %s", err, string(output))
		return "", fmt.Errorf("failed to rephrase with Claude: %v", err)
	}

	rephrased := strings.TrimSpace(string(output))
	if rephrased == "" {
		return "", fmt.Errorf("Claude returned empty response")
	}

	log.Printf("Claude rephrasing:\nOriginal: %s\nRephrased: %s", text, rephrased)
	return rephrased, nil
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
