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

# Allow override via environment variables with sensible defaults
: ${GOWHISPER_INSTALL_DIR:="$HOME/.go-whisper"}

# Set up CGO environment for whisper.cpp
export CGO_ENABLED=1
export C_INCLUDE_PATH="$GOWHISPER_INSTALL_DIR/whisper.cpp/include:$GOWHISPER_INSTALL_DIR/whisper.cpp/ggml/include"
export LIBRARY_PATH="$GOWHISPER_INSTALL_DIR/whisper.cpp/build/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-metal:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-blas"
export CGO_LDFLAGS="-L$GOWHISPER_INSTALL_DIR/whisper.cpp/build/src -L$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src -L$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-metal -L$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-blas -lwhisper -lggml -Wl,-rpath,$GOWHISPER_INSTALL_DIR/whisper.cpp/build/src -Wl,-rpath,$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src -Wl,-rpath,$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-metal -Wl,-rpath,$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-blas"
export DYLD_LIBRARY_PATH="$GOWHISPER_INSTALL_DIR/whisper.cpp/build/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-metal:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-blas"

# Check if whisper.cpp exists
if [ ! -d "$GOWHISPER_INSTALL_DIR/whisper.cpp/build" ]; then
    echo -e "${RED}Error: whisper.cpp not found at $GOWHISPER_INSTALL_DIR/whisper.cpp/${NC}"
    echo "Please rebuild whisper.cpp or run the setup script."
    echo ""
    echo "Tip: You can set GOWHISPER_INSTALL_DIR to use a different location:"
    echo "  export GOWHISPER_INSTALL_DIR=/path/to/installation"
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

# Allow override via environment variables with sensible defaults
: ${GOWHISPER_INSTALL_DIR:="$HOME/.go-whisper"}
: ${GOWHISPER_MODEL:="$GOWHISPER_INSTALL_DIR/models/ggml-small.en.bin"}
: ${GOWHISPER_LOG:="/tmp/go-whisper.log"}

# Export for the Go application
export GOWHISPER_MODEL

# Set up dynamic library paths
export DYLD_LIBRARY_PATH="$GOWHISPER_INSTALL_DIR/whisper.cpp/build/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-metal:$GOWHISPER_INSTALL_DIR/whisper.cpp/build/ggml/src/ggml-blas"

# Check if whisper.cpp exists
if [ ! -d "$GOWHISPER_INSTALL_DIR/whisper.cpp/build" ]; then
    echo "Error: whisper.cpp not found at $GOWHISPER_INSTALL_DIR/whisper.cpp/"
    echo "Please rebuild whisper.cpp or run the setup script."
    echo ""
    echo "Tip: You can set GOWHISPER_INSTALL_DIR to use a different location:"
    echo "  export GOWHISPER_INSTALL_DIR=/path/to/installation"
    exit 1
fi

# Check if model exists
if [ ! -f "$GOWHISPER_MODEL" ]; then
    echo "Error: Whisper model not found at $GOWHISPER_MODEL"
    echo "Please download the model first."
    echo ""
    echo "Tip: You can set GOWHISPER_MODEL to use a different model:"
    echo "  export GOWHISPER_MODEL=/path/to/model.bin"
    exit 1
fi

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Kill any existing instances
pkill -9 -f "go-whisper|GoWhisper" 2>/dev/null

# Launch the binary
"$SCRIPT_DIR/GoWhisper" > "$GOWHISPER_LOG" 2>&1 &

# Wait a moment and check if it started
sleep 2
if pgrep -f "GoWhisper" > /dev/null; then
    echo "✅ GoWhisper started successfully!"
    echo "   Press Cmd+Shift+P to start/stop recording"
    echo "   Model: $GOWHISPER_MODEL"
    echo "   Logs: $GOWHISPER_LOG"
else
    echo "❌ Failed to start GoWhisper"
    echo "   Check logs: $GOWHISPER_LOG"
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
