---
marp: true
theme: default
paginate: true
---

# Local Avatar Demo

**Fully offline video generation**

<!-- Welcome to the local avatar demo. This presentation shows how to generate
talking-head videos entirely on your local machine, without any cloud APIs. -->

---

# The Offline Pipeline

1. **F5-TTS** (MLX) — Text-to-speech on Apple Silicon
2. **LivePortrait + JoyVASA** — Audio-driven avatar rendering
3. **Whisper** (MLX) — Speech-to-text for subtitles
4. **FFmpeg** — Video composition and encoding

<!-- The offline pipeline uses four components, all running locally on Apple Silicon.
F5-TTS generates the narration audio. LivePortrait with JoyVASA creates the
talking-head video. Whisper generates accurate subtitles. And FFmpeg composites
everything together. -->

---

# No Cloud Required

- No API keys needed
- No network requests during generation
- Full control over your content
- Works offline (after initial model download)

<!-- The key benefit is complete independence from cloud services. Once you've
downloaded the model weights, everything runs locally. Your content never
leaves your machine. -->

---

# Getting Started

```bash
# Start the local TTS server
cd omnivoice-core/providers/f5tts-mlx/server
./run.sh

# Start the local avatar server
cd omniavatar-core/providers/liveportrait-joyvasa/server
./run.sh
```

<!-- To get started, you need to run two Python servers. The F5-TTS server handles
text-to-speech synthesis. The LivePortrait JoyVASA server handles avatar rendering.
Both connect via Unix sockets. -->

---

# Generate Your Video

```bash
vac slides video \
  --input presentation.md \
  --tts-provider f5tts-mlx \
  --avatar-provider liveportrait-joyvasa \
  --avatar-id example \
  --output video.mp4
```

<!-- With both servers running, use the vac command to generate your video.
Specify the local TTS and avatar providers, point to your avatar bundle,
and vac handles the rest. -->

---

# Thank You

**videoascode** + **omnivoice** + **omniavatar**

*Fully local AI video generation*

<!-- Thank you for watching this demo. The combination of videoascode, omnivoice,
and omniavatar enables fully local AI video generation on Apple Silicon. -->
