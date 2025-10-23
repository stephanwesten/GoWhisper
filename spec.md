# go-whisper: Voice-to-Terminal macOS Menu Bar App

## Overview
A native macOS menu bar application written in Go that provides voice-to-text transcription directly to the active terminal window. The app uses local Whisper AI processing for privacy and runs entirely offline (no cloud dependency).

## Project Goals
- **MVP First**: Simple, working push-to-talk button in menu bar
- **Native & Lightweight**: Pure Go with minimal C dependencies (whisper.cpp)
- **Privacy-focused**: All processing happens locally on the user's machine
- **Terminal-specific**: Optimized for developer workflow, typing commands via voice

## Core Features

### 1. Menu Bar Interface
- Small icon in macOS menu bar (top-right status area)
- Dropdown menu with:
  - **Enable/Disable** toggle for the transcription service
  - **Status indicator** showing current state
  - **Quit** option
- Visual state indication:
  - Different icon or color when enabled vs disabled
  - Recording indicator when actively listening

### 2. Voice Recording (MVP: Push-to-Talk)
- Click menu bar icon → select "Start Recording" button
- Record until user clicks "Stop Recording"
- Alternative: Future enhancement could add global hotkey support
- Audio buffer management for recording sessions

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

### 4. Terminal Detection & Text Insertion
- **Detection Method**: macOS Accessibility APIs
  - Detect currently focused (frontmost) application
  - Verify it's a terminal application (Terminal.app, iTerm2, etc.)
- **Text Insertion**: Insert at cursor position
  - Use Accessibility APIs to send keystrokes
  - Insert transcribed text character-by-character or as paste
- **Error Handling**: Show notification/popup if no terminal is active

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
- Customizable hotkeys

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
fyne.io/systray                                  # Menu bar icon
# OR github.com/getlantern/systray
```

### System Requirements
- macOS 10.15 (Catalina) or later
- Microphone access permission
- Accessibility access permission (for keystroke injection)
- Disk space: ~600MB (app + whisper model)
- RAM: ~2GB during transcription

### External Dependencies
- **whisper.cpp**: Must compile libwhisper.a
  - Build from source or provide pre-built binaries
  - Installation: `make` in whisper.cpp directory
- **PortAudio**: Audio I/O library
  - Install via Homebrew: `brew install portaudio`
- **Whisper Model**: ggml-small.en.bin
  - Download from Hugging Face or OpenAI
  - Bundle with app or download on first run

## Permissions Required

### 1. Microphone Access
- System Preferences → Security & Privacy → Microphone
- App must request permission on first run

### 2. Accessibility Access
- System Preferences → Security & Privacy → Accessibility
- Required for detecting active window and sending keystrokes
- App must be added to the allowed list

### 3. Optional: Input Monitoring
- May be required depending on keystroke injection method
- System Preferences → Security & Privacy → Input Monitoring

## File Structure
```
go-whisper/
├── main.go                    # Entry point, menu bar setup
├── audio/
│   ├── recorder.go           # PortAudio recording
│   └── buffer.go             # Audio buffer management
├── whisper/
│   ├── transcribe.go         # Whisper integration
│   └── model.go              # Model loading/management
├── macos/
│   ├── accessibility.go      # Accessibility API bindings
│   ├── terminal.go           # Terminal detection
│   └── keyboard.go           # Keystroke injection
├── ui/
│   ├── systray.go           # Menu bar UI
│   └── icons.go             # Icon assets
├── models/
│   └── ggml-small.en.bin    # Whisper model (gitignored, downloaded)
├── Makefile                  # Build automation
├── go.mod
├── go.sum
├── README.md
└── spec.md                   # This file
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
- **Terminal-specific**: Validates terminal is active, optimized for command-line
- **Whisper.cpp**: Local processing, no cloud, no CoreML dependency
- **Developer-focused**: Built for typing commands, not general dictation

## Success Criteria

### MVP Success
- [ ] Menu bar icon appears and is clickable
- [ ] Can toggle enabled/disabled state
- [ ] Can record audio via push-to-talk button
- [ ] Audio is transcribed using Whisper (small model)
- [ ] Detects if terminal is active window
- [ ] Shows error if no terminal active
- [ ] Inserts transcribed text at cursor in terminal
- [ ] Works on macOS 13+ (Ventura)

### Performance Targets
- Transcription latency: <3 seconds for 10 second audio clip
- Accuracy: >90% for clear English speech
- Memory usage: <500MB idle, <2GB during transcription
- CPU usage: <50% during transcription

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

### Hotkey System Beep (Pre-MVP)
**Issue**: When pressing the global hotkey (Cmd+Shift+H), macOS produces a system beep sound along with the text injection.

**Status**: Known cosmetic issue, does not affect functionality.

**Cause**: The hotkey library (`golang.design/x/hotkey`) registers the hotkey successfully, but macOS still plays a system sound when the key combination is pressed. This may be related to how the hotkey is intercepted at the system level.

**Attempted Solutions**:
- Tried multiple hotkey combinations:
  - Cmd+Shift+L: Had beep issue
  - Cmd+Shift+V: Conflicted with paste special
  - Option+Cmd+Space: Reserved by macOS (Spotlight)
  - Ctrl+Shift+Space: Reserved by macOS (search)
  - Option+Cmd+V: Opened new terminal tab
  - **Cmd+Shift+H**: Works but has beep (current choice)

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

**Decision**: Accept beep for MVP, revisit in Post-MVP Phase 1.

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

**Last Updated**: 2025-10-23
**Version**: 0.1.0-spec
