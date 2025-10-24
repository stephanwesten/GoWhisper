#!/bin/bash

# GoWhisper Build Script
# Creates a standalone binary with all dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building GoWhisper standalone binary...${NC}"

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
    echo -e "${RED}Error: whisper.cpp not found at $HOME/.go-whisper/whisper.cpp/${NC}"
    echo "Please rebuild whisper.cpp or run the setup script."
    exit 1
fi

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the binary
echo -e "${YELLOW}Compiling go-whisper...${NC}"
go build -o bin/GoWhisper src/main.go

# Check if build was successful
if [ ! -f "bin/GoWhisper" ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Make it executable
chmod +x bin/GoWhisper

# Create a wrapper script that sets up the environment
echo -e "${YELLOW}Creating run script...${NC}"
cat > bin/run.sh << 'LAUNCHER_EOF'
#!/bin/bash

# GoWhisper Launcher
# Sets up environment and launches the binary

# Set up dynamic library paths
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

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Kill any existing instances
pkill -f "GoWhisper" 2>/dev/null
pkill -f "/exe/main" 2>/dev/null

# Launch the binary
"$SCRIPT_DIR/GoWhisper" > /tmp/go-whisper.log 2>&1 &

# Wait a moment and check if it started
sleep 2
if pgrep -f "GoWhisper" > /dev/null; then
    echo "✅ GoWhisper started successfully!"
    echo "   Press Cmd+Shift+P to start/stop recording"
    echo "   Logs: /tmp/go-whisper.log"
else
    echo "❌ Failed to start GoWhisper"
    echo "   Check logs: /tmp/go-whisper.log"
    exit 1
fi
LAUNCHER_EOF

chmod +x bin/run.sh

# Show binary size
BINARY_SIZE=$(du -h bin/GoWhisper | cut -f1)
echo ""
echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo "Binary location: bin/GoWhisper"
echo "Binary size: $BINARY_SIZE"
echo ""
echo "To run:"
echo "  ./bin/run.sh"
echo ""
echo "Or directly:"
echo "  export DYLD_LIBRARY_PATH=\"\$HOME/.go-whisper/whisper.cpp/build/src:\$HOME/.go-whisper/whisper.cpp/build/ggml/src:\$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-metal:\$HOME/.go-whisper/whisper.cpp/build/ggml/src/ggml-blas\""
echo "  ./bin/GoWhisper"
