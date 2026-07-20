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
| **ElevenLabs** | Yes | Text-to-speech generation |
| **Deepgram** | Optional | Subtitle generation (speech-to-text) |
| **HeyGen** / **bitHuman** / **Tavus** | Optional | AI avatar presenter overlay (`vac avatar`, `--avatar-id`) |

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
