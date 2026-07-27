# Subtitle Generation

vac can generate subtitle files (SRT/VTT) in two ways:

1. **STT-based**: Using speech-to-text for word-level accuracy (Marp slides)
2. **Timing-based**: Using voiceover timing without STT (Browser videos)

## Quick Start

=== "Marp Slides (STT)"

    ```bash
    # Generate subtitles using speech-to-text
    vac subtitle --audio audio/en-US/

    # Output:
    # subtitles/en-US.srt
    # subtitles/en-US.vtt
    ```

=== "Browser Video (No STT)"

    ```bash
    # Generate subtitles from voiceover timing
    vac browser video --config demo.yaml --output demo.mp4 \
      --subtitles

    # Output:
    # demo.srt (alongside demo.mp4)
    ```

## How It Works

1. **Audio input**: Reads MP3 files from `audio/{lang}/`
2. **Speech-to-text**: Uses Deepgram (default) or local Whisper STT for word-level timing
3. **Subtitle generation**: Creates SRT and VTT files with accurate timestamps

### Timing Accuracy

The `subtitle` command uses an STT provider to transcribe the audio files and extract word-level timestamps. This provides accurate subtitle timing that matches the actual speech patterns, rather than estimating from text length. Deepgram is the default; ElevenLabs and local Whisper are also supported via `--provider`.

### Local, offline transcription (Whisper)

On Apple Silicon you can transcribe entirely on-device with the local Whisper provider — no API key required:

```bash
# Requires the local Whisper server (see the Local Providers guide)
vac subtitle --audio audio/en-US/ --lang en-US --provider whisper-mlx --local
```

BCP-47 locales like `en-US` are accepted; the region subtag is stripped automatically to the ISO-639-1 code Whisper expects (`en`). See [Local Providers](local-providers.md) for server setup.

## Command Reference

```
vac subtitle [flags]

Flags:
  -a, --audio string        Audio directory containing manifest.json (required)
  -o, --output string       Output directory for subtitle files (default "subtitles")
  -l, --lang string         Language code (auto-detected from manifest if not specified)
      --provider string     STT provider: deepgram, elevenlabs, or whisper-mlx (default: deepgram)
      --local               Enable local STT providers (Whisper MLX; Apple Silicon)
      --whisper-endpoint string  Whisper MLX gRPC endpoint (default unix:///tmp/omnivoice-whisper.sock)
      --individual          Also generate individual subtitle files per slide
```

### Examples

```bash
# Generate French subtitles
vac subtitle --audio audio/fr-FR/

# Custom output directory
vac subtitle --audio audio/zh-Hans/ --output subs/

# Keep individual slide subtitle files
vac subtitle --audio audio/en-US/ --individual
```

## Multi-Language Workflow

```bash
# Step 1: Generate audio for each language
vac tts --transcript transcript.json --output audio/en-US/ --lang en-US
vac tts --transcript transcript.json --output audio/fr-FR/ --lang fr-FR

# Step 2: Generate subtitles for each language
vac subtitle --audio audio/en-US/
vac subtitle --audio audio/fr-FR/

# Step 3: Generate videos
vac video --input slides.md --manifest audio/en-US/manifest.json --output video/en-US.mp4
vac video --input slides.md --manifest audio/fr-FR/manifest.json --output video/fr-FR.mp4
```

### Output Structure

```
project/
├── audio/
│   ├── en-US/
│   │   ├── manifest.json
│   │   └── slide_*.mp3
│   └── fr-FR/
│       ├── manifest.json
│       └── slide_*.mp3
├── subtitles/
│   ├── en-US.srt
│   ├── en-US.vtt
│   ├── fr-FR.srt
│   └── fr-FR.vtt
└── video/
    ├── en-US.mp4
    └── fr-FR.mp4
```

## Browser Video Subtitles

The `browser-video` command supports built-in subtitle generation:

### Options

| Flag | Description | Requirements |
|------|-------------|--------------|
| `--subtitles` | Generate subtitles from voiceover timing | None |
| `--subtitles-stt` | Generate word-level subtitles using STT | Deepgram API |
| `--subtitles-burn` | Burn subtitles into video (permanent) | FFmpeg with libass |
| `--no-audio` | Generate video without audio (TTS used for timing) | None |

!!! warning "FFmpeg libass Requirement"
    The `--subtitles-burn` flag requires FFmpeg compiled with libass support.
    Check with: `ffmpeg -filters 2>&1 | grep subtitles`

    If not available, install via:
    ```bash
    # macOS
    brew uninstall ffmpeg
    brew tap homebrew-ffmpeg/ffmpeg
    brew install homebrew-ffmpeg/ffmpeg/ffmpeg

    # Linux (Ubuntu/Debian)
    sudo apt install ffmpeg libass-dev
    ```

### Examples

```bash
# Simple subtitles from voiceover timing (no API cost)
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles

# Word-level subtitles using speech-to-text
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles-stt

# Burn subtitles permanently into video
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn

# Silent video with burned subtitles (no audio track)
# Useful for demos where viewers read subtitles instead of listening
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn --no-audio
```

### How Timing-Based Subtitles Work

When using `--subtitles` (without `--subtitles-stt`):

1. Each voiceover text becomes a subtitle entry
2. Long text is automatically split into 2-line chunks (max 42 chars per line)
3. Start/end times are calculated from TTS audio durations with word-based timing
4. Pauses between voiceovers are accounted for
5. No additional API calls required

### Automatic Text Chunking

Long voiceover text is automatically split into readable subtitle chunks:

- **Max 2 lines per chunk** - Standard for video subtitles
- **Max 42 characters per line** - Optimized for 1080p display
- **Word-aware splitting** - Text breaks at word boundaries, not mid-word
- **Proportional timing** - Each chunk's duration is based on word count, not character count

Example: A 100-word voiceover becomes multiple 2-line subtitle entries, each timed proportionally based on the words it contains.

This approach provides sentence-level accuracy and is ideal when:

- You want to avoid STT API costs
- Your voiceover text matches what should appear as subtitles
- You're iterating quickly on content

---

## Current Implementation: Standard Subtitles

The current implementation generates **standard subtitles** - the most professional and widely-used format:

```
┌─────────────────────────────────────┐
│                                     │
│         [Slide Content]             │
│                                     │
│  ─────────────────────────────────  │
│  Two types of AI users are emerging │
└─────────────────────────────────────┘
```

**Characteristics:**

- White text with black outline or semi-transparent background
- 1-2 lines maximum (42 chars per line), positioned at bottom of screen
- Automatic text chunking for long voiceovers
- Word-based timing distribution for natural reading pace
- VFR to CFR conversion ensures reliable timing when burning subtitles
- Industry standard for professional content

**Used by:** Netflix, YouTube, broadcast TV, Udemy, Coursera

---

## Future Caption Styles

The following caption styles are planned for future implementation. They offer different trade-offs between engagement and professionalism.

### Karaoke Style (Word Highlight)

Words change color/highlight as they are spoken:

```
Two types of [AI users] are emerging
              ^^^^^^^^^ (highlighted in yellow as spoken)
```

**Characteristics:**

- Words highlight in sequence as spoken
- Requires word-level timestamps (available via Deepgram)
- More engaging than static subtitles
- Can be distracting for technical content

**Best for:** Music videos, language learning, accessibility features

**Implementation notes:**

- Requires word-level timing from STT (already available)
- Output format: ASS/SSA (Advanced SubStation Alpha) for styling
- Or: Burn into video using ffmpeg drawtext filter

### Word-by-Word Reveal (Social Media Style)

Words appear one at a time as spoken:

```
Frame 1: Two
Frame 2: Two types
Frame 3: Two types of
Frame 4: Two types of AI
...
```

**Characteristics:**

- Highly engaging, attention-grabbing
- Popular on TikTok, Instagram Reels, YouTube Shorts
- Less formal, not suitable for all content types
- Often combined with animations

**Best for:** Social media clips, promotional videos, short-form content

**Implementation notes:**

- Requires word-level timing
- Output: Burned into video (not separate subtitle file)
- Consider text positioning, font size, animations

### Animated Captions (CapCut/Premiere Style)

Words animate in with effects (pop, slide, bounce):

```
     ╭─────────────╮
     │  AI USERS   │  ← pops in with scale animation
     ╰─────────────╯
```

**Characteristics:**

- Very engaging, trendy aesthetic
- Complex to implement, requires video editing
- Not suitable for professional/corporate content
- Popular with content creators

**Best for:** Social media, entertainment, creator content

**Implementation notes:**

- Requires video compositing (ffmpeg complex filters or external tool)
- Template-based approach for consistency
- Consider offering preset animation styles

---

## Implementation Roadmap

| Style | Status | Priority | Notes |
|-------|--------|----------|-------|
| Standard (SRT/VTT) | ✅ Implemented | - | Current default |
| Burned-in standard | ✅ Implemented | - | `--subtitles-burn` flag |
| Timing-based (no STT) | ✅ Implemented | - | `browser-video --subtitles` |
| Karaoke highlight | 🔲 Planned | Medium | ASS format + ffmpeg |
| Word-by-word reveal | 🔲 Planned | Medium | Social media use case |
| Animated captions | 🔲 Planned | Low | Complex, may use templates |

### Proposed CLI Extensions

```bash
# Burn subtitles into video (standard style)
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --subtitles subtitles/en-US.srt --output video/en-US.mp4

# Karaoke style (future)
vac subtitle --audio audio/en-US/ --style karaoke --output subtitles/

# Word-by-word reveal (future)
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --caption-style reveal --output video/en-US.mp4
```

---

## Technical Considerations

### Subtitle Formats

| Format | Extension | Features | Use Case |
|--------|-----------|----------|----------|
| SRT | `.srt` | Basic timing + text | Universal compatibility |
| VTT | `.vtt` | Timing + basic styling | Web video players |
| ASS/SSA | `.ass` | Full styling, positioning, effects | Karaoke, anime fansubs |
| TTML | `.ttml` | XML-based, broadcast standard | Broadcast, streaming |

### Burning Subtitles into Video

```bash
# Using ffmpeg with SRT
ffmpeg -i video.mp4 -vf "subtitles=subtitles.srt" output.mp4

# With custom styling
ffmpeg -i video.mp4 -vf "subtitles=subtitles.srt:force_style='FontSize=24,PrimaryColour=&HFFFFFF&'" output.mp4

# ASS format for full control
ffmpeg -i video.mp4 -vf "ass=subtitles.ass" output.mp4
```

### Word-Level Timing

Deepgram STT returns word-level timestamps:

```json
{
  "words": [
    {"word": "Two", "start": 0.0, "end": 0.2},
    {"word": "types", "start": 0.25, "end": 0.5},
    {"word": "of", "start": 0.52, "end": 0.6},
    {"word": "AI", "start": 0.65, "end": 0.9}
  ]
}
```

This data is already captured during subtitle generation and can be used for karaoke and word-reveal styles.
