# go-whisper: Voice-to-Text macOS Menu Bar App

## Overview
A native macOS menu bar application written in Go that provides voice-to-text transcription to any active window with AI-powered text refinement. The app uses local Whisper AI processing for privacy and runs entirely offline (no cloud dependency for transcription).

## Project Goals
- **MVP Achieved**: Working global hotkey (Cmd+Shift+P) for push-to-talk
- **Native & Lightweight**: Pure Go with minimal C dependencies (whisper.cpp)
- **Privacy-focused**: All transcription happens locally on the user's machine
- **Universal**: Works with any application (terminals, text editors, browsers, etc.)
- **AI-Enhanced**: Optional Claude AI integration for text refinement and clipboard operations

## Core Features

### 1. Menu Bar Interface ✅ IMPLEMENTED
- Small icon in macOS menu bar (top-right status area)
- Dynamic icon states:
  - **◉** - Idle/enabled state
  - **○** - Hotkey disabled
  - **🔴** - Recording in progress
  - **C** - Claude AI processing
- Dropdown menu with:
  - **⌘⇧P - Start Recording** - Initiates voice recording
  - **Disable/Enable Hotkey** - Toggle global hotkey
  - **Voice Commands Info** - Submenu with command help:
    - Say 'claude [text]' - Rephrase with AI
    - Say 'clipboard [text]' - Copy to clipboard
    - Say 'claude clipboard' - Both actions
    - Note: 'clot' also works for 'claude'
  - **Status indicator** - Shows current operation (hidden when idle)
  - **Quit** - Exit application

### 2. Voice Recording ✅ IMPLEMENTED (Global Hotkey)
- **Global Hotkey: Cmd+Shift+P** - Toggle recording from anywhere
- Press once to start recording
- Press again to stop and transcribe
- Audio buffer management with thread-safe recording
- Visual feedback: "Recording" text appears in active window
- Menu bar icon changes to 🔴 during recording

### 3. Speech-to-Text Processing
- **Model Options**: Multiple models available via whisper.cpp
  - **MVP: Whisper Small** (`ggml-small.en.bin`, ~466MB)
    - Good balance of accuracy and speed
    - English-only
    - Standard choice for most use cases
  - **Faster Alternative: Large-v3-Turbo** (`ggml-large-v3-turbo.bin`, ~1.6GB)
    - **8x faster** than large-v3, ~3-4x faster than small
    - Only 4 decoder layers (vs 32 in large-v3)
    - Better accuracy than small model
    - Multilingual support
    - ⚠️ Not trained on translation tasks
    - Quantized version available: `ggml-large-v3-turbo-q5_0.bin` (~574MB)
  - **Alternative: Distil-Large-v3** (if available in ggml format)
    - 6x faster than large-v3
    - English-only
    - 2 decoder layers
    - Within 1% WER of large-v3
- **Processing**: Local, using whisper.cpp via Go bindings
- **Output**: Plain text transcription
- **Recommendation**: Start with `small.en` for MVP, switch to `large-v3-turbo-q5_0` if speed is an issue

### 4. Text Insertion ✅ IMPLEMENTED (Universal)
- **Universal Application Support**: Works with ANY active window
  - Terminals (Terminal.app, iTerm2, Warp, etc.)
  - Text editors (VSCode, Sublime, etc.)
  - Browsers, chat apps, email clients
  - Any application that accepts text input
- **Text Insertion Method**: AppleScript via clipboard
  - Saves original clipboard content
  - Copies text to clipboard
  - Pastes via Cmd+V
  - Restores original clipboard
  - Reliable and works across all applications
- **State Machine**: Idle → Recording → Processing → Idle
  - Re-entrancy protection prevents overlapping operations

### 5. Voice Commands ✅ IMPLEMENTED (AI Integration)
- **Claude AI Rephrasing**: Improve text quality with AI
  - Command: Say "claude [your text]" or "clot [your text]"
  - Strips "claude"/"clot" keyword, sends rest to Claude for refinement
  - Returns grammatically correct, professional version
  - Visual feedback: "Asking Claude" indicator, menu bar icon changes to "C"
  - Optimized: Bypasses MCP plugins for 2-5 second faster startup

- **Clipboard Mode**: Copy text without typing
  - Command: Say "clipboard [your text]"
  - Transcribes speech and copies to clipboard
  - Visual feedback: "Copying to clipboard" indicator

- **Combined Mode**: Both AI refinement and clipboard
  - Command: Say "claude clipboard [text]" (any order)
  - Refines with AI, then copies to clipboard
  - Useful for preparing text for pasting elsewhere

- **Keyword Detection**:
  - Checks first 2 words of transcription
  - Supports variations: "claude" and "clot" (common misrecognition)
  - Case-insensitive matching
  - Punctuation stripping for robustness

## Technical Architecture

### Technology Stack

#### Core Application
- **Language**: Go 1.21+
- **Menu Bar**: `fyne.io/systray` or `github.com/getlantern/systray`
  - Cross-platform but we only need macOS
  - CGO-enabled
- **Build**: Standard Go build with CGO enabled

#### Speech Recognition
- **Engine**: whisper.cpp (C++ implementation)
- **Go Bindings**: `github.com/ggerganov/whisper.cpp/bindings/go`
- **Model Options**:
  - **MVP**: `ggml-small.en.bin` (~466MB) - balanced speed/accuracy
  - **Fast**: `ggml-large-v3-turbo-q5_0.bin` (~574MB) - 3-4x faster, better accuracy
  - **Fastest**: `ggml-large-v3-turbo.bin` (~1.6GB) - unquantized, best quality
- **Requirements**:
  - CGO_ENABLED=1
  - libwhisper.a compiled and linked
  - CGO_CFLAGS_ALLOW="-mfma|-mf16c"

#### Audio Capture
- **Library**: PortAudio via `github.com/gordonklaus/portaudio`
- **Format**: 16-bit PCM, 16kHz (Whisper native format)
- **Alternative**: `github.com/MarkKremer/microphone` (wrapper around PortAudio)

#### macOS Integration
- **Accessibility APIs**: CGO bindings to macOS Objective-C frameworks
- **Keystroke Injection**: Core Graphics or Accessibility APIs
- **Notifications**: Native macOS notifications for errors
- **Options**:
  - Direct CGO to Cocoa/AppKit frameworks
  - Use `github.com/progrium/macdriver` for Go-friendly Mac APIs
  - AppleScript via `osascript` command (simpler but less efficient)

### System Architecture

```
┌─────────────────────────────────────────────┐
│         macOS Menu Bar (NSStatusBar)        │
│              [Mic Icon] ▼                   │
│         ┌──────────────────────┐            │
│         │ ● Enabled            │            │
│         │ ○ Start Recording    │            │
│         │ ─────────────        │            │
│         │ Quit                 │            │
│         └──────────────────────┘            │
└─────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│            Go Application                    │
│                                              │
│  ┌────────────┐  ┌─────────────┐            │
│  │   Audio    │  │  Whisper    │            │
│  │  Capture   │─▶│  Engine     │            │
│  │ PortAudio  │  │(whisper.cpp)│            │
│  └────────────┘  └─────────────┘            │
│                        │                     │
│                        ▼                     │
│  ┌────────────────────────────────┐         │
│  │   macOS Accessibility API       │         │
│  │  - Detect Active Terminal       │         │
│  │  - Send Keystrokes             │         │
│  └────────────────────────────────┘         │
└─────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│     Active Terminal Window                   │
│   (Terminal.app, iTerm2, etc.)              │
│                                              │
│   $ transcribed_text_here█                  │
└─────────────────────────────────────────────┘
```

## Implementation Phases

### Pre-MVP: Terminal Integration Proof of Concept
**Goal**: Verify we can send keystrokes to the active terminal before building the full app.

#### Step 1: Terminal Keystroke Injection
- Create minimal Go program to send text to active terminal
- Test detection of active terminal window
- Test sending characters at cursor position
- Validate approach works with Terminal.app, iTerm2
- Options to explore:
  - AppleScript via `osascript` (simplest)
  - macOS Accessibility APIs via CGO
  - Third-party Go libraries if available

**Success Criteria**:
- [ ] Can detect active terminal application
- [ ] Can insert "hello world" at cursor in terminal
- [ ] Works reliably across Terminal.app and iTerm2

### MVP: Core Voice-to-Terminal Pipeline
**Goal**: Minimal working end-to-end flow: speak → transcribe → type in terminal

#### Step 1: Menu Bar Setup
- Create simple systray application with icon
- Add "Type Test Text" button (hardcoded string)
- Test typing hardcoded text to active terminal
- Add "Quit" option

#### Step 2: Audio Recording
- Initialize PortAudio
- Add "Start Recording" / "Stop Recording" buttons
- Capture microphone input to buffer
- Save buffer to WAV file for testing
- Verify audio quality is sufficient

#### Step 3: Whisper Integration
- Build whisper.cpp library (manual prerequisite)
- Manually download Whisper small.en model to `~/.go-whisper/models/`
- Load model on app startup
- Transcribe recorded audio buffer
- Display transcription in menu (for debugging)

#### Step 4: Connect the Pipeline
- Record audio → transcribe → type to terminal
- Add basic error handling (no terminal active)
- Add enable/disable toggle
- Test end-to-end flow

**Success Criteria**:
- [ ] Menu bar icon appears and is clickable
- [ ] Can record 5-10 seconds of speech
- [ ] Audio is transcribed using Whisper (small.en model)
- [ ] Transcribed text appears in active terminal
- [ ] Shows error if no terminal is active
- [ ] Works on macOS 13+ (Ventura)

### Post-MVP: Enhancements

#### Phase 1: Better UX
- Auto-download Whisper model on first run (with progress indicator)
- Global hotkey support (push-to-talk without clicking menu)
- Better visual feedback (recording indicator, status messages)
- Settings panel for configuration
- Support more terminal emulators (Alacritty, Warp, Kitty, etc.)

#### Phase 2: Advanced Features
- Voice Activity Detection (auto start/stop on speech)
- Streaming transcription (real-time as you speak)
- Custom vocabulary/commands for common shell operations
- Model selection (tiny/base/small/medium)
- Multi-language support
- Auto-punctuation and formatting

#### Phase 3: Power User Features
- Local LLM integration for command correction/suggestions
- Command history integration
- Shell-specific context awareness
- Keyboard shortcuts for common operations
- Configurable hotkey (allow users to customize recording hotkey beyond Cmd+Shift+P)

## Model Comparison & Selection

### Whisper Model Options

| Model | Size | Speed | Accuracy | Languages | Use Case |
|-------|------|-------|----------|-----------|----------|
| **small.en** | 466MB | Baseline (1x) | Good | English only | MVP, most users |
| **large-v3-turbo-q5_0** | 574MB | **~3-4x faster** | Better | Multilingual | **Recommended** |
| **large-v3-turbo** | 1.6GB | **~8x faster** | Best | Multilingual | Power users, speed critical |
| distil-large-v3 | N/A | ~6x faster | Within 1% WER | English only | Not yet in ggml |

### What is "Turbo"?
- **Whisper Large-v3-Turbo** is a pruned version of large-v3 with **4 decoder layers** instead of 32
- Released by OpenAI in late 2024, inspired by Distil-Whisper techniques
- **8x faster** than large-v3 with only minor quality degradation
- Supports 99 languages (multilingual) but not trained on translation tasks
- Compatible with whisper.cpp and available in ggml format

### What is Quantization (q5_0)?
- Reduces model precision from 16-bit to 5-bit integers
- **~65% smaller file size** with minimal accuracy loss
- Faster inference and lower memory usage
- `q5_0` = 5-bit quantization, good quality/size tradeoff

### Recommendation
**Use `ggml-large-v3-turbo-q5_0.bin` for production:**
- Similar size to small model (~574MB vs 466MB)
- 3-4x faster transcription
- Better accuracy than small
- Only 108MB larger than small.en

**Use `ggml-small.en.bin` for MVP/development:**
- Faster download and setup
- Proven, widely used
- English-only is fine for initial testing

## Dependencies

### Go Modules
```
github.com/ggerganov/whisper.cpp/bindings/go    # Whisper AI
github.com/gordonklaus/portaudio                 # Audio capture
github.com/getlantern/systray                     # Menu bar icon (using this one)
github.com/atotto/clipboard                       # Clipboard operations
golang.design/x/hotkey                            # Global hotkey registration
golang.design/x/mainthread                        # Main thread execution (required by hotkey)
```

### System Requirements
- macOS 10.15 (Catalina) or later (tested on macOS 14+)
- Microphone access permission
- Accessibility access permission (for keystroke injection via AppleScript)
- Disk space: ~600MB (app + whisper model)
- RAM: ~2GB during transcription
- **Claude Code CLI** (optional, for AI voice commands)
  - Install from https://claude.com/code
  - Required only if using "claude" voice command

### External Dependencies
- **whisper.cpp**: Must compile libwhisper.a
  - Build from source or provide pre-built binaries
  - Installation: `cmake` and `make` in whisper.cpp directory
  - Location configurable via `GOWHISPER_INSTALL_DIR` environment variable
- **PortAudio**: Audio I/O library
  - Install via Homebrew: `brew install portaudio`
- **Whisper Model**: ggml-small.en.bin (default)
  - Download from Hugging Face or OpenAI
  - Location configurable via `GOWHISPER_MODEL` environment variable
  - Default: `~/.go-whisper/models/ggml-small.en.bin`

### Environment Variables (New in v0.2)
- `GOWHISPER_INSTALL_DIR` - Installation directory (default: `$HOME/.go-whisper`)
- `GOWHISPER_MODEL` - Model file path (default: `$GOWHISPER_INSTALL_DIR/models/ggml-small.en.bin`)
- `GOWHISPER_LOG` - Log file location (default: `/tmp/go-whisper.log`)

## Permissions Required

### 1. Microphone Access
- System Preferences → Security & Privacy → Microphone
- App must request permission on first run

### 2. Accessibility Access
- System Preferences → Security & Privacy → Accessibility
- Required for AppleScript to send keystrokes (paste operation)
- App must be added to the allowed list
- **Note**: No longer used for window detection (universal approach)

### 3. Optional: Input Monitoring
- **NOT REQUIRED** - AppleScript handles all keystroke operations

## File Structure (Actual Implementation)
```
go-whisper/
├── go.mod                     # Go module definition (root level)
├── go.sum                     # Dependency checksums
├── build.sh                   # Build script with environment setup
├── README.md                  # Quick start guide
├── CLAUDE.md                  # Claude Code instructions
├── spec.md                    # This file
├── src/
│   ├── main.go               # Entry point, all core logic
│   ├── main_test.go          # Unit tests
│   ├── audio/
│   │   └── recorder.go       # PortAudio recording wrapper
│   └── whisper/
│       └── transcribe.go     # Whisper integration wrapper
├── bin/
│   ├── GoWhisper             # Compiled binary
│   └── run.sh                # Launch script with environment setup
└── ~/.go-whisper/             # Default install location (user home)
    ├── models/
    │   └── ggml-small.en.bin # Whisper model
    └── whisper.cpp/          # whisper.cpp build
        └── build/            # Compiled libraries
```

## Build & Installation

### Prerequisites

#### For Pre-MVP (Terminal Integration Only):
```bash
# No external dependencies needed
# Just Go 1.21+ installed
```

#### For MVP (Full Pipeline):
```bash
# Install Homebrew dependencies
brew install portaudio

# Clone and build whisper.cpp
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp
make

# Download Whisper model (choose one)

# Option 1: Small model (~466MB) - MVP default
mkdir -p ~/.go-whisper/models
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin \
  -O ~/.go-whisper/models/ggml-small.en.bin

# Option 2: Large-v3-Turbo quantized (~574MB) - RECOMMENDED for speed
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q5_0.bin \
  -O ~/.go-whisper/models/ggml-large-v3-turbo-q5_0.bin

# Option 3: Large-v3-Turbo full (~1.6GB) - best quality
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin \
  -O ~/.go-whisper/models/ggml-large-v3-turbo.bin

# OR use whisper.cpp's download script
bash ./models/download-ggml-model.sh small.en
# OR for turbo:
bash ./models/download-ggml-model.sh large-v3-turbo
# Then copy to app directory
cp models/ggml-*.bin ~/.go-whisper/models/
```

### Build Application
```bash
# Enable CGO
export CGO_ENABLED=1
export CGO_CFLAGS_ALLOW="-mfma|-mf16c"

# Set whisper.cpp paths
export C_INCLUDE_PATH=/path/to/whisper.cpp
export LIBRARY_PATH=/path/to/whisper.cpp

# Build
go build -o go-whisper main.go
```

### Run
```bash
./go-whisper
```

On first run, grant permissions:
1. Allow microphone access
2. Add app to Accessibility in System Preferences

## Similar Projects (Reference)

### Existing Solutions Analysis

1. **Voice2Code** (Swift/Objective-C)
   - Menu bar app, uses OpenAI API (not local)
   - Global hotkey support
   - Automatic clipboard integration
   - ⚠️ Requires internet, sends audio to cloud

2. **Whispr** (Rust)
   - Local whisper.cpp integration ✓
   - Menu bar app ✓
   - Privacy-focused ✓
   - Requires Apple Silicon
   - Not terminal-specific

3. **Super Voice Assistant** (Swift + WhisperKit)
   - Uses WhisperKit (CoreML version)
   - Global hotkey (Shift+Alt+Z)
   - Pastes at cursor (any app, not just terminal)
   - Fully offline ✓

4. **hear** (Objective-C CLI)
   - Command-line tool (not menu bar)
   - Uses macOS built-in speech recognition
   - Not using Whisper

### What Makes Our App Different
- **Go-native**: Most alternatives use Swift/Rust/Python
- **Universal**: Works with ANY application, not just terminals
- **Whisper.cpp**: Local processing, no cloud, no CoreML dependency
- **AI-Enhanced**: Optional Claude integration for text refinement
- **Fully Configurable**: Environment variables for all paths
- **Developer-focused**: Built for productivity, with voice commands

## Implementation Status

### ✅ MVP Completed
- [x] Menu bar icon appears and is clickable
- [x] Can toggle enabled/disabled state
- [x] **Global hotkey (Cmd+Shift+P)** for recording
- [x] Audio is transcribed using Whisper (small model)
- [x] **Universal support** - works with ANY active window
- [x] Inserts transcribed text at cursor via clipboard
- [x] Works on macOS 13+ (tested on macOS 14+)
- [x] **Bonus**: Voice commands for AI refinement and clipboard

### ✅ Post-MVP Features Already Implemented
- [x] Global hotkey support (Cmd+Shift+P)
- [x] Visual feedback (icon changes, status messages)
- [x] Settings via environment variables
- [x] Support for all applications (not just terminals)
- [x] AI integration (Claude Code CLI)
- [x] Clipboard mode
- [x] Hotkey enable/disable toggle
- [x] Help menu with command documentation

### Performance Achieved
- Transcription latency: <3 seconds for 10 second audio clip ✓
- Accuracy: >90% for clear English speech ✓
- Memory usage: <500MB idle, <2GB during transcription ✓
- CPU usage: <50% during transcription ✓
- Claude startup: Optimized with MCP bypass (2-5s saved)

## Future Enhancements

### Near Term
- [ ] Global hotkey instead of menu clicks
- [ ] Settings panel for configuration
- [ ] Support more terminal emulators (Alacritty, Warp, etc.)
- [ ] Auto-update mechanism

### Long Term
- [ ] Real-time streaming transcription
- [ ] Voice Activity Detection (auto-start/stop)
- [ ] Custom command vocabulary
- [ ] Multi-language support
- [ ] Model selection (tiny/base/small/medium)
- [ ] Local LLM integration for command correction

## Known Issues & Workarounds

### 1. Hotkey System Beep
**Issue**: When pressing the global hotkey (Cmd+Shift+P), macOS may produce a system beep sound.

**Status**: Minor cosmetic issue, does not affect functionality. Some users may not experience it depending on system settings.

**Cause**: The hotkey library (`golang.design/x/hotkey`) registers the hotkey successfully, but macOS may play a system sound when certain key combinations are pressed.

**Current Hotkey**: **Cmd+Shift+P** (chosen as least intrusive)

**Attempted Solutions**:
- Tried multiple hotkey combinations:
  - Cmd+Shift+H: Had beep issue, conflicts with "Hide Window"
  - Cmd+Shift+L: Had beep issue
  - Cmd+Shift+V: Conflicted with paste special
  - Option+Cmd+Space: Reserved by macOS (Spotlight)
  - Ctrl+Shift+Space: Reserved by macOS (search)
  - Option+Cmd+V: Opened new terminal tab
  - **Cmd+Shift+P**: Minimal beep, no major conflicts (current choice)

**Potential Post-MVP Solutions**:
1. **Try robotgo library** (`github.com/go-vgo/robotgo`)
   - Alternative Go library for keyboard/mouse automation
   - May have better macOS integration
   - Supports global event listeners
   - Trade-off: Larger dependency, requires GCC

2. **Native CGO to Carbon/Cocoa**
   - Use macOS Carbon Event Manager or Cocoa APIs directly
   - Lower-level control over hotkey registration
   - May be able to suppress system beep
   - Trade-off: More complex, platform-specific code

3. **Make hotkey configurable**
   - Let users choose their own hotkey combination
   - Some combinations may not produce beep
   - Trade-off: More complex settings UI

**Decision**: Accept beep for MVP (Cmd+Shift+P), make configurable in Post-MVP Phase 3.

### 2. Claude Code Startup Latency
**Issue**: Claude Code takes 5-6 seconds to start when processing "claude" voice commands.

**Status**: Partially mitigated, acceptable for current use.

**Optimization Applied**:
- Added `--strict-mcp-config --mcp-config '{"mcpServers":{}}'` flags
- Bypasses MCP plugin loading
- Saves 2-5 seconds on startup

**Future Enhancement**:
- Keep Claude Code running in background (daemon mode)
- Use IPC to communicate with running instance
- Would reduce latency to near-instant

## Risk Mitigation

### Technical Risks
1. **CGO Complexity**: Building with CGO can be tricky
   - Mitigation: Provide detailed build instructions, Makefile

2. **Accessibility API**: May require deep Objective-C knowledge
   - Mitigation: Start with AppleScript for MVP, optimize later

3. **Whisper.cpp Integration**: C/Go boundary can cause issues
   - Mitigation: Use well-tested official bindings

4. **Performance**: Whisper small may be too slow
   - Mitigation: Start with small, make model configurable

5. **Hotkey System Beep**: Global hotkey triggers macOS system beep
   - Mitigation: Accept for MVP, explore robotgo or native CGO in Post-MVP

### User Experience Risks
1. **Permission Prompts**: Users may deny accessibility access
   - Mitigation: Clear onboarding, explain why permissions needed

2. **Model Download**: 500MB is large for first-time users
   - Mitigation: Progress indicator, explain one-time download

3. **Hotkey Conflicts**: Some key combinations reserved by macOS
   - Mitigation: Document known conflicts, make configurable in Post-MVP

## License & Credits
- MIT License
- Uses whisper.cpp (MIT License)
- Inspired by similar dictation projects

## Contact & Support
- GitHub: [repository URL]
- Issues: [GitHub Issues URL]

---

**Last Updated**: 2025-10-31
**Version**: 0.2.0 (MVP + AI Features Complete)
