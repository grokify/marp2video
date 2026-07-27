# Local Avatar (Offline)

Fully offline video generation using local TTS and avatar rendering on Apple Silicon.

## Overview

This example demonstrates the complete offline pipeline:

1. **F5-TTS MLX** — Text-to-speech synthesis
2. **LivePortrait + JoyVASA** — Audio-driven avatar rendering
3. **Whisper MLX** — Speech-to-text for subtitles
4. **FFmpeg** — Video composition

No cloud APIs or network requests required after initial model download.

## Prerequisites

### Start Local Servers

**F5-TTS server:**

```bash
cd ~/go/src/github.com/plexusone/omnivoice-core/providers/f5tts-mlx/server
source .venv/bin/activate
./run.sh
```

**LivePortrait + JoyVASA server:**

```bash
cd ~/go/src/github.com/plexusone/omniavatar-core/providers/liveportrait-joyvasa/server
source .venv/bin/activate
./run.sh
```

### Create Avatar Bundle

```bash
mkdir -p ~/.omniavatar/avatars/example/idle

cat > ~/.omniavatar/avatars/example/metadata.json << 'EOF'
{
  "name": "example",
  "fps": 25,
  "resolution": {"width": 512, "height": 512}
}
EOF

# Add your idle video clip (3-10s, neutral, mouth-closed)
cp your-idle-clip.mp4 ~/.omniavatar/avatars/example/idle/idle.mp4
```

## Running the Example

### Full Pipeline

```bash
cd examples/local-avatar

vac slides video \
  --input presentation.md \
  --tts-provider f5tts-mlx \
  --avatar-provider liveportrait-joyvasa \
  --avatar-id example \
  --subtitles-provider whisper-mlx \
  --output output.mp4
```

### Step by Step

```bash
# 1. Generate TTS audio
vac slides tts \
  --input presentation.md \
  --provider f5tts-mlx \
  --output-dir audio/

# 2. Generate avatar video
vac avatar generate \
  --manifest audio/manifest.json \
  --provider liveportrait-joyvasa \
  --avatar-id example \
  --output presenter.mp4

# 3. Generate subtitles
vac stt \
  --input audio/concatenated.wav \
  --provider whisper-mlx \
  --output subtitles.srt

# 4. Composite final video
vac slides video \
  --input presentation.md \
  --manifest audio/manifest.json \
  --avatar-video presenter.mp4 \
  --subtitles subtitles.srt \
  --output output.mp4
```

## Performance

On Apple Silicon (M-series), for a 6-slide presentation:

| Stage | Time |
|-------|------|
| TTS (F5-TTS MLX) | ~30s |
| Avatar (LivePortrait + JoyVASA) | ~5 min |
| Subtitles (Whisper MLX) | ~10s |
| Composition (FFmpeg) | ~5s |

**Total:** ~6 minutes

## Source Files

- [`presentation.md`](https://github.com/grokify/videoascode/blob/main/examples/local-avatar/presentation.md) — 6-slide Marp presentation
- [`README.md`](https://github.com/grokify/videoascode/blob/main/examples/local-avatar/README.md) — Detailed setup instructions

## See Also

- [Local Providers Guide](../guide/local-providers.md) — F5-TTS and Whisper setup
- [Avatar Presenter Overlay](../guide/avatar-presenter.md) — Avatar integration options
