#!/bin/bash

# GoWhisper Startup Script
# Starts the voice-to-terminal transcription menu bar app

# Set working directory
cd "$(dirname "$0")"

# Set up CGO environment for whisper.cpp
export CGO_ENABLED=1
export C_INCLUDE_PATH="$HOME/.go-whisper/whisper.cpp/include:$HOME/.go-whisper/whisper.cpp/ggml/include"
export LIBRARY_PATH="$HOME/.go-whisper/whisper.cpp/build/src:$HOME/.go-whisper/whisper.cpp/build/ggml/src:$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-metal:$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-blas"
export CGO_LDFLAGS="-L$HOME/.go-whisper/whisper.cpp/build/src -L$HOME/.go-whisper/whisper.cpp/build/ggml/src -L$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-metal -L$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-blas -lwhisper -lggml -Wl,-rpath,$HOME/.go-whisper/whisper.cpp/build/src -Wl,-rpath,$HOME/.go-whisper/whisper.cpp/build/ggml/src -Wl,-rpath,$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-metal -Wl,-rpath,$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-blas"
export DYLD_LIBRARY_PATH="$HOME/.go-whisper/whisper.cpp/build/src:$HOME/.go-whisper/whisper.cpp/build/ggml/src:$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-metal:$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-blas"

# Check if whisper.cpp exists
if [ ! -d "$HOME/.go-whisper/whisper.cpp/build" ]; then
    echo "Error: whisper.cpp not found at $HOME/.go-whisper/whisper.cpp/"
    echo "Please rebuild whisper.cpp or run the setup script."
    exit 1
fi

# Check if model exists
if [ ! -f "$HOME/.go-whisper/models/ggml-small.en.bin" ]; then
    echo "Error: Whisper model not found at $HOME/.go-whisper/models/ggml-small.en.bin"
    echo "Please download the model first."
    exit 1
fi

# Kill any existing instances
pkill -f "/exe/main" 2>/dev/null

# Start the application in the background
echo "Starting GoWhisper..."
go run src/main.go > /tmp/go-whisper.log 2>&1 &

# Wait a moment and check if it started
sleep 2
if pgrep -f "/exe/main" > /dev/null; then
    echo "✅ GoWhisper started successfully!"
    echo "   Press Cmd+Shift+H to start/stop recording"
    echo "   Logs: /tmp/go-whisper.log"
else
    echo "❌ Failed to start GoWhisper"
    echo "   Check logs: /tmp/go-whisper.log"
    exit 1
fi
