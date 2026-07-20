# Pipeline Overview

vac uses a 6-step pipeline to convert presentations to video, plus an
optional avatar presenter overlay stage.

## Pipeline Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│  INPUT: presentation.md (Marp markdown with voiceover)                  │
│         transcript.json (optional, for multi-language)                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 1: Parse Markdown                                                 │
│  • Extract slides from Marp file                                        │
│  • Extract voiceover text from HTML comments or transcript.json         │
│  • Parse [PAUSE:ms] timing directives                                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 2: Generate Audio (TTS)                                           │
│  • Send voiceover text to ElevenLabs/OmniVoice                          │
│  • Receive MP3 audio files (one per slide)                              │
│  • Output: workdir/audio/slide_000.mp3, slide_001.mp3, ...              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 3: Render HTML (Marp CLI)                                         │
│  • Execute: marp presentation.md -o presentation.html --html            │
│  • Creates navigable HTML presentation with all slides                  │
│  • Output: workdir/html/presentation.html                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 4: Record Slides (Browser + ffmpeg)                               │
│  • Launch headless browser via Rod (Chromium)                           │
│  • Load HTML presentation                                               │
│  • For each slide:                                                      │
│    ├─ Navigate to slide (keyboard: Home + Arrow keys)                   │
│    ├─ Start screen recording with audio overlay                         │
│    ├─ Record for: audio duration + pause directives                     │
│    └─ Save: workdir/video/slide_000.mp4, slide_001.mp4, ...             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 5: Combine Videos (ffmpeg)                                        │
│  • Concatenate all slide videos in sequence                             │
│  • Optional: Apply crossfade transitions (--transition flag)            │
│  • Output: output.mp4                                                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 6: Export Individual Videos (Optional)                            │
│  • Copy individual slide videos to output directory                     │
│  • For Udemy courses: --output-individual ./lectures/                   │
│  • Output: lectures/slide_000.mp4, slide_001.mp4, ...                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  STEP 7: Avatar Presenter Overlay (Optional, --avatar-id)              │
│  • Concatenate slide narration into one track (matches the timeline)    │
│  • Generate a talking-head video via OmniAvatar (HeyGen / bitHuman)     │
│  • Composite the circle overlay onto output.mp4 with ffmpeg             │
│  • Output: output.mp4 (with presenter)                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Step Details

| Step | Package | Tool | Input | Output |
|------|---------|------|-------|--------|
| 1 | `pkg/parser` | Go | `slides.md` | Slides + voiceovers |
| 2 | `pkg/tts` | ElevenLabs API | Voiceover text | `slide_*.mp3` |
| 3 | `pkg/renderer` | Marp CLI | `slides.md` | `presentation.html` |
| 4 | `pkg/video` | Rod + ffmpeg | HTML + MP3 | `slide_*.mp4` |
| 5 | `pkg/video` | ffmpeg | `slide_*.mp4` | `output.mp4` |
| 6 | `pkg/orchestrator` | Go | `slide_*.mp4` | Individual files |
| 7 | `pkg/avatar`, `pkg/video` | OmniAvatar + ffmpeg | `output.mp4` + narration | `output.mp4` (with presenter) |

## Working Directory

During processing, vac creates a temporary working directory:

```
/tmp/vac/
├── audio/
│   ├── slide_000.mp3
│   ├── slide_001.mp3
│   └── ...
├── video/
│   ├── slide_000.mp4
│   ├── slide_001.mp4
│   └── ...
└── html/
    └── presentation.html
```

Use `--workdir` to specify a custom location.

## Timing Calculation

Each slide's recording duration is calculated as:

```
slide_duration = audio_duration + sum(pause_directives)
```

For example, if the TTS audio is 5 seconds and you have `[PAUSE:1000]`:

```
slide_duration = 5000ms + 1000ms = 6000ms (6 seconds)
```
