# CLI Reference

Complete command-line interface reference.

## Command Structure

vac uses a hierarchical command structure:

```
vac
├── slides              # Marp slide presentations
│   ├── video          # Full pipeline: parse, TTS, record, combine
│   └── tts            # Generate audio from transcript
├── browser            # Browser automation recordings
│   ├── video          # Record with TTS voiceover
│   └── record         # Silent recording (no audio)
├── avatar             # Talking-head presenter overlay (optional)
│   ├── list-avatars   # List provider avatar IDs (for --avatar-id)
│   ├── generate       # Narration audio -> presenter video
│   └── compose        # Overlay presenter circle onto slides video
└── subtitle           # Generate subtitles from audio
```

---

## slides video

Generate video from Marp presentation (full pipeline).

```bash
vac slides video [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-i, --input` | string | *required* | Input Marp markdown file |
| `-o, --output` | string | `output.mp4` | Output video file |
| `-m, --manifest` | string | | Audio manifest file (from `slides tts`) |
| `-k, --api-key` | string | `$ELEVENLABS_API_KEY` | ElevenLabs API key |
| `-v, --voice` | string | `pNInz6obpgDQGcFmaJgB` | ElevenLabs voice ID (Adam) |
| `--width` | int | `1920` | Video width in pixels |
| `--height` | int | `1080` | Video height in pixels |
| `--fps` | int | `30` | Video frame rate |
| `--transition` | float | `0` | Transition duration (seconds) |
| `--subtitles` | string | | Subtitle file to embed (SRT or VTT) |
| `--subtitles-lang` | string | auto-detect | Subtitle language code |
| `--output-individual` | string | | Directory for individual slide videos |
| `--screen-device` | string | auto-detect | macOS screen capture device |
| `--workdir` | string | system temp | Working directory for temp files |
| `--check` | bool | | Verify dependencies and exit |

#### Avatar overlay flags (optional)

Setting `--avatar-id` enables an integrated talking-head presenter overlay
as a final pipeline stage — the one-shot equivalent of running
`vac avatar generate` + `vac avatar compose` by hand. The narration is
concatenated from the slide audio and uploaded, so the provider must
support audio upload (`heygen` or `bithuman`); for Tavus, use the
decoupled `vac avatar generate --audio-url` flow.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--avatar-id` | string | | Enable overlay with this avatar identity (heygen `avatar_id` / bithuman `agent_id`) |
| `--avatar-provider` | string | `heygen` | Avatar provider: `heygen` or `bithuman` |
| `--avatar-api-key` | string | from env | Avatar provider API key |
| `--avatar-ext` | key=value | | Provider-specific request option (repeatable) |
| `--avatar-poll` | duration | `5s` | Job status poll interval |
| `--avatar-no-cache` | bool | `false` | Disable presenter video caching |
| `--avatar-diameter` | int | `320` | Circle diameter in pixels |
| `--avatar-position` | string | `bottom-right` | `bottom-right`, `bottom-left`, `top-right`, `top-left` |
| `--avatar-margin-x` | int | `56` | Horizontal margin in pixels |
| `--avatar-margin-y` | int | `56` | Vertical margin in pixels |
| `--avatar-border` | int | `0` | Border ring width in pixels (0 disables) |
| `--avatar-border-color` | string | `white` | Border ring color |

### Examples

```bash
# Full pipeline with inline voiceovers
vac slides video --input slides.md --output video.mp4

# Use pre-generated audio
vac slides video --input slides.md --manifest audio/manifest.json --output video.mp4

# With transitions and custom resolution
vac slides video --input slides.md --output video.mp4 \
  --transition 0.5 --width 1280 --height 720

# Generate individual slide videos for Udemy
vac slides video --input slides.md --output combined.mp4 \
  --output-individual ./lectures/

# One-shot with an avatar presenter overlay (HeyGen)
export HEYGEN_API_KEY=...
vac slides video --input slides.md --manifest audio/en-US/manifest.json \
  --output final.mp4 --avatar-id <avatar-id> --avatar-border 6

# Check dependencies
vac slides video --check
```

---

## slides tts

Generate audio files from a transcript JSON file.

```bash
vac slides tts [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-t, --transcript` | string | *required* | Transcript JSON file |
| `-o, --output` | string | `audio` | Output directory for audio files |
| `-l, --lang` | string | from transcript | Language/locale code (e.g., `en-US`) |
| `--provider` | string | auto-detect | TTS provider: `elevenlabs`, `deepgram`, or `f5tts-mlx` |
| `--elevenlabs-api-key` | string | `$ELEVENLABS_API_KEY` | ElevenLabs API key |
| `--deepgram-api-key` | string | `$DEEPGRAM_API_KEY` | Deepgram API key |
| `--local` | bool | `false` | Enable local TTS providers (F5-TTS MLX; Apple Silicon) |
| `--f5tts-endpoint` | string | `unix:///tmp/omnivoice-f5tts.sock` | F5-TTS MLX gRPC endpoint |
| `-f, --force` | bool | `false` | Regenerate audio even if files exist |

### Examples

```bash
# Generate English audio
vac slides tts --transcript transcript.json --output audio/en-US/ --lang en-US

# Generate Spanish audio with Deepgram
vac slides tts --transcript transcript.json --output audio/es-ES/ \
  --lang es-ES --provider deepgram

# Generate audio locally with F5-TTS (no API key; requires the local server)
vac slides tts --transcript transcript.json --output audio/en-US/ \
  --lang en-US --provider f5tts-mlx --local

# Force regeneration
vac slides tts --transcript transcript.json --output audio/ --force
```

See [Local Providers](../guide/local-providers.md) for the `--local` server setup.

---

## browser video

Record browser-driven demos with AI-generated voiceover.

```bash
vac browser video [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-c, --config` | string | *required* | Configuration file (YAML/JSON) |
| `-o, --output` | string | `output.mp4` | Output video file |
| `-a, --audio-dir` | string | | Directory to save/reuse audio tracks |
| `-p, --provider` | string | auto-detect | TTS provider: `elevenlabs`, `deepgram`, or `f5tts-mlx` |
| `-v, --voice` | string | from config | TTS voice ID |
| `-l, --lang` | string | `en-US` | Languages to generate (comma-separated) |
| `--elevenlabs-api-key` | string | `$ELEVENLABS_API_KEY` | ElevenLabs API key |
| `--deepgram-api-key` | string | `$DEEPGRAM_API_KEY` | Deepgram API key |
| `--local` | bool | `false` | Enable local TTS providers (F5-TTS MLX; Apple Silicon) |
| `--f5tts-endpoint` | string | `unix:///tmp/omnivoice-f5tts.sock` | F5-TTS MLX gRPC endpoint |
| `--width` | int | `1920` | Video width in pixels |
| `--height` | int | `1080` | Video height in pixels |
| `--fps` | int | `30` | Video frame rate |
| `--transition` | float | `0` | Transition duration (seconds) |
| `--headless` | bool | `false` | Run browser in headless mode |
| `--subtitles` | bool | `false` | Generate subtitles from voiceover timing |
| `--subtitles-stt` | bool | `false` | Generate word-level subtitles using STT |
| `--subtitles-burn` | bool | `false` | Burn subtitles into video (requires FFmpeg with libass) |
| `--no-audio` | bool | `false` | Generate video without audio (TTS used for timing/subtitles) |
| `--fast` | bool | `false` | Use hardware-accelerated encoding (VideoToolbox on macOS) |
| `--limit` | int | `0` | Limit to first N segments (0 = no limit, for testing) |
| `--limit-steps` | int | `0` | Limit browser segments to first N steps (0 = no limit, for testing) |
| `--workdir` | string | system temp | Working directory for temp files |

### Examples

```bash
# Basic browser demo
vac browser video --config demo.yaml --output demo.mp4

# Multi-language with audio caching
vac browser video --config demo.yaml --output demo.mp4 \
  --audio-dir ./audio --lang en-US,fr-FR,zh-Hans

# With subtitles burned in (requires FFmpeg with libass)
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn

# Silent video with burned subtitles (no audio track)
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn --no-audio

# Headless mode for CI/CD
vac browser video --config demo.yaml --output demo.mp4 --headless

# Using Deepgram TTS
vac browser video --config demo.yaml --output demo.mp4 --provider deepgram

# Fast encoding with hardware acceleration (macOS VideoToolbox)
vac browser video --config demo.yaml --output demo.mp4 --fast

# Test with limited segments (faster iteration)
vac browser video --config demo.yaml --output demo.mp4 --limit 2

# Test with limited browser steps (faster iteration)
vac browser video --config demo.yaml --output demo.mp4 --limit-steps 3
```

### Audio Caching

When using `--audio-dir`, vac caches generated TTS audio:

- Audio files stored as `{audio-dir}/{language}/segment_XXX.mp3`
- Metadata JSON files store per-voiceover timing information
- Subsequent runs skip TTS generation if cached audio exists

### Multi-Language Timing

When generating multiple languages, the video is paced to the longest audio:

1. TTS audio is generated for all requested languages
2. Per-voiceover durations are compared across languages
3. Each browser step uses the maximum duration
4. All language versions sync with the same video

---

## browser record

Record browser session without audio (silent recording).

```bash
vac browser record [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-c, --config` | string | | Configuration file (YAML/JSON) |
| `-s, --steps` | string | | Steps file defining browser actions |
| `-u, --url` | string | | Starting URL for the browser |
| `-o, --output` | string | `recording.mp4` | Output video file |
| `--width` | int | `1920` | Browser viewport width |
| `--height` | int | `1080` | Browser viewport height |
| `--fps` | int | `30` | Video frame rate |
| `--headless` | bool | `false` | Run browser in headless mode |
| `-t, --timing` | string | | Output timing JSON file |
| `--timeout` | int | `30000` | Default step timeout (ms) |
| `--workdir` | string | system temp | Working directory |
| `--cleanup` | bool | `true` | Clean up temp files after recording |

### Examples

```bash
# Record from steps file
vac browser record --url https://example.com --steps demo.json --output demo.mp4

# Record from config file
vac browser record --config demo.yaml --output demo.mp4

# Export timing data for later audio sync
vac browser record --url https://example.com --steps demo.json \
  --output demo.mp4 --timing timing.json

# Headless mode
vac browser record --url https://example.com --steps demo.json \
  --output demo.mp4 --headless
```

---

## subtitle

Generate subtitles from audio files using speech-to-text.

```bash
vac subtitle [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-a, --audio` | string | *required* | Audio directory containing manifest.json |
| `-o, --output` | string | `subtitles` | Output directory for subtitle files |
| `-l, --lang` | string | from manifest | Language code |
| `--provider` | string | `deepgram` | STT provider: `deepgram`, `elevenlabs`, or `whisper-mlx` |
| `--local` | bool | `false` | Enable local STT providers (Whisper MLX; Apple Silicon) |
| `--whisper-endpoint` | string | `unix:///tmp/omnivoice-whisper.sock` | Whisper MLX gRPC endpoint |
| `--individual` | bool | `false` | Also generate per-slide subtitle files |

### Examples

```bash
# Generate subtitles (language auto-detected)
vac subtitle --audio audio/en-US/

# Custom output directory
vac subtitle --audio audio/fr-FR/ --output subs/

# Transcribe locally with Whisper (no API key; requires the local server)
vac subtitle --audio audio/en-US/ --lang en-US --provider whisper-mlx --local

# Keep individual slide subtitles
vac subtitle --audio audio/en-US/ --individual
```

See [Local Providers](../guide/local-providers.md) for the `--local` server setup.

---

## avatar list-avatars

List avatar IDs available from a provider, for use as `--avatar-id`.
Provider avatar IDs are provider-native and not always the same as the
IDs returned by a provider's dashboard or other list endpoints — for
HeyGen, `--avatar-id` needs a **v2 avatar ID** (e.g.
`Abigail_expressive_2024112501`), which this command returns.

```bash
vac avatar list-avatars [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-p, --provider` | string | `heygen` | Avatar provider: `heygen`, `tavus`, or `bithuman` |
| `-k, --api-key` | string | from env | Provider API key |
| `--search` | string | | Filter by ID or name substring (case-insensitive) |
| `--limit` | int | `50` | Maximum avatars to print (0 = all) |

### Examples

```bash
# Find HeyGen avatars matching "abigail"
vac avatar list-avatars --provider heygen --search abigail

# Dump the full catalog
vac avatar list-avatars --provider heygen --limit 0 > avatars.txt
```

!!! note "Provider support"
    Listing requires a provider that implements avatar discovery
    (`omniavatar.AvatarLister`). HeyGen is supported; the HeyGen catalog
    is large, so the first response can take a moment.

---

## avatar generate

Generate a talking-head presenter video from narration audio using an AI
avatar provider ([OmniAvatar](https://github.com/plexusone/omniavatar):
HeyGen, Tavus, or bitHuman). Optional — presentations render exactly as
before unless the avatar commands are used.

```bash
vac avatar generate [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-m, --manifest` | string | | Audio manifest from `vac slides tts` (concatenated with pause gaps) |
| `--audio` | string | | Narration audio file (MP3 recommended) |
| `--audio-url` | string | | Pre-hosted narration audio URL |
| `-p, --provider` | string | `heygen` | Avatar provider: `heygen`, `tavus`, or `bithuman` |
| `--avatar-id` | string | *required* | Avatar identity (heygen avatar_id / tavus replica_id / bithuman agent_id) |
| `-k, --api-key` | string | from env | Provider API key |
| `-o, --output` | string | `presenter.mp4` | Output presenter video file |
| `--poll` | duration | `5s` | Job status poll interval |
| `--cache-dir` | string | user cache dir | Presenter video cache directory |
| `--no-cache` | bool | `false` | Disable presenter video caching |
| `--ext` | key=value | | Provider-specific request option (repeatable) |

Exactly one of `--manifest`, `--audio`, or `--audio-url` is required.
With `--manifest`, per-slide audio is concatenated **including pause
gaps** so avatar lip-sync matches the final video timeline. Generated
videos are cached by narration content + avatar configuration.

!!! note "Provider limitations"
    Tavus has no audio upload API and requires `--audio-url`. HeyGen's
    upload API documents MP3 (`audio/mpeg`) as its supported audio type.

### Examples

```bash
# Generate presenter video from the TTS manifest
vac avatar generate --manifest audio/en-US/manifest.json \
  --provider heygen --avatar-id <avatar-id> --output presenter.mp4

# HeyGen test mode (watermarked, no credits)
vac avatar generate --audio narration.mp3 --provider heygen \
  --avatar-id <avatar-id> --ext test=true --output presenter.mp4
```

---

## avatar compose

Composite a talking-head presenter video as a circular overlay onto a
slides video. The circle mask is applied locally with FFmpeg, so output
is identical across avatar providers. The presenter video's own audio is
always discarded.

```bash
vac avatar compose [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slides` | string | *required* | Slides (base) video file |
| `--avatar` | string | *required* | Presenter (avatar) video file |
| `--audio` | string | | Narration audio to use as the authoritative audio track |
| `-o, --output` | string | `final.mp4` | Output video file |
| `--diameter` | int | `320` | Avatar circle diameter in pixels |
| `--position` | string | `bottom-right` | `bottom-right`, `bottom-left`, `top-right`, `top-left` |
| `--margin-x` | int | `56` | Horizontal margin in pixels |
| `--margin-y` | int | `56` | Vertical margin in pixels |
| `--border` | int | `0` | Border ring width in pixels (0 disables) |
| `--border-color` | string | `white` | Border ring color (ffmpeg color name or 0xRRGGBB) |

### Examples

```bash
# Composite with narration as the authoritative audio track
vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
  --audio narration.mp3 --output final.mp4

# Smaller circle, bottom-left, with a border ring
vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
  --output final.mp4 --diameter 280 --position bottom-left --border 6
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ELEVENLABS_API_KEY` | ElevenLabs API key for TTS |
| `DEEPGRAM_API_KEY` | Deepgram API key for TTS/STT |
| `HEYGEN_API_KEY` | HeyGen API key for avatar generation |
| `TAVUS_API_KEY` | Tavus API key for avatar generation |
| `BITHUMAN_API_KEY` | bitHuman API key for avatar generation |

---

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Missing dependencies |
| 3 | Invalid input file |
| 4 | TTS generation failed |
| 5 | Recording failed |
| 6 | Video combination failed |
