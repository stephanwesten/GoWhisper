package main

import (
	"sync"
	"testing"
)

// TestStateManagement tests the thread-safe state management functions
func TestStateManagement(t *testing.T) {
	// Save original state
	originalState := currentState
	defer func() { currentState = originalState }()

	t.Run("getState and setState", func(t *testing.T) {
		setState(StateIdle)
		if got := getState(); got != StateIdle {
			t.Errorf("getState() = %v, want %v", got, StateIdle)
		}

		setState(StateRecording)
		if got := getState(); got != StateRecording {
			t.Errorf("getState() = %v, want %v", got, StateRecording)
		}

		setState(StateProcessing)
		if got := getState(); got != StateProcessing {
			t.Errorf("getState() = %v, want %v", got, StateProcessing)
		}
	})

	t.Run("tryTransitionState success", func(t *testing.T) {
		setState(StateIdle)
		if !tryTransitionState(StateIdle, StateRecording) {
			t.Error("tryTransitionState(StateIdle, StateRecording) = false, want true")
		}
		if got := getState(); got != StateRecording {
			t.Errorf("After transition, state = %v, want %v", got, StateRecording)
		}
	})

	t.Run("tryTransitionState failure", func(t *testing.T) {
		setState(StateIdle)
		if tryTransitionState(StateRecording, StateProcessing) {
			t.Error("tryTransitionState with wrong expected state = true, want false")
		}
		if got := getState(); got != StateIdle {
			t.Errorf("After failed transition, state = %v, want %v (unchanged)", got, StateIdle)
		}
	})

	t.Run("concurrent state access", func(t *testing.T) {
		setState(StateIdle)
		var wg sync.WaitGroup
		iterations := 100

		// Multiple goroutines trying to transition state
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tryTransitionState(StateIdle, StateRecording)
			}()
		}

		wg.Wait()

		// Final state should be either Idle or Recording, never corrupted
		finalState := getState()
		if finalState != StateIdle && finalState != StateRecording {
			t.Errorf("After concurrent access, state = %v, want StateIdle or StateRecording", finalState)
		}
	})
}

// TestHotkeyEnabledState tests the enable/disable state management
func TestHotkeyEnabledState(t *testing.T) {
	// Save original state
	originalEnabled := isEnabled
	defer func() { isEnabled = originalEnabled }()

	t.Run("isHotkeyEnabled and setHotkeyEnabled", func(t *testing.T) {
		setHotkeyEnabled(true)
		if !isHotkeyEnabled() {
			t.Error("isHotkeyEnabled() = false, want true")
		}

		setHotkeyEnabled(false)
		if isHotkeyEnabled() {
			t.Error("isHotkeyEnabled() = true, want false")
		}
	})

	t.Run("concurrent enabled state access", func(t *testing.T) {
		setHotkeyEnabled(true)
		var wg sync.WaitGroup
		iterations := 100

		// Multiple goroutines toggling enabled state
		for i := 0; i < iterations; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				setHotkeyEnabled(true)
			}()
			go func() {
				defer wg.Done()
				setHotkeyEnabled(false)
			}()
		}

		wg.Wait()

		// Final state should be boolean, never corrupted
		_ = isHotkeyEnabled() // Should not panic or return invalid value
	})
}

// TestAppStateString tests the String() method of AppState
func TestAppStateString(t *testing.T) {
	tests := []struct {
		state AppState
		want  string
	}{
		{StateIdle, "Idle"},
		{StateRecording, "Recording"},
		{StateProcessing, "Processing"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("AppState(%d).String() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

// TestHandleHotkeyWithDisabledState tests the critical fix for Bug #1
func TestHandleHotkeyWithDisabledState(t *testing.T) {
	// Save original states
	originalState := currentState
	originalEnabled := isEnabled
	defer func() {
		currentState = originalState
		isEnabled = originalEnabled
	}()

	t.Run("hotkey ignored when disabled", func(t *testing.T) {
		setState(StateIdle)
		setHotkeyEnabled(false)

		// Simulate hotkey press by calling handleHotkey directly
		// It should return immediately without changing state
		handleHotkey()

		// State should remain Idle
		if got := getState(); got != StateIdle {
			t.Errorf("After handleHotkey() with disabled=false, state = %v, want StateIdle", got)
		}
	})

	t.Run("hotkey processed when enabled and idle", func(t *testing.T) {
		setState(StateIdle)
		setHotkeyEnabled(true)

		// Note: We can't fully test handleHotkey() as it requires recorder/transcriber
		// but we can verify the enabled check works
		if !isHotkeyEnabled() {
			t.Error("Expected hotkey to be enabled")
		}
	})

	t.Run("hotkey ignored during processing", func(t *testing.T) {
		setState(StateProcessing)
		setHotkeyEnabled(true)

		initialState := getState()
		handleHotkey()

		// State should remain unchanged
		if got := getState(); got != initialState {
			t.Errorf("After handleHotkey() during processing, state changed from %v to %v", initialState, got)
		}
	})
}

// TestToggleHotkeyStateChanges tests the enable/disable toggle logic
func TestToggleHotkeyStateChanges(t *testing.T) {
	// Save original states
	originalEnabled := isEnabled
	originalState := currentState
	defer func() {
		isEnabled = originalEnabled
		currentState = originalState
	}()

	t.Run("disable sets state before other operations", func(t *testing.T) {
		setHotkeyEnabled(true)
		setState(StateIdle)

		// We can't fully test toggleHotkey() as it requires hotkey, systray, etc.
		// But we can verify the state setting order by checking our fix is in place

		// Verify initial state
		if !isHotkeyEnabled() {
			t.Error("Expected hotkey to start enabled")
		}

		// Manually test the critical section
		setHotkeyEnabled(false)

		// At this point, even if unregister fails, isEnabled should be false
		if isHotkeyEnabled() {
			t.Error("Expected hotkey to be disabled after setHotkeyEnabled(false)")
		}
	})

	t.Run("enable to disable state transition", func(t *testing.T) {
		setHotkeyEnabled(true)
		if !isHotkeyEnabled() {
			t.Error("Failed to set enabled to true")
		}

		setHotkeyEnabled(false)
		if isHotkeyEnabled() {
			t.Error("Failed to set enabled to false")
		}
	})

	t.Run("disable to enable state transition", func(t *testing.T) {
		setHotkeyEnabled(false)
		if isHotkeyEnabled() {
			t.Error("Failed to set enabled to false")
		}

		setHotkeyEnabled(true)
		if !isHotkeyEnabled() {
			t.Error("Failed to set enabled to true")
		}
	})
}

// TestRaceConditionProtection tests that the bug fix prevents race conditions
func TestRaceConditionProtection(t *testing.T) {
	// Save original states
	originalEnabled := isEnabled
	originalState := currentState
	defer func() {
		isEnabled = originalEnabled
		currentState = originalState
	}()

	t.Run("concurrent enable/disable and state checks", func(t *testing.T) {
		setHotkeyEnabled(true)
		setState(StateIdle)

		var wg sync.WaitGroup
		iterations := 100

		// Goroutines toggling enabled state
		for i := 0; i < iterations; i++ {
			wg.Add(3)
			go func() {
				defer wg.Done()
				setHotkeyEnabled(false)
			}()
			go func() {
				defer wg.Done()
				setHotkeyEnabled(true)
			}()
			go func() {
				defer wg.Done()
				// Check that reading state while toggling doesn't cause issues
				_ = isHotkeyEnabled()
				_ = getState()
			}()
		}

		wg.Wait()

		// Should complete without panicking
		_ = isHotkeyEnabled() // Should return valid boolean
		_ = getState()        // Should return valid state
	})

	t.Run("handleHotkey always checks enabled state first", func(t *testing.T) {
		// This test verifies the order of operations in handleHotkey
		setState(StateIdle)
		setHotkeyEnabled(false)

		// Even with valid state, disabled hotkey should be ignored
		initialState := getState()
		handleHotkey()

		// State should not have changed
		if getState() != initialState {
			t.Error("handleHotkey() changed state despite being disabled")
		}
	})
}

// TestClipboardDetection tests the clipboard prefix detection and removal
func TestClipboardDetection(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldDetect   bool
		expectedOutput string
	}{
		{
			name:           "starts with clipboard lowercase",
			input:          "clipboard this is a test",
			shouldDetect:   true,
			expectedOutput: "this is a test",
		},
		{
			name:           "starts with Clipboard capitalized",
			input:          "Clipboard another test",
			shouldDetect:   true,
			expectedOutput: "another test",
		},
		{
			name:           "starts with CLIPBOARD uppercase",
			input:          "CLIPBOARD all caps test",
			shouldDetect:   true,
			expectedOutput: "all caps test",
		},
		{
			name:           "starts with mixed case ClipBoard",
			input:          "ClipBoard mixed case",
			shouldDetect:   true,
			expectedOutput: "mixed case",
		},
		{
			name:           "clipboard with leading whitespace",
			input:          "  clipboard trimmed test",
			shouldDetect:   true,
			expectedOutput: "trimmed test",
		},
		{
			name:           "clipboard with extra spaces after",
			input:          "clipboard    multiple spaces",
			shouldDetect:   true,
			expectedOutput: "multiple spaces",
		},
		{
			name:           "clipboard alone",
			input:          "clipboard",
			shouldDetect:   true,
			expectedOutput: "",
		},
		{
			name:           "does not start with clipboard",
			input:          "this is not clipboard",
			shouldDetect:   false,
			expectedOutput: "this is not clipboard",
		},
		{
			name:           "clipboard in middle",
			input:          "copy to clipboard please",
			shouldDetect:   false,
			expectedOutput: "copy to clipboard please",
		},
		{
			name:           "empty string",
			input:          "",
			shouldDetect:   false,
			expectedOutput: "",
		},
		{
			name:           "just whitespace",
			input:          "   ",
			shouldDetect:   false,
			expectedOutput: "",
		},
		{
			name:           "clipboard with punctuation",
			input:          "clipboard, this has a comma",
			shouldDetect:   true,
			expectedOutput: ", this has a comma",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test detection
			detected := startsWithClipboard(tt.input)
			if detected != tt.shouldDetect {
				t.Errorf("startsWithClipboard(%q) = %v, want %v", tt.input, detected, tt.shouldDetect)
			}

			// Test removal
			output := removeClipboardPrefix(tt.input)
			if output != tt.expectedOutput {
				t.Errorf("removeClipboardPrefix(%q) = %q, want %q", tt.input, output, tt.expectedOutput)
			}
		})
	}
}

// TestClipboardDetectionEdgeCases tests edge cases for clipboard detection
func TestClipboardDetectionEdgeCases(t *testing.T) {
	t.Run("clipboard variations", func(t *testing.T) {
		variations := []string{
			"clipboard",
			"Clipboard",
			"CLIPBOARD",
			"ClipBoard",
			"cLiPbOaRd",
		}

		for _, v := range variations {
			if !startsWithClipboard(v) {
				t.Errorf("startsWithClipboard(%q) = false, want true", v)
			}
		}
	})

	t.Run("not clipboard variations", func(t *testing.T) {
		notClipboard := []string{
			"clipboar",      // missing 'd'
			"xclipboard",    // has prefix
			"clipboard_test", // technically starts with clipboard, should work
			"clip board",    // has space
			"clipboard-test", // has hyphen, should work
		}

		results := []bool{false, false, true, false, true}

		for i, v := range notClipboard {
			got := startsWithClipboard(v)
			want := results[i]
			if got != want {
				t.Errorf("startsWithClipboard(%q) = %v, want %v", v, got, want)
			}
		}
	})
}

// TestStateTransitionLogic tests the state machine logic
func TestStateTransitionLogic(t *testing.T) {
	// Save original state
	originalState := currentState
	defer func() { currentState = originalState }()

	tests := []struct {
		name          string
		initialState  AppState
		expectedState AppState
		newState      AppState
		wantSuccess   bool
		wantFinalState AppState
	}{
		{
			name:          "Idle to Recording - valid",
			initialState:  StateIdle,
			expectedState: StateIdle,
			newState:      StateRecording,
			wantSuccess:   true,
			wantFinalState: StateRecording,
		},
		{
			name:          "Recording to Processing - valid",
			initialState:  StateRecording,
			expectedState: StateRecording,
			newState:      StateProcessing,
			wantSuccess:   true,
			wantFinalState: StateProcessing,
		},
		{
			name:          "Processing to Idle - valid",
			initialState:  StateProcessing,
			expectedState: StateProcessing,
			newState:      StateIdle,
			wantSuccess:   true,
			wantFinalState: StateIdle,
		},
		{
			name:          "Idle to Processing - invalid (skip Recording)",
			initialState:  StateIdle,
			expectedState: StateIdle,
			newState:      StateProcessing,
			wantSuccess:   true, // tryTransitionState allows any transition if expected matches
			wantFinalState: StateProcessing,
		},
		{
			name:          "Wrong expected state",
			initialState:  StateIdle,
			expectedState: StateRecording,
			newState:      StateProcessing,
			wantSuccess:   false,
			wantFinalState: StateIdle, // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setState(tt.initialState)
			got := tryTransitionState(tt.expectedState, tt.newState)
			if got != tt.wantSuccess {
				t.Errorf("tryTransitionState() = %v, want %v", got, tt.wantSuccess)
			}
			finalState := getState()
			if finalState != tt.wantFinalState {
				t.Errorf("Final state = %v, want %v", finalState, tt.wantFinalState)
			}
		})
	}
}

// TestClaudeDetection tests the Claude keyword detection
func TestClaudeDetection(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		shouldDetect bool
	}{
		{
			name:         "starts with claude lowercase",
			input:        "claude this is a test",
			shouldDetect: true,
		},
		{
			name:         "starts with Claude capitalized",
			input:        "Claude another test",
			shouldDetect: true,
		},
		{
			name:         "starts with CLAUDE uppercase",
			input:        "CLAUDE all caps test",
			shouldDetect: true,
		},
		{
			name:         "claude as second word",
			input:        "hello claude world",
			shouldDetect: true,
		},
		{
			name:         "claude as third word",
			input:        "one two claude three",
			shouldDetect: false,
		},
		{
			name:         "claude as fourth word",
			input:        "one two three claude four",
			shouldDetect: false,
		},
		{
			name:         "does not contain claude",
			input:        "this is not a match",
			shouldDetect: false,
		},
		{
			name:         "claude in middle",
			input:        "the quick brown claude fox",
			shouldDetect: false,
		},
		{
			name:         "empty string",
			input:        "",
			shouldDetect: false,
		},
		{
			name:         "just whitespace",
			input:        "   ",
			shouldDetect: false,
		},
		{
			name:         "claude with comma",
			input:        "Claude, search for bags",
			shouldDetect: true,
		},
		{
			name:         "claude with period",
			input:        "Claude. This is a test",
			shouldDetect: true,
		},
		{
			name:         "claude with exclamation",
			input:        "Claude! Do this now",
			shouldDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := containsClaude(tt.input)
			if detected != tt.shouldDetect {
				t.Errorf("containsClaude(%q) = %v, want %v", tt.input, detected, tt.shouldDetect)
			}
		})
	}
}

// TestClipboardKeywordDetection tests clipboard detection in first 3 words
func TestClipboardKeywordDetection(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		shouldDetect bool
	}{
		{
			name:         "starts with clipboard",
			input:        "clipboard this is a test",
			shouldDetect: true,
		},
		{
			name:         "clipboard as second word",
			input:        "hello clipboard world",
			shouldDetect: true,
		},
		{
			name:         "clipboard as third word",
			input:        "one two clipboard three",
			shouldDetect: false,
		},
		{
			name:         "clipboard as fourth word",
			input:        "one two three clipboard four",
			shouldDetect: false,
		},
		{
			name:         "does not contain clipboard",
			input:        "this is a test",
			shouldDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := containsClipboardKeyword(tt.input)
			if detected != tt.shouldDetect {
				t.Errorf("containsClipboardKeyword(%q) = %v, want %v", tt.input, detected, tt.shouldDetect)
			}
		})
	}
}

// TestRemoveCombinedKeywords tests removing both claude and clipboard
func TestRemoveCombinedKeywords(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "claude clipboard text",
			input:          "claude clipboard this is a test",
			expectedOutput: "this is a test",
		},
		{
			name:           "clipboard claude text",
			input:          "clipboard claude another test",
			expectedOutput: "another test",
		},
		{
			name:           "CLAUDE CLIPBOARD mixed case",
			input:          "CLAUDE CLIPBOARD caps test",
			expectedOutput: "caps test",
		},
		{
			name:           "Clipboard Claude capitalized",
			input:          "Clipboard Claude proper case",
			expectedOutput: "proper case",
		},
		{
			name:           "only claude",
			input:          "claude just this keyword",
			expectedOutput: "just this keyword",
		},
		{
			name:           "only clipboard",
			input:          "clipboard just this keyword",
			expectedOutput: "just this keyword",
		},
		{
			name:           "neither keyword",
			input:          "this has neither",
			expectedOutput: "this has neither",
		},
		{
			name:           "claude in middle",
			input:          "text claude in middle",
			expectedOutput: "text in middle",
		},
		{
			name:           "clipboard in middle",
			input:          "text clipboard in middle",
			expectedOutput: "text in middle",
		},
		{
			name:           "multiple spaces",
			input:          "claude   clipboard   extra   spaces",
			expectedOutput: "extra spaces",
		},
		{
			name:           "empty after removal",
			input:          "claude clipboard",
			expectedOutput: "",
		},
		{
			name:           "with punctuation attached to word",
			input:          "claude clipboard, this has punctuation",
			expectedOutput: "this has punctuation",
		},
		{
			name:           "with separate punctuation",
			input:          "claude clipboard , this has punctuation",
			expectedOutput: ", this has punctuation",
		},
		{
			name:           "mixed order in text",
			input:          "clipboard text claude more text",
			expectedOutput: "text more text",
		},
		{
			name:           "claude with comma",
			input:          "Claude, search for bags",
			expectedOutput: "search for bags",
		},
		{
			name:           "clipboard with comma",
			input:          "Clipboard, copy this text",
			expectedOutput: "copy this text",
		},
		{
			name:           "both with punctuation",
			input:          "Claude, clipboard! do this now",
			expectedOutput: "do this now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := removeCombinedKeywords(tt.input)
			if output != tt.expectedOutput {
				t.Errorf("removeCombinedKeywords(%q) = %q, want %q", tt.input, output, tt.expectedOutput)
			}
		})
	}
}

// TestKeywordCombinations tests all 4 scenarios
func TestKeywordCombinations(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectClaude      bool
		expectClipboard   bool
		expectedProcessed string
	}{
		{
			name:              "neither keyword",
			input:             "just some text",
			expectClaude:      false,
			expectClipboard:   false,
			expectedProcessed: "just some text",
		},
		{
			name:              "only clipboard",
			input:             "clipboard copy this",
			expectClaude:      false,
			expectClipboard:   true,
			expectedProcessed: "copy this",
		},
		{
			name:              "only claude",
			input:             "claude rephrase this",
			expectClaude:      true,
			expectClipboard:   false,
			expectedProcessed: "rephrase this",
		},
		{
			name:              "both keywords - claude first",
			input:             "claude clipboard do both",
			expectClaude:      true,
			expectClipboard:   true,
			expectedProcessed: "do both",
		},
		{
			name:              "both keywords - clipboard first",
			input:             "clipboard claude reverse order",
			expectClaude:      true,
			expectClipboard:   true,
			expectedProcessed: "reverse order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test detection
			hasClaude := containsClaude(tt.input)
			hasClipboard := containsClipboardKeyword(tt.input)

			if hasClaude != tt.expectClaude {
				t.Errorf("containsClaude(%q) = %v, want %v", tt.input, hasClaude, tt.expectClaude)
			}
			if hasClipboard != tt.expectClipboard {
				t.Errorf("containsClipboardKeyword(%q) = %v, want %v", tt.input, hasClipboard, tt.expectClipboard)
			}

			// Test keyword removal
			var processed string
			if hasClaude || hasClipboard {
				processed = removeCombinedKeywords(tt.input)
			} else {
				processed = tt.input
			}

			if processed != tt.expectedProcessed {
				t.Errorf("After processing %q, got %q, want %q", tt.input, processed, tt.expectedProcessed)
			}
		})
	}
}
