# VideoAsCode (vac)

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/videoascode/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/videoascode/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/videoascode/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/videoascode/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/videoascode/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/videoascode/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/videoascode
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/videoascode
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-guide-blue.svg
 [docs-mkdoc-url]: https://grokify.github.io/videoascode/
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fvideoascode
 [loc-svg]: https://tokei.rs/b1/github/grokify/videoascode
 [repo-url]: https://github.com/grokify/videoascode
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/videoascode/blob/main/LICENSE

Convert Marp presentations with voiceovers to video files.

This tool takes a Marp markdown presentation with voiceover text (inline comments or JSON transcript), generates speech using text-to-speech (TTS), and creates a synchronized video recording of the presentation with optional subtitles.

**Powered by [OmniVoice](https://github.com/plexusone/omnivoice)** - a unified interface for TTS/STT providers. Additional providers beyond those listed below are available via OmniVoice. Tested with:

- **[ElevenLabs](https://elevenlabs.io/)** - Known for high-quality AI voices (TTS and STT available)
- **[Deepgram](https://deepgram.com/)** - Known for fast, accurate transcription (STT and TTS available)

Both providers offer TTS and STT capabilities. You can use either one for both functions, though vac defaults to ElevenLabs for voice generation and Deepgram for subtitle transcription based on their respective strengths. See the [OmniVoice repository](https://github.com/plexusone/omnivoice) for the full list of supported providers.

## Features

- 📝 **Parse Marp presentations** with voiceover in HTML comments
- 🌐 **JSON transcript support** for multi-language voiceovers
- 🎙️ **Text-to-speech** via OmniVoice (ElevenLabs, Deepgram)
- 🗣️ **Multi-language** support with per-slide voice configuration
- 🖼️ **Image-based rendering** using Marp PNG export for reliable output
- 🎬 **Video generation** with synchronized audio using ffmpeg
- 💻 **Cross-platform** support (macOS, Linux, Windows)
- ⏱️ **Pause directives** like `[PAUSE:1000]` for timing control
- ⚙️ **Full orchestration** - entire process automated in Go
- ▶️ **YouTube-ready** combined video output with optional transitions
- 🎓 **Udemy-ready** individual slide videos for course lectures
- 🔄 **Decoupled workflow** - generate audio and video separately
- 📜 **Subtitle generation** - SRT/VTT from audio via OmniVoice (Deepgram STT)
- 🌐 **Browser demo recording** - automated browser interactions with voiceover
- 🎯 **TTS audio caching** - reuse generated audio across runs
- 🧑‍💼 **Avatar presenter overlay** (optional) - talking-head circle via [OmniAvatar](https://github.com/plexusone/omniavatar) (HeyGen, Tavus, bitHuman)
- ⚡ **Hardware acceleration** - fast encoding with VideoToolbox (macOS)

## Installation

### Prerequisites

1. **Go 1.21+**
   ```bash
   go version
   ```

2. **ffmpeg** (for video recording and processing)
   ```bash
   # macOS
   brew install ffmpeg

   # Linux
   sudo apt install ffmpeg

   # Windows
   # Download from https://ffmpeg.org/download.html
   ```

3. **Marp CLI** (for rendering presentations)
   ```bash
   npm install -g @marp-team/marp-cli
   ```

4. **ElevenLabs API Key** (for TTS)
   - Sign up at [ElevenLabs](https://elevenlabs.io/)
   - Get your API key from the dashboard

5. **Deepgram API Key** (for subtitle generation)
   - Sign up at [Deepgram](https://deepgram.com/)
   - Get your API key from the console

### Build from Source

```bash
git clone https://github.com/grokify/videoascode
cd videoascode
go build -o bin/vac ./cmd/vac
```

## Usage

vac provides two main command groups:

**Marp Slides:**

- `vac slides video` - Full pipeline: parse slides, generate TTS, record, combine
- `vac slides tts` - Generate audio from JSON transcript

**Browser Recording:**

- `vac browser video` - Record browser demo with TTS voiceover
- `vac browser record` - Record browser demo (silent, no audio)

**Avatar Presenter (optional):**

- `vac slides video --avatar-id ...` - One-shot: slides + presenter overlay in a single command
- `vac avatar list-avatars` - List provider avatar IDs (for `--avatar-id`)
- `vac avatar generate` - Generate talking-head presenter video from narration audio
- `vac avatar compose` - Composite the presenter circle onto a slides video

**Utilities:**

- `vac subtitle` - Generate subtitles from audio using STT

### Quick Start (Full Pipeline)

```bash
# Set API keys
export ELEVENLABS_API_KEY="your-elevenlabs-key"  # For TTS
export DEEPGRAM_API_KEY="your-deepgram-key"      # For subtitles (optional)

# Using inline voiceover comments
vac slides video --input slides.md --output video.mp4
```

### Two-Step Workflow (Recommended for Multi-Language)

```bash
# Step 1: Generate audio from transcript
vac slides tts --transcript transcript.json --output audio/en-US/ --lang en-US

# Step 2: Generate video with pre-generated audio
vac slides video --input slides.md --manifest audio/en-US/manifest.json --output video/en-US.mp4
```

### Command: `vac slides tts`

Generate audio files from a JSON transcript.

```
vac slides tts [flags]

Flags:
  -t, --transcript string   Transcript JSON file (required)
  -o, --output string       Output directory for audio files (default "audio")
  -l, --lang string         Language/locale code (e.g., en-US, es-ES)
      --provider string     TTS provider: elevenlabs or deepgram
```

**Output:**

- `audio/{lang}/slide_000.mp3`, `slide_001.mp3`, ... (one per slide)
- `audio/{lang}/manifest.json` (timing information for video recording)

**Example:**

```bash
# Generate audio for Spanish
vac slides tts --transcript transcript.json --output audio/es-ES/ --lang es-ES
```

### Command: `vac slides video`

Generate video from Marp presentation.

```
vac slides video [flags]

Flags:
  -i, --input string              Input Marp markdown file (required)
  -o, --output string             Output video file (default "output.mp4")
  -m, --manifest string           Audio manifest file (from 'vac slides tts')
  -k, --api-key string            ElevenLabs API key (or use ELEVENLABS_API_KEY env var)
  -v, --voice string              ElevenLabs voice ID (default: Adam)
      --width int                 Video width (default 1920)
      --height int                Video height (default 1080)
      --fps int                   Frame rate (default 30)
      --transition float          Transition duration in seconds
      --subtitles string          Subtitle file to embed (SRT or VTT)
      --subtitles-lang string     Subtitle language code (auto-detected from filename)
      --output-individual string  Directory for individual slide videos
      --workdir string            Working directory for temp files
      --screen-device string      Screen capture device (macOS)
      --check                     Check dependencies and exit

  Avatar overlay (optional; set --avatar-id to enable):
      --avatar-id string          Avatar identity (enables the overlay stage)
      --avatar-provider string    heygen or bithuman (default "heygen")
      --avatar-api-key string     Provider API key (or use the provider's env var)
      --avatar-diameter int       Circle diameter in pixels (default 320)
      --avatar-position string    bottom-right, bottom-left, top-right, top-left
      --avatar-border int         Border ring width in pixels (0 disables)
      --avatar-ext stringArray    Provider-specific request option key=value (repeatable)
      --avatar-no-cache           Disable presenter video caching
```

With `--avatar-id`, the presenter overlay runs as an integrated final
stage (narration concat → generate → composite). The provider must
support audio upload (`heygen` or `bithuman`); for Tavus use the
decoupled `vac avatar generate --audio-url` flow. See
[`vac avatar generate`](#command-vac-avatar-generate) for the standalone
commands.

### Command: `vac subtitle`

Generate subtitle files (SRT/VTT) from audio files using speech-to-text.

```
vac subtitle [flags]

Flags:
  -a, --audio string        Audio directory containing manifest.json (required)
  -o, --output string       Output directory for subtitle files (default "subtitles")
  -l, --lang string         Language code (auto-detected from manifest if not specified)
      --provider string     STT provider: deepgram or elevenlabs (default: deepgram)
      --individual          Also generate individual subtitle files per slide
```

**Output:**

- `subtitles/{lang}.srt` - SRT format subtitle file
- `subtitles/{lang}.vtt` - WebVTT format subtitle file

**Example:**

```bash
# Generate French subtitles (language auto-detected from manifest)
vac subtitle --audio audio/fr-FR/

# Generate with explicit language and custom output
vac subtitle --audio audio/zh-Hans/ --lang zh-Hans --output subs/
```

### Command: `vac avatar generate`

Generate a talking-head presenter video from narration audio using an AI avatar provider ([OmniAvatar](https://github.com/plexusone/omniavatar): HeyGen, Tavus, or bitHuman). This feature is optional — presentations render exactly as before unless the avatar commands are used.

```
vac avatar generate [flags]

Flags:
  -m, --manifest string     Audio manifest from 'vac slides tts' (concatenated with pause gaps)
      --audio string        Narration audio file (MP3 recommended)
      --audio-url string    Pre-hosted narration audio URL
  -p, --provider string     Avatar provider: heygen, tavus, or bithuman (default "heygen")
      --avatar-id string    Avatar identity (heygen avatar_id / tavus replica_id / bithuman agent_id) (required)
  -k, --api-key string      Provider API key (or use the provider's env var)
  -o, --output string       Output presenter video file (default "presenter.mp4")
      --poll duration       Job status poll interval (default 5s)
      --cache-dir string    Presenter video cache directory (default: user cache dir)
      --no-cache            Disable presenter video caching
      --ext stringArray     Provider-specific request option as key=value (repeatable)
```

API keys are read from `HEYGEN_API_KEY`, `TAVUS_API_KEY`, or `BITHUMAN_API_KEY`. Exactly one of `--manifest`, `--audio`, or `--audio-url` is required. With `--manifest`, per-slide audio is concatenated **including pause gaps** so the avatar lip-sync matches the final video timeline. Generated videos are cached by narration content + avatar configuration, so re-runs with unchanged audio are free.

Note: Tavus has no audio upload API, so it requires `--audio-url`. HeyGen's upload API documents MP3 (`audio/mpeg`) as its supported audio type.

**Example:**

```bash
# Generate presenter video from the TTS manifest
vac avatar generate --manifest audio/en-US/manifest.json \
  --provider heygen --avatar-id <avatar-id> --output presenter.mp4

# HeyGen test mode (watermarked, no credits)
vac avatar generate --audio narration.mp3 --provider heygen \
  --avatar-id <avatar-id> --ext test=true --output presenter.mp4
```

### Command: `vac avatar compose`

Composite a talking-head presenter video as a circular overlay onto a slides video. The circle mask is applied locally with FFmpeg, so output is identical across avatar providers. The presenter video's own audio is always discarded; pass `--audio` to use the narration file as the authoritative audio track.

```
vac avatar compose [flags]

Flags:
      --slides string        Slides (base) video file (required)
      --avatar string        Presenter (avatar) video file (required)
      --audio string         Narration audio file to use as the authoritative audio track
  -o, --output string        Output video file (default "final.mp4")
      --diameter int         Avatar circle diameter in pixels (default 320)
      --position string      Overlay position: bottom-right, bottom-left, top-right, top-left (default "bottom-right")
      --margin-x int         Horizontal margin in pixels (default 56)
      --margin-y int         Vertical margin in pixels (default 56)
      --border int           Border ring width in pixels (0 disables)
      --border-color string  Border ring color (ffmpeg color name or 0xRRGGBB) (default "white")
```

**Example:**

```bash
# Full workflow: narration -> presenter -> slides -> final video
vac slides tts --transcript transcript.json --output audio/
vac avatar generate --manifest audio/en-US/manifest.json \
  --provider heygen --avatar-id <id> --output presenter.mp4
vac slides video --input slides.md --manifest audio/en-US/manifest.json --output slides.mp4
vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
  --audio narration.mp3 --output final.mp4 --border 6
```

### Command: `vac browser video`

Record browser-driven demos with AI-generated voiceover. This command automates browser interactions (navigation, clicks, scrolling) while generating synchronized narration.

```
vac browser video [flags]

Flags:
  -c, --config string           Configuration file (YAML/JSON) with browser segments (required)
  -o, --output string           Output video file (default "output.mp4")
  -a, --audio-dir string        Save/reuse audio tracks in this directory (per-language subdirs)
  -p, --provider string         TTS provider: elevenlabs or deepgram (default: auto-detect)
  -v, --voice string            TTS voice ID (default: from config or provider default)
  -l, --lang string             Languages to generate, comma-separated (default "en-US")
      --elevenlabs-api-key      ElevenLabs API key (or use ELEVENLABS_API_KEY env var)
      --deepgram-api-key        Deepgram API key (or use DEEPGRAM_API_KEY env var)
      --width int               Video width (default 1920)
      --height int              Video height (default 1080)
      --fps int                 Video frame rate (default 30)
      --headless                Run browser in headless mode
      --transition float        Transition duration between segments (seconds)
      --subtitles               Generate subtitles from voiceover timing (no STT)
      --subtitles-stt           Generate word-level subtitles using STT (requires API)
      --subtitles-burn          Burn subtitles into video (permanent, requires FFmpeg with libass)
      --no-audio                Generate video without audio (TTS still used for timing/subtitles)
      --fast                    Use hardware-accelerated encoding (VideoToolbox on macOS)
      --limit int               Limit to first N segments (for testing)
      --limit-steps int         Limit browser segments to first N steps (for testing)
      --workdir string          Working directory for temp files
```

### Command: `vac browser record`

Record browser session without audio (silent recording).

```
vac browser record [flags]

Flags:
  -c, --config string     Configuration file (YAML/JSON) with segments
  -s, --steps string      Steps file (JSON/YAML) defining browser actions
  -u, --url string        Starting URL for the browser
  -o, --output string     Output video file (default "recording.mp4")
      --width int         Browser viewport width (default 1920)
      --height int        Browser viewport height (default 1080)
      --fps int           Video frame rate (default 30)
      --headless          Run browser in headless mode
  -t, --timing string     Output timing JSON file for transcript sync
      --workdir string    Working directory for temp files
```

**Key Features:**

- **Multi-language support**: Generate videos in multiple languages with `--lang en-US,fr-FR,zh-Hans`
- **Audio caching**: Use `--audio-dir` to cache TTS audio and skip regeneration on subsequent runs
- **Pace to longest language**: Video timing automatically matches the longest audio across all languages
- **Per-voiceover timing**: Each browser step is paced to its corresponding voiceover duration
- **Subtitle generation**: Create SRT/VTT subtitles from voiceover timing or word-level STT

**Example Config (demo.yaml):**

```yaml
metadata:
  title: "Product Demo"
  defaultLanguage: "en-US"

defaultVoice:
  provider: "elevenlabs"
  voiceId: "pNInz6obpgDQGcFmaJgB"

segments:
  - id: "segment_000"
    type: "browser"
    browser:
      url: "https://example.com"
      steps:
        - action: "wait"
          duration: 1000
          voiceover:
            en-US: "Welcome to our product demo."
            fr-FR: "Bienvenue dans notre démonstration."
        - action: "click"
          selector: "#login-button"
          voiceover:
            en-US: "Click the login button to get started."
            fr-FR: "Cliquez sur le bouton de connexion."
        - action: "scroll"
          scrollY: 500
          voiceover:
            en-US: "Scroll down to see more features."
            fr-FR: "Faites défiler pour voir plus de fonctionnalités."
```

**Example Usage:**

```bash
# Basic browser demo recording
vac browser video --config demo.yaml --output demo.mp4

# Multi-language with audio caching
vac browser video --config demo.yaml --output demo.mp4 \
  --audio-dir ./audio --lang en-US,fr-FR,zh-Hans

# With subtitles burned into video (requires FFmpeg with libass)
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn

# Video with burned subtitles but no audio (for silent demos)
vac browser video --config demo.yaml --output demo.mp4 \
  --subtitles --subtitles-burn --no-audio

# Using Deepgram instead of ElevenLabs
vac browser video --config demo.yaml --output demo.mp4 \
  --provider deepgram

# Silent browser recording (no audio)
vac browser record --url https://example.com --steps demo.json --output demo.mp4

# Fast encoding with hardware acceleration (macOS VideoToolbox)
vac browser video --config demo.yaml --output demo.mp4 --fast

# Test first 2 segments only (faster iteration)
vac browser video --config demo.yaml --output demo.mp4 --limit 2

# Test first 3 steps of browser segment (faster iteration)
vac browser video --config demo.yaml --output demo.mp4 --limit-steps 3
```

**Output Structure:**

When using `--audio-dir` and multiple languages:

```
project/
├── demo.yaml
├── demo.mp4              # Primary language video
├── demo_fr-FR.mp4        # French version
├── demo_zh-Hans.mp4      # Chinese version
├── demo.srt              # Subtitles (if --subtitles)
└── audio/
    ├── en-US/
    │   ├── segment_000.mp3
    │   ├── segment_000.json    # Cached timing metadata
    │   └── combined.mp3
    ├── fr-FR/
    │   └── ...
    └── zh-Hans/
        └── ...
```

### Examples

**Full pipeline with inline voiceovers:**

```bash
vac slides video \
  --input presentation.md \
  --output youtube_video.mp4 \
  --transition 0.5
```

**Multi-language workflow:**

```bash
# Step 1: Generate audio for each language (directory matches locale code)
vac slides tts --transcript transcript.json --output audio/en-US/ --lang en-US
vac slides tts --transcript transcript.json --output audio/es-ES/ --lang es-ES
vac slides tts --transcript transcript.json --output audio/zh-Hans/ --lang zh-Hans

# Step 2: Generate subtitles for each language (uses Deepgram STT)
vac subtitle --audio audio/en-US/
vac subtitle --audio audio/es-ES/
vac subtitle --audio audio/zh-Hans/

# Step 3: Generate videos with embedded subtitles
vac slides video --input slides.md --manifest audio/en-US/manifest.json \
  --output video/en-US.mp4 --subtitles subtitles/en-US.srt
vac slides video --input slides.md --manifest audio/es-ES/manifest.json \
  --output video/es-ES.mp4 --subtitles subtitles/es-ES.srt
vac slides video --input slides.md --manifest audio/zh-Hans/manifest.json \
  --output video/zh-Hans.mp4 --subtitles subtitles/zh-Hans.srt
```

**Directory structure** (locale codes enable automation):

```
project/
├── presentation.md
├── transcript.json
├── audio/
│   ├── en-US/
│   │   ├── manifest.json
│   │   └── slide_*.mp3
│   ├── es-ES/
│   │   ├── manifest.json
│   │   └── slide_*.mp3
│   └── zh-Hans/
│       ├── manifest.json
│       └── slide_*.mp3
├── subtitles/
│   ├── en-US.srt
│   ├── en-US.vtt
│   ├── es-ES.srt
│   ├── es-ES.vtt
│   ├── zh-Hans.srt
│   └── zh-Hans.vtt
└── video/
    ├── en-US.mp4
    ├── es-ES.mp4
    └── zh-Hans.mp4
```

**Generate individual videos for Udemy:**

```bash
vac slides video \
  --input presentation.md \
  --output combined.mp4 \
  --output-individual ./udemy_videos/
```

### Check Dependencies

```bash
vac slides video --check
```

This will verify that all required tools (ffmpeg, marp) are installed.

## Voiceover Formats

vac supports two voiceover formats:

1. **Inline HTML comments** - Simple, single-language
2. **JSON transcript** - Multi-language, advanced TTS control

### Option 1: Inline Voiceover Comments

Add voiceover text in HTML comments before or after slide content:

```markdown
---
marp: true
---

<!--
This is the voiceover for the first slide.
It will be converted to speech using ElevenLabs.
[PAUSE:1000]
You can add pause directives for timing control.
-->

# First Slide

This is the visible content

---

<!--
Voiceover for slide 2...
-->

# Second Slide

More content
```

#### Pause Directives

Use `[PAUSE:milliseconds]` to add pauses in the voiceover:

```markdown
<!--
First sentence.
[PAUSE:1000]
Second sentence after a 1-second pause.
[PAUSE:2000]
Third sentence after a 2-second pause.
-->
```

The pause directives are automatically removed from the spoken text.

### Option 2: JSON Transcript

For multi-language support and advanced TTS configuration, use a JSON transcript file:

```json
{
  "version": "1.0",
  "metadata": {
    "title": "My Presentation",
    "defaultLanguage": "en-US",
    "defaultVoice": {
      "provider": "elevenlabs",
      "voiceId": "pNInz6obpgDQGcFmaJgB",
      "voiceName": "Adam",
      "model": "eleven_multilingual_v2",
      "stability": 0.5,
      "similarityBoost": 0.75
    },
    "defaultVenue": "youtube"
  },
  "slides": [
    {
      "index": 0,
      "title": "Title Slide",
      "transcripts": {
        "en-US": {
          "segments": [
            { "text": "Welcome to the presentation.", "pause": 500 },
            { "text": "Let's get started." }
          ]
        },
        "es-ES": {
          "voice": {
            "voiceId": "onwK4e9ZLuTAKqWW03F9",
            "voiceName": "Daniel"
          },
          "segments": [
            { "text": "Bienvenido a la presentación.", "pause": 500 },
            { "text": "Comencemos." }
          ]
        }
      }
    }
  ]
}
```

#### Transcript Features

| Feature | Description |
|---------|-------------|
| **Multi-language** | Per-slide transcripts for each locale (en-US, es-ES, etc.) |
| **Voice override** | Different voice per language or segment |
| **Pause control** | Pause after each segment (milliseconds) |
| **Venue presets** | Optimized settings for YouTube, Udemy, Coursera |
| **TTS parameters** | Stability, similarity boost, style exaggeration |

#### Audio Manifest

When using `vac tts`, a manifest is generated with timing info:

```json
{
  "version": "1.0",
  "language": "en-US",
  "generatedAt": "2024-01-01T12:00:00Z",
  "slides": [
    {
      "index": 0,
      "audioFile": "slide_000.mp3",
      "audioDurationMs": 5200,
      "pauseDurationMs": 500,
      "totalDurationMs": 5700
    }
  ]
}
```

This manifest is used by `vac video --manifest` for precise slide timing.

## How It Works

### Pipeline Overview

vac supports two workflows:

**Workflow A: Marp Slides - Full Pipeline (inline voiceovers)**
```
presentation.md → Parse → TTS → Render → Record → Combine → video.mp4
```

**Workflow B: Marp Slides - Two-Step (JSON transcript)**
```
Step 1: transcript.json → vac slides tts → audio/{lang}/*.mp3 + manifest.json
Step 2: presentation.md + manifest.json → vac slides video → video/{lang}.mp4
```

**Workflow C: Browser Demo with Voiceover**
```
config.yaml → vac browser video → TTS + Record + Combine → demo.mp4
```

### Detailed Pipeline

```
┌─────────────────────────────────────────────────────────────────────────┐
│  INPUT OPTIONS                                                          │
│  ┌─────────────────────────┐    ┌─────────────────────────────────────┐ │
│  │ A: presentation.md      │ OR │ B: transcript.json (multi-language) │ │
│  │    (inline voiceovers)  │    │    + presentation.md                │ │
│  └─────────────────────────┘    └─────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 1: Parse / Load Transcript                                        │
│  • A: Extract voiceover from HTML comments + parse [PAUSE:ms]           │
│  • B: Load transcript.json, select language, resolve voice config       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 2: Generate Audio (OmniVoice TTS)                                 │
│  • Send voiceover text to TTS provider (ElevenLabs)                     │
│  • Apply voice settings (stability, similarity, style)                  │
│  • Output: audio/{lang}/slide_000.mp3, slide_001.mp3, ...               │
│  • Output: audio/{lang}/manifest.json (timing for video recording)      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 3: Render HTML (Marp CLI)                                         │
│  • Execute: marp presentation.md -o presentation.html --html            │
│  • Creates navigable HTML presentation with all slides                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 4: Record Slides (Browser + ffmpeg)                               │
│  • Launch headless browser via Rod (Chromium)                           │
│  • Load HTML presentation                                               │
│  • For each slide:                                                      │
│    ├─ Navigate to slide                                                 │
│    ├─ Record for: audioDurationMs + pauseDurationMs (from manifest)     │
│    └─ Save: video/slide_000.mp4, slide_001.mp4, ...                     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 5: Combine Videos (ffmpeg)                                        │
│  • Concatenate all slide videos in sequence                             │
│  • Optional: Apply crossfade transitions (--transition flag)            │
│  • Output: video.mp4                                                    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 6: Export Individual Videos (Optional)                            │
│  • Copy individual slide videos to output directory                     │
│  • For Udemy courses: --output-individual ./lectures/                   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Step Details

| Step | Component | Tool | Input | Output |
|------|-----------|------|-------|--------|
| 1 | Parser | Go | `slides.md` | Slides + voiceovers |
| 2 | TTS | OmniVoice (ElevenLabs) | Voiceover text | `slide_*.mp3` |
| 3 | Renderer | Marp CLI | `slides.md` | `presentation.html` |
| 4 | Recorder | Rod + ffmpeg | HTML + MP3 | `slide_*.mp4` |
| 5 | Combiner | ffmpeg | `slide_*.mp4` | `output.mp4` |
| 6 | Exporter | Go | `slide_*.mp4` | Individual files |

## Architecture

```
vac/
├── cmd/vac/          # CLI (Cobra-based)
│   ├── main.go              # Entry point
│   ├── root.go              # Root command
│   ├── tts.go               # TTS subcommand
│   └── video.go             # Video subcommand
├── pkg/
│   ├── parser/              # Marp markdown parser
│   ├── transcript/          # JSON transcript types
│   ├── tts/                 # TTS generation + manifest
│   ├── omnivoice/           # OmniVoice TTS/STT provider wrappers
│   ├── renderer/            # Marp HTML renderer & browser control
│   ├── audio/               # Audio utilities
│   ├── video/               # Video recording & combination
│   └── orchestrator/        # Main workflow coordinator
├── examples/                # Example presentations
│   └── intro/               # Self-documenting example
│       ├── presentation.md
│       ├── transcript.json
│       └── README.md
└── docs/                    # MkDocs documentation
```

## Platform-Specific Recording

### macOS (including Apple Silicon M1/M2/M3)

Fully compatible with Apple Silicon Macs. Uses `avfoundation` for screen capture:
```bash
ffmpeg -f avfoundation -i "<device>:none" ...
```

#### Required Permissions

**Screen Recording permission is required.** Before running vac, grant permission to your terminal app:

1. Open **System Settings** (or System Preferences on older macOS)
2. Navigate to **Privacy & Security > Screen Recording**
3. Enable your terminal application (Terminal, iTerm2, VS Code, etc.)
4. Restart the terminal after granting permission

Without this permission, ffmpeg will fail with "Could not find video device" or similar errors.

#### Screen Device Auto-Detection

The tool automatically detects the correct screen capture device. On Macs with external displays or connected iPhones, the device number varies. To list available devices:
```bash
ffmpeg -f avfoundation -list_devices true -i ""
```

You can manually specify the device if needed:
```bash
vac video --input slides.md --output video.mp4 --screen-device "4:none"
```

### Linux
Uses `x11grab` for screen capture:
```bash
ffmpeg -f x11grab -i ":0.0" ...
```

### Windows
Uses `gdigrab` for screen capture:
```bash
ffmpeg -f gdigrab -i "desktop" ...
```

## Output Format & Platform Compatibility

Videos are encoded with settings optimized for direct upload to YouTube and Udemy - no re-encoding required.

### Video Specifications

| Setting | Value | Notes |
|---------|-------|-------|
| **Container** | MP4 | Universal compatibility |
| **Video Codec** | H.264 (libx264) | Required by YouTube & Udemy |
| **Resolution** | 1920x1080 | Full HD (configurable) |
| **Frame Rate** | 30fps | Standard (configurable) |
| **Quality** | CRF 23 | Good quality/size balance |
| **Pixel Format** | yuv420p | Maximum compatibility |
| **Audio Codec** | AAC | Required by both platforms |
| **Audio Bitrate** | 192kbps | Clear speech audio |

### YouTube Upload

The combined video (`--output`) is ready for direct upload:
- Includes optional crossfade transitions (`--transition 0.5`)
- Single file containing all slides with narration
- No processing or re-encoding needed

### Udemy Upload

Individual slide videos (`--output-individual`) are designed for Udemy courses:
- Each slide saved as separate file (slide_000.mp4, slide_001.mp4, etc.)
- Upload as individual lectures in your course curriculum
- Sequential naming for easy organization

**Tip for Udemy**: Udemy recommends lectures be 2+ minutes. For short slides, consider:
- Adding longer pause directives (`[PAUSE:5000]`)
- Combining related slides into single lectures
- Using more detailed voiceover scripts

## Examples

The `examples/` directory contains self-contained examples:

```
examples/
├── intro/                    # Introduction to vac
│   ├── presentation.md       # Marp markdown source (13 slides)
│   ├── transcript.json       # Multi-language transcript (en-US, en-GB, es-ES)
│   ├── README.md             # Detailed usage instructions
│   └── audio/                # Generated audio (after running tts)
│       ├── en-US/
│       │   ├── manifest.json
│       │   └── slide_*.mp3
│       └── es-ES/
│           ├── manifest.json
│           └── slide_*.mp3
└── README.md
```

### Running the Intro Example

**Option A: Full pipeline (inline voiceovers)**

```bash
vac slides video \
  --input examples/intro/presentation.md \
  --output examples/intro/output.mp4
```

**Option B: Two-step with transcript (multi-language)**

```bash
# Generate audio for English
vac slides tts \
  --transcript examples/intro/transcript.json \
  --output examples/intro/audio/en-US/ \
  --lang en-US

# Generate video
vac slides video \
  --input examples/intro/presentation.md \
  --manifest examples/intro/audio/en-US/manifest.json \
  --output examples/intro/video/en-US.mp4

# Generate Spanish version
vac slides tts \
  --transcript examples/intro/transcript.json \
  --output examples/intro/audio/es-ES/ \
  --lang es-ES

vac slides video \
  --input examples/intro/presentation.md \
  --manifest examples/intro/audio/es-ES/manifest.json \
  --output examples/intro/video/es-ES.mp4
```

The `intro` example is a self-documenting presentation that explains what vac does - using vac itself.

### Additional Example

See `example_presentation.md` for a complete example with:

- Custom Marp theme
- Voiceover comments on each slide
- Pause directives for timing

## Troubleshooting

### "ffmpeg not found"
Install ffmpeg using your package manager (see Prerequisites)

### "marp CLI not found"
Install Marp CLI: `npm install -g @marp-team/marp-cli`

### "ElevenLabs API error"
- Verify your API key is correct
- Check your ElevenLabs account has sufficient credits
- Ensure you have access to the voice ID you specified

### Subtitle burning fails (--subtitles-burn)

The `--subtitles-burn` flag requires FFmpeg compiled with libass support. If you see an error like "FFmpeg subtitles filter not available", your FFmpeg installation needs to be updated.

**Check if your FFmpeg has subtitle support:**
```bash
ffmpeg -filters 2>&1 | grep subtitles
```

If nothing is returned, install FFmpeg with libass:

```bash
# macOS: Use homebrew-ffmpeg tap (includes libass by default)
brew uninstall ffmpeg
brew tap homebrew-ffmpeg/ffmpeg
brew install homebrew-ffmpeg/ffmpeg/ffmpeg

# Linux (Ubuntu/Debian)
sudo apt install ffmpeg libass-dev

# Verify installation
ffmpeg -filters 2>&1 | grep subtitles
# Should show: subtitles V->V Render text subtitles...
```

**Alternative:** Use `--subtitles` without `--subtitles-burn` to generate a separate `.srt` file that video players can load.

### Recording issues on macOS

**"recording failed: exit status 1" or ffmpeg errors:**

1. **Grant Screen Recording permission** (most common issue):
   - Go to **System Settings > Privacy & Security > Screen Recording**
   - Enable your terminal app (Terminal, iTerm2, VS Code, etc.)
   - **Restart your terminal** after granting permission

2. **Verify ffmpeg can access the screen:**
   ```bash
   ffmpeg -f avfoundation -list_devices true -i ""
   ```
   You should see "Capture screen 0" or similar in the output.

3. **Use verbose mode to see ffmpeg errors:**
   ```bash
   vac video --input slides.md --output video.mp4 --verbose
   ```

4. **Other tips:**
   - Ensure the browser window is visible during recording
   - Try reducing video resolution if performance is poor
   - Manually specify screen device with `--screen-device "1:none"`

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/vac ./cmd/vac
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [Marp](https://marp.app/) - Markdown presentation ecosystem
- [OmniVoice](https://github.com/plexusone/omnivoice) - Unified TTS/STT provider interface
- [ElevenLabs](https://elevenlabs.io/) - AI voice generation (TTS)
- [Deepgram](https://deepgram.com/) - Speech-to-text (STT) for subtitles
- [Rod](https://github.com/go-rod/rod) - Browser automation framework
- [ffmpeg](https://ffmpeg.org/) - Multimedia processing

## Related Projects

- **[Marptalk](https://github.com/imjasonh/marptalk)** - Node.js-based Marp-to-video tool using Google Cloud TTS. Features browser-based TTS fallback for quick iteration without API costs, YouTube chapter markers generation, and LLM-assisted presentation drafting.

## Roadmap

- [x] Custom voice settings (stability, similarity, style)
- [x] Video transitions between slides
- [x] Individual slide video export (for Udemy)
- [x] JSON transcript for multi-language support
- [x] Decoupled TTS workflow (separate audio generation)
- [x] Audio manifest with timing information
- [x] Progress bar during conversion
- [x] Add subtitle/caption generation
- [x] Browser demo recording with voiceover
- [x] Multi-language video generation
- [x] TTS audio caching for faster iterations
- [ ] Support for background music
- [ ] Batch processing of multiple presentations
- [ ] Web UI for easier configuration
- [ ] Export to different video formats
- [ ] Avatar integration (HeyGen, Synthesia)
