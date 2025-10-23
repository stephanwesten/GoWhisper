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

---

## test_audio_record.go

Tests audio recording from microphone using PortAudio.

**Purpose**: Verify that we can capture audio from the microphone and save it to a WAV file in the format required by Whisper (16kHz, mono, 16-bit PCM).

**Prerequisites**:
- PortAudio installed: `brew install portaudio`
- Microphone access permission granted to Terminal

**Usage**:
```bash
go run poc/test_audio_record.go <duration_seconds> <output.wav>

# Example: Record 5 seconds of audio
go run poc/test_audio_record.go 5 /tmp/test_recording.wav
```

**Expected output**:
```
2025/10/23 19:54:49 Recording for 5 seconds to /tmp/test_recording.wav...
2025/10/23 19:54:49 Speak into your microphone!
2025/10/23 19:54:57 Recording time completed
2025/10/23 19:54:57 Recorded 79995 samples (5.00 seconds)
2025/10/23 19:54:57 Audio saved to /tmp/test_recording.wav
2025/10/23 19:54:57 ✅ Test completed successfully!
```

**Testing with Whisper**:
After recording, you can test transcription:
```bash
# Record audio
go run poc/test_audio_record.go 5 /tmp/test.wav

# Transcribe with Whisper (use env vars from test_whisper.go)
go run poc/test_whisper.go ~/.go-whisper/models/ggml-small.en.bin /tmp/test.wav
```

**Audio format**:
- Sample rate: 16kHz (Whisper requirement)
- Channels: Mono (1 channel)
- Bit depth: 16-bit PCM
- Format: WAV

**Status**: ✅ Working - Audio recording verified and tested with Whisper transcription
