# Requirements

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **OS** | macOS, Linux, Windows | macOS or Linux |
| **RAM** | 4 GB | 8 GB+ |
| **Disk** | 1 GB free | 5 GB+ for video processing |
| **Display** | 1920x1080 | Matches output resolution |

## Software Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| **Go** | 1.21+ | Building vac |
| **ffmpeg** | 4.0+ | Video recording and encoding |
| **Marp CLI** | 3.0+ | Markdown to HTML rendering |
| **Node.js** | 16+ | Required for Marp CLI |
| **Chrome/Chromium** | Latest | Browser automation (auto-managed by Rod) |

## API Keys

| Service | Required | Purpose |
|---------|----------|---------|
| **ElevenLabs** | Yes* | Text-to-speech generation |
| **Deepgram** | Optional | Subtitle generation (speech-to-text) |
| **HeyGen** / **bitHuman** / **Tavus** | Optional | AI avatar presenter overlay (`vac avatar`, `--avatar-id`) |

\* Not required when using the local F5-TTS provider for TTS (and Whisper for
STT) — see [Local Providers](../guide/local-providers.md). Local providers run
entirely on-device with no API keys.

## Local Providers (optional, Apple Silicon)

Local TTS/STT via [OmniVoice](https://github.com/plexusone/omnivoice-core) MLX
providers requires:

| Dependency | Version | Purpose |
|------------|---------|---------|
| **Apple Silicon** | M1/M2/M3/M4 | MLX runs GPU inference on the Neural Engine / GPU |
| **Python (arm64)** | 3.11+ | Runs the F5-TTS / Whisper gRPC servers (MLX wheels are arm64-only) |
| **Disk** | ~4 GB free | F5-TTS (~2 GB) and Whisper `large-v3-turbo` (~1.6 GB) model weights |

The `scripts/localvoice.sh` helper sets these up automatically. See
[Local Providers](../guide/local-providers.md).

## Platform-Specific Notes

### macOS

- Screen recording permission required (System Preferences > Privacy)
- Apple Silicon (M1/M2/M3) fully supported
- Uses `avfoundation` for screen capture

### Linux

- X11 display server required (Wayland not yet supported)
- Uses `x11grab` for screen capture
- May need `pulseaudio` for audio

### Windows

- Uses `gdigrab` for screen capture
- Administrator privileges may be required
