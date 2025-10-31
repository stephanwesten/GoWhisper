# GoWhisper

A MacOS menu bar application for voice-to-text transcription using OpenAI's Whisper model with GPU acceleration, 
targeted at using terminal / Claude Code.

Press **Cmd+Shift+P** to start/stop recording, and your speech will be transcribed and typed into the active window.
The letter P is chosen as it is close to the Enter key.

**Platform Support**: Currently only tested on MacOS (M1 Pro). Linux/Windows support is not tested.

**Voice Commands**: Use special keywords to modify behavior:
- Say **"clipboard [your text]"** to copy transcribed text to clipboard instead of typing
- Say **"claude [your text]"** to have Claude AI rephrase your text for better grammar and clarity
- Say **"clipboard claude [your text]"** (or reverse order) to rephrase with Claude AND copy to clipboard

DISCLAIMER: this is a hobby project and by no means professional software. Most of the code is vibe-coded. 

KNOWN ISSUES
* Sometimes the hot key is incorrectly processed and for example a terminal inspector is shown.

---

## 🤖 Claude Setup Instructions

<details>
<summary>Click to expand automated setup commands for Claude</summary>

If you're Claude Code and a user asks you to set up this project, execute these commands:

```bash
# 1. Install PortAudio
brew install portaudio

# 2. Clone and build whisper.cpp
cd /tmp
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp
cmake -B build -DWHISPER_METAL=ON
cmake --build build

# 3. Move to permanent location
mkdir -p ~/.go-whisper
cp -r /tmp/whisper.cpp ~/.go-whisper/

# 4. Download Whisper model
mkdir -p ~/.go-whisper/models
curl -L -o ~/.go-whisper/models/ggml-small.en.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin

# 5. Build the application
cd /path/to/go-whisper
./build.sh

# 6. Launch
./bin/run.sh
```

**Note**: Whisper.cpp build takes ~5 minutes. The model download is 487MB.

</details>

---

## Features

- 🎤 Push-to-talk recording with global hotkey (Cmd+Shift+P)
- 🧠 Local transcription using Whisper.cpp with Metal GPU acceleration
- 🤖 Claude AI integration for text rephrasing and improvement
- 📋 Clipboard mode for copying instead of typing
- ⚡ Fast processing with multi-threading support
- 🔄 Multiple consecutive recordings supported
- 📝 Automatic text insertion into active window
- 🔴 Visual recording indicator in menu bar
- 🔕 Enable/disable hotkey toggle (useful during presentations)
- ⚠️  User-friendly error dialogs

## Setup

### Prerequisites

- macOS (only tested on M1 Pro - other platforms not supported)
- Homebrew
- Go 1.21 or later
- Xcode Command Line Tools
- Claude CLI (optional - only needed for Claude rephrasing feature)

### Installation Steps

**1. Install PortAudio**
```bash
brew install portaudio
```

**2. Clone and build whisper.cpp**
```bash
# Clone whisper.cpp repository
cd /tmp
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp

# Build with Metal GPU support
cmake -B build -DWHISPER_METAL=ON
cmake --build build

# Move to permanent location
mkdir -p ~/.go-whisper
cp -r /tmp/whisper.cpp ~/.go-whisper/
```

**3. Download Whisper model**
```bash
# Create models directory
mkdir -p ~/.go-whisper/models

# Download the small English model (487MB)
cd ~/.go-whisper/models
curl -L -o ggml-small.en.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin
```

**4. Clone this repository**
```bash
git clone <your-repo-url>
cd go-whisper
```

**5. Build the application**
```bash
./build.sh
```

**6. Run GoWhisper**
```bash
./bin/run.sh
```

## Usage

### Basic Recording

1. Launch the application using `./bin/run.sh`
2. Look for "◉" in your menu bar
3. Press **Cmd+Shift+P** to start recording (indicator changes to blinking 🔴/⭕)
4. Speak clearly into your microphone
5. Press **Cmd+Shift+P** again to stop recording
6. The transcribed text will be typed into your active window

### Voice Command Examples

**Normal transcription:**
- Press Cmd+Shift+P
- Say: "This is a test message"
- Press Cmd+Shift+P
- Result: "This is a test message" is typed into active window

**Clipboard mode:**
- Press Cmd+Shift+P
- Say: "clipboard this is a test message"
- Press Cmd+Shift+P
- Result: "this is a test message" is copied to clipboard (not typed)

**Claude rephrasing mode:**
- Press Cmd+Shift+P
- Say: "claude hey this is test message want to make better"
- Press Cmd+Shift+P
- Result: Claude rephrases to "Hey, this is a test message that I would like to improve." and types it

**Combined mode (Claude + Clipboard):**
- Press Cmd+Shift+P
- Say: "clipboard claude fix grammar in this sentence"
- Press Cmd+Shift+P
- Result: Claude rephrases the text and copies it to clipboard (not typed)

### Keyword Detection Rules

- Keywords must appear in the **first 2 words** of your speech
- Detection is **case-insensitive** (clipboard, Clipboard, CLIPBOARD all work)
- Keywords can appear in **any order** when combined
- Keywords are **automatically removed** from the final output

### Menu Bar Controls

- **⌘⇧P Menu Item**: Click to start/stop recording (same as hotkey)
- **Disable/Enable Hotkey**: Toggle to temporarily disable the global hotkey (useful during Zoom presentations)
- **Quit**: Exit the application

## Stopping/Restarting the Application

**IMPORTANT**: This is a menu bar application that runs in the background. Multiple instances may exist if you've built the app using different methods.

**To reliably stop ALL instances:**
```bash
# Kill all go-whisper processes (works for all binary names)
pkill -9 -f "go-whisper|GoWhisper"
```

**To verify all instances are stopped:**
```bash
ps aux | grep -E "go-whisper|GoWhisper" | grep -v grep
```

**Recommended restart workflow:**
```bash
# 1. Kill all instances
pkill -9 -f "go-whisper|GoWhisper"

# 2. Verify nothing is running (should return no results)
ps aux | grep -E "go-whisper|GoWhisper" | grep -v grep

# 3. Rebuild and run
./build.sh
./bin/run.sh
```

**Why multiple instances can occur:**
- Running `go build -o go-whisper src/main.go` creates `./go-whisper` in the root directory
- Running `./build.sh` creates `./bin/GoWhisper`
- Both binaries can run simultaneously if not properly cleaned up
- The `-f` flag in `pkill` matches the full command line, catching both naming variations

## Permissions

On first use, you'll need to grant:

- **Microphone access**: Required for recording audio
- **Accessibility permissions**: Required for typing text into active windows
  - Go to: System Settings → Privacy & Security → Accessibility
  - Add your Terminal app to the allowed list

## Troubleshooting

**"No speech detected"**
- Speak louder or closer to the microphone
- Check your microphone input levels in System Settings
- Audio amplitude should be above 0.3 for reliable detection

**"osascript is not allowed to send keystrokes"**
- You need to grant Accessibility permissions (see Permissions section above)
- An error dialog will guide you through this

**App won't start after reboot**
- Whisper.cpp and the model are in `~/.go-whisper/` and will survive reboots
- Just run `./bin/run.sh` again

## Architecture

- **Audio Recording**: PortAudio for microphone capture (16kHz mono)
- **Transcription**: Whisper.cpp with Metal GPU acceleration
- **AI Rephrasing**: Claude CLI for text improvement (optional)
- **UI**: systray for menu bar integration
- **Hotkeys**: golang.design/x/hotkey for global keyboard shortcuts
- **Text Input**: AppleScript for typing into active windows
- **Clipboard**: github.com/atotto/clipboard for clipboard operations

## Development

To run from source:
```bash
./start-go-whisper.sh
```

To rebuild the binary:
```bash
./build.sh
```

## License

MIT
