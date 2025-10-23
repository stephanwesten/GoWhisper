# Proof of Concept Tests

This directory contains test programs used during development to validate individual components.

## test_whisper.go

Tests Whisper.cpp integration with Go bindings.

**Purpose**: Verify that we can load the Whisper model and transcribe audio files.

**Prerequisites**:
- whisper.cpp built in `/tmp/whisper.cpp`
- Whisper model downloaded to `~/.go-whisper/models/ggml-small.en.bin`
- WAV audio file for testing

**Usage**:
```bash
export CGO_ENABLED=1
export C_INCLUDE_PATH="/tmp/whisper.cpp/include:/tmp/whisper.cpp/ggml/include"
export LIBRARY_PATH="/tmp/whisper.cpp/build/src:/tmp/whisper.cpp/build/ggml/src:/tmp/whisper.cpp/build/ggml/src/ggml-metal:/tmp/whisper.cpp/build/ggml/src/ggml-blas"
export CGO_LDFLAGS="-L/tmp/whisper.cpp/build/src -L/tmp/whisper.cpp/build/ggml/src -L/tmp/whisper.cpp/build/ggml/src/ggml-metal -L/tmp/whisper.cpp/build/ggml/src/ggml-blas -lwhisper -lggml -Wl,-rpath,/tmp/whisper.cpp/build/src -Wl,-rpath,/tmp/whisper.cpp/build/ggml/src -Wl,-rpath,/tmp/whisper.cpp/build/ggml/src/ggml-metal -Wl,-rpath,/tmp/whisper.cpp/build/ggml/src/ggml-blas"

go run poc/test_whisper.go ~/.go-whisper/models/ggml-small.en.bin /tmp/whisper.cpp/samples/jfk.wav
```

**Expected output**:
```
=== Transcription ===
[    0s->    8s] And so, my fellow Americans, ask not what your country can do for you.
[    8s->   11s] Ask what you can do for your country.

=== Test completed successfully! ===
```

**Status**: ✅ Working - Whisper integration verified
