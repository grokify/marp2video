# Local Avatar Demo

Demonstrates the fully offline video generation pipeline using local TTS and
avatar rendering on Apple Silicon.

## Pipeline

```
presentation.md
       │
       ▼
┌─────────────────┐
│   F5-TTS MLX    │  Text-to-speech (local)
└────────┬────────┘
         │ narration.wav
         ▼
┌─────────────────┐
│ LivePortrait +  │  Audio-driven avatar (local)
│    JoyVASA      │
└────────┬────────┘
         │ presenter.mp4
         ▼
┌─────────────────┐
│  Whisper MLX    │  Subtitles (local)
└────────┬────────┘
         │ subtitles.srt
         ▼
┌─────────────────┐
│     FFmpeg      │  Composition
└────────┬────────┘
         │
         ▼
    output.mp4
```

## Prerequisites

### 1. Start Local Servers

**F5-TTS server** (text-to-speech):

```bash
cd ~/go/src/github.com/plexusone/omnivoice-core/providers/f5tts-mlx/server
source .venv/bin/activate
./run.sh
```

**LivePortrait + JoyVASA server** (avatar rendering):

```bash
cd ~/go/src/github.com/plexusone/omniavatar-core/providers/liveportrait-joyvasa/server
source .venv/bin/activate
./run.sh
```

### 2. Create Avatar Bundle

Create an avatar bundle at `~/.omniavatar/avatars/example/`:

```bash
mkdir -p ~/.omniavatar/avatars/example/idle

# Add metadata.json
cat > ~/.omniavatar/avatars/example/metadata.json << 'EOF'
{
  "name": "example",
  "fps": 25,
  "resolution": {"width": 512, "height": 512}
}
EOF

# Add your idle video clip
cp your-idle-clip.mp4 ~/.omniavatar/avatars/example/idle/idle.mp4
```

The idle clip should be:
- 3-10 seconds of neutral, mouth-closed footage
- Shot at eye level with good lighting
- 512x512 or higher resolution

## Running the Demo

### Full Pipeline (TTS + Avatar + Subtitles)

```bash
cd examples/local-avatar

# Generate with local providers
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

On Apple Silicon (M-series):

| Stage | Time (6 slides) |
|-------|-----------------|
| TTS (F5-TTS MLX) | ~30 seconds |
| Avatar (LivePortrait + JoyVASA) | ~5 minutes |
| Subtitles (Whisper MLX) | ~10 seconds |
| Composition (FFmpeg) | ~5 seconds |

Total: approximately 6 minutes for a 6-slide presentation.

## Troubleshooting

### Server Connection Failed

Verify the servers are running:

```bash
ls -la /tmp/omnivoice-f5tts.sock
ls -la /tmp/omniavatar-liveportrait-joyvasa.sock
```

### Out of Memory

The avatar model requires ~4GB. Close other memory-intensive applications.

### Model Loading Slow

First run downloads model weights (~4GB total). Subsequent runs use cached weights.

## Output

After running, you'll have:

- `output.mp4` — Final video with avatar overlay and subtitles
- `audio/` — Generated TTS audio files
- `presenter.mp4` — Avatar video (if running step-by-step)
