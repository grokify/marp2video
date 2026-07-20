# Complete Workflow: Marp to Video with Subtitles

This guide walks through the entire process of creating a narrated video with subtitles from a Marp presentation.

## Overview

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Marp      │ -> │    Audio     │ -> │    Video     │ -> │  Subtitles   │ -> │ Final Video  │
│ Presentation │    │    (TTS)     │    │  Generation  │    │    (STT)     │    │ + Subtitles  │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
```

**Time required:** ~10 minutes for a 5-slide presentation

## Prerequisites

- [vac installed](../getting-started/installation.md)
- [Marp CLI](https://github.com/marp-team/marp-cli) installed
- [ffmpeg](https://ffmpeg.org/) installed
- ElevenLabs API key (for TTS)
- Deepgram API key (for STT/subtitles)

```bash
# Set API keys
export ELEVENLABS_API_KEY="your-elevenlabs-key"
export DEEPGRAM_API_KEY="your-deepgram-key"
```

## Step 1: Create Your Marp Presentation

Create a file called `slides.md`:

```markdown
---
marp: true
theme: default
paginate: true
---

# Introduction to AI Agents

A quick overview of autonomous AI systems

<!-- Welcome to this presentation on AI agents.
     Today we'll explore what makes them unique. -->

---

# What Are AI Agents?

- Autonomous software systems
- Can perceive, decide, and act
- Learn from their environment

<!-- AI agents are autonomous software systems
     that can perceive their environment,
     make decisions, and take actions.
     [PAUSE:500]
     They continuously learn and adapt. -->

---

# Key Components

1. **Perception** - Sensors and data input
2. **Reasoning** - Decision-making logic
3. **Action** - Output and execution

<!-- Every AI agent has three key components.
     First, perception through sensors and data.
     Second, reasoning and decision-making.
     Third, the ability to take action. -->

---

# Real-World Examples

- ChatGPT and Claude for conversation
- Self-driving cars
- Recommendation systems

<!-- You interact with AI agents every day.
     ChatGPT and Claude are conversational agents.
     Self-driving cars use multiple AI systems.
     And recommendation systems on Netflix and Spotify. -->

---

# Thank You

Questions?

<!-- Thanks for watching!
     Feel free to reach out with questions. -->
```

### Key Elements

- **Frontmatter**: `marp: true` enables Marp processing
- **Slide separators**: `---` separates slides
- **Voiceover comments**: `<!-- text -->` contains narration
- **Pause directives**: `[PAUSE:500]` adds 500ms pause

## Step 2: Generate Audio (TTS)

Convert your voiceover text to speech using ElevenLabs:

```bash
# Parse the presentation and generate audio
vac tts --input slides.md --output audio/en-US/
```

This creates:

```
audio/en-US/
├── manifest.json      # Metadata and durations
├── slide_000.mp3      # "Welcome to this presentation..."
├── slide_001.mp3      # "AI agents are autonomous..."
├── slide_002.mp3      # "Every AI agent has..."
├── slide_003.mp3      # "You interact with AI agents..."
└── slide_004.mp3      # "Thanks for watching!"
```

### Optional: Customize Voice

```bash
# Use a specific voice
vac tts --input slides.md --output audio/en-US/ --voice "Rachel"

# Adjust speaking rate
vac tts --input slides.md --output audio/en-US/ --stability 0.5 --similarity 0.75
```

## Step 3: Generate Video

Create the video by combining slides with audio:

```bash
vac video --input slides.md --manifest audio/en-US/manifest.json --output video/presentation.mp4
```

This:

1. Renders slides to PNG images via Marp CLI
2. Creates individual slide videos with synchronized audio
3. Combines all slides into `video/presentation.mp4`

### Optional: Add Subtitles

You can embed subtitles directly during video generation:

```bash
# Embed subtitles (language auto-detected from filename)
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --subtitles subtitles/en-US.srt

# Explicitly specify subtitle language
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --subtitles subtitles/en-US.srt --subtitles-lang en-US
```

Both SRT and VTT formats are supported. The language code is auto-detected from the filename (e.g., `en-US.srt` → `en-US`) if not specified.

### Optional: Add Transitions

```bash
# Add 500ms crossfade between slides
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --transition 500
```

### Optional: Export Individual Slides

```bash
# Also export individual slide videos (for Udemy, etc.)
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --output-individual video/slides/
```

### Optional: Add an Avatar Presenter

Add a talking-head presenter overlay in one shot by setting `--avatar-id`.
The narration is concatenated from the slide audio, a presenter video is
generated via [OmniAvatar](https://github.com/plexusone/omniavatar), and
the circle is composited onto the output automatically:

```bash
export HEYGEN_API_KEY="your-key"
vac slides video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --avatar-id <avatar-id> --avatar-border 6
```

The provider must support audio upload (`heygen` or `bithuman`); for Tavus,
use the decoupled `vac avatar generate --audio-url` flow. See the
[Avatar Presenter guide](avatar-presenter.md) for layout options, caching,
and the standalone `vac avatar generate` / `vac avatar compose` commands.

## Step 4: Generate Subtitles (STT)

Create subtitles from the audio using speech-to-text:

```bash
vac subtitle --audio audio/en-US/
```

This creates:

```
subtitles/
├── en-US.srt      # SubRip format
└── en-US.vtt      # WebVTT format
```

### Example SRT Output

```srt
1
00:00:00,000 --> 00:00:03,500
Welcome to this presentation on AI agents.

2
00:00:03,500 --> 00:00:06,200
Today we'll explore what makes them unique.

3
00:00:07,000 --> 00:00:11,500
AI agents are autonomous software systems
that can perceive their environment,
```

### Optional: Keep Individual Slide Subtitles

```bash
vac subtitle --audio audio/en-US/ --individual
```

## Step 5: Embed Subtitles into Video

Add subtitles as a selectable track (soft subtitles):

```bash
ffmpeg -i video/presentation.mp4 -i subtitles/en-US.srt \
  -c:v copy -c:a copy -c:s mov_text \
  -metadata:s:s:0 language=eng \
  video/presentation_with_subs.mp4
```

Or burn subtitles directly into the video (hard subtitles):

```bash
ffmpeg -i video/presentation.mp4 \
  -vf "subtitles=subtitles/en-US.srt" \
  video/presentation_burned_subs.mp4
```

### Soft vs Hard Subtitles

| Type | Pros | Cons |
|------|------|------|
| **Soft** | Viewer can toggle on/off, multiple languages | Not all players support |
| **Hard** | Always visible, universal compatibility | Cannot be turned off |

## Complete Project Structure

After completing all steps:

```
project/
├── slides.md                           # Source presentation
├── audio/
│   └── en-US/
│       ├── manifest.json
│       └── slide_*.mp3
├── subtitles/
│   ├── en-US.srt
│   └── en-US.vtt
└── video/
    ├── presentation.mp4                # Without subtitles
    ├── presentation_with_subs.mp4      # With soft subtitles
    └── slides/                         # Individual slides (optional)
        ├── slide_000.mp4
        ├── slide_001.mp4
        └── ...
```

## Quick Reference: All Commands

```bash
# 1. Set API keys
export ELEVENLABS_API_KEY="your-key"
export DEEPGRAM_API_KEY="your-key"

# 2. Generate audio
vac tts --input slides.md --output audio/en-US/

# 3. Generate subtitles
vac subtitle --audio audio/en-US/

# 4. Generate video with embedded subtitles
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/presentation.mp4 --subtitles subtitles/en-US.srt
```

### Alternative: Embed Subtitles with ffmpeg

If you prefer to embed subtitles separately (e.g., to add multiple subtitle tracks), you can use ffmpeg:

```bash
# Generate video without subtitles
vac video --input slides.md --manifest audio/en-US/manifest.json --output video/presentation.mp4

# Then embed subtitles (soft)
ffmpeg -i video/presentation.mp4 -i subtitles/en-US.srt \
  -c:v copy -c:a copy -c:s mov_text \
  -metadata:s:s:0 language=eng \
  video/presentation_with_subs.mp4
```

## Multi-Language Workflow

To create videos in multiple languages, see the [multi-language guide](multi-language.md) or use this quick workflow:

```bash
# Create transcript.json with translations
# (see transcript-schema reference)

# Generate audio for each language
vac tts --transcript transcript.json --output audio/en-US/ --lang en-US
vac tts --transcript transcript.json --output audio/fr-FR/ --lang fr-FR
vac tts --transcript transcript.json --output audio/zh-Hans/ --lang zh-Hans

# Generate subtitles for each language
vac subtitle --audio audio/en-US/
vac subtitle --audio audio/fr-FR/
vac subtitle --audio audio/zh-Hans/

# Generate videos with embedded subtitles
vac video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/en-US.mp4 --subtitles subtitles/en-US.srt
vac video --input slides.md --manifest audio/fr-FR/manifest.json \
  --output video/fr-FR.mp4 --subtitles subtitles/fr-FR.srt
vac video --input slides.md --manifest audio/zh-Hans/manifest.json \
  --output video/zh-Hans.mp4 --subtitles subtitles/zh-Hans.srt
```

## Troubleshooting

### Audio generation fails

```bash
# Check API key is set
echo $ELEVENLABS_API_KEY

# Use verbose mode for debugging
vac tts --input slides.md --output audio/en-US/ --verbose
```

### Video is blank or has timing issues

```bash
# Ensure Marp CLI is installed
marp --version

# Check ffmpeg is available
ffmpeg -version

# Use verbose mode
vac video --input slides.md --manifest audio/en-US/manifest.json --output video/test.mp4 --verbose
```

### Subtitle timing is off

The STT transcription provides accurate word-level timing. If subtitles seem misaligned:

1. Check the audio files play correctly
2. Verify manifest.json has correct durations
3. Re-run subtitle generation with `--verbose`

### ffmpeg subtitle embedding fails

```bash
# Check subtitle file encoding (should be UTF-8)
file subtitles/en-US.srt

# Convert if needed
iconv -f ISO-8859-1 -t UTF-8 subtitles/en-US.srt > subtitles/en-US-utf8.srt
```

## Next Steps

- [Customize voice settings](../reference/voice-settings.md)
- [Add custom dictionaries for subtitle capitalization](subtitles.md)
- [Output options for different platforms](output-options.md)
- [Pipeline architecture details](pipeline.md)
