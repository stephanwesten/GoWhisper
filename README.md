# GoWhisper

A macOS menu bar application for voice-to-text transcription using OpenAI's Whisper model with GPU acceleration.

Press **Cmd+Shift+P** to start/stop recording, and your speech will be transcribed and typed into the active window.

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
- ⚡ Fast processing with multi-threading support
- 🔄 Multiple consecutive recordings supported
- 📝 Automatic text insertion into active window
- 🔴 Visual recording indicator in menu bar
- ⚠️  User-friendly error dialogs

## Setup

### Prerequisites

- macOS (tested on M1 Pro)
- Homebrew
- Go 1.21 or later
- Xcode Command Line Tools

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

1. Launch the application using `./bin/run.sh`
2. Look for "GW ●" in your menu bar
3. Press **Cmd+Shift+P** to start recording (indicator changes to "GW 🔴")
4. Speak clearly into your microphone
5. Press **Cmd+Shift+P** again to stop recording
6. The transcribed text will be typed into your active window

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
- **UI**: systray for menu bar integration
- **Hotkeys**: golang.design/x/hotkey for global keyboard shortcuts
- **Text Input**: AppleScript for typing into active windows

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
