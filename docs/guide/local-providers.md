# Local, Offline TTS & STT (Apple Silicon)

vac can run **both** text-to-speech and speech-to-text entirely on your own
machine through [OmniVoice](https://github.com/plexusone/omnivoice-core)'s MLX
providers — no API keys, no cloud calls, no per-character costs, and nothing
leaves your device.

| Capability | Provider | Registry name | Socket |
|------------|----------|---------------|--------|
| Text-to-speech | F5-TTS MLX | `f5tts-mlx` | `unix:///tmp/omnivoice-f5tts.sock` |
| Speech-to-text | Whisper MLX | `whisper-mlx` | `unix:///tmp/omnivoice-whisper.sock` |

Each provider is a small Python/MLX gRPC server that vac talks to over a Unix
domain socket. Only the language changes — vac reaches these providers through
the same OmniVoice interfaces it uses for ElevenLabs and Deepgram, so you opt in
with a single `--local` flag.

## Requirements

- **Apple Silicon** (M1/M2/M3/M4). The MLX wheels are arm64-only.
- **Python 3.11+** capable of running as arm64 (a universal2 build works;
  Homebrew's `python` is arm64 native).
- **~4 GB free disk** for model weights: F5-TTS (~2 GB) and Whisper
  `large-v3-turbo` (~1.6 GB). Both download from Hugging Face on first use and
  are cached thereafter.

!!! note "Running under Rosetta"
    If your shell is x86_64 (Rosetta), a normally-created venv will be x86_64
    and MLX will fail to load with an *"incompatible architecture"* error. The
    `scripts/localvoice.sh` helper always launches Python under `arch -arm64`,
    so it works regardless of your shell architecture.

## Quick Start

### 1. Start the local voice servers

The `scripts/localvoice.sh` helper creates an arm64 Python environment,
installs the MLX dependencies, generates the gRPC stubs, and runs both servers.

```bash
# One-time setup (arm64 venv + MLX deps + proto stubs)
scripts/localvoice.sh setup

# Start both servers in the background
scripts/localvoice.sh start -d

# Check whether the sockets are live
scripts/localvoice.sh status

# Stop the servers
scripts/localvoice.sh stop
```

`scripts/localvoice.sh up` runs setup-if-needed and then starts in the
background. Run `scripts/localvoice.sh start` (no `-d`) to run in the
foreground, where `Ctrl-C` stops both servers.

The helper resolves the server sources from the `omnivoice-core` module via
`go list`, so there are no hard-coded paths — it works wherever the module is
checked out or replaced.

#### Script commands

| Command | Description |
|---------|-------------|
| `setup` | Create the arm64 venv, install MLX deps, generate proto stubs |
| `start` | Run both servers in the foreground (`Ctrl-C` stops both) |
| `start -d` | Run both servers in the background |
| `stop` | Stop the background servers |
| `status` | Show whether the gRPC sockets are live |
| `up` | `setup` if needed, then `start -d` |

#### Environment overrides

| Variable | Default | Purpose |
|----------|---------|---------|
| `WHISPER_MODEL` | `large-v3-turbo` | Whisper model variant (e.g. `small`, `medium`, `large-v3`) |
| `VENV_DIR` | `<repo>/.localvoice-venv` | Location of the arm64 venv |
| `PYTHON` | autodetect | arm64-capable `python3` used to build the venv |
| `F5TTS_SOCK` | `/tmp/omnivoice-f5tts.sock` | F5-TTS socket path |
| `WHISPER_SOCK` | `/tmp/omnivoice-whisper.sock` | Whisper socket path |

For example, to trade accuracy for speed and a smaller download:

```bash
WHISPER_MODEL=small scripts/localvoice.sh up
```

### 2. Generate a video with local providers

`vac slides video`'s built-in one-shot TTS is ElevenLabs-only, so local
providers are used through the **decoupled workflow**: generate the audio and
subtitles with `--local`, then render the video from the pre-generated
manifest.

```bash
# 1. Voiceover with F5-TTS (local)
vac slides tts --transcript transcript.json --output audio/en-US/ \
  --lang en-US --provider f5tts-mlx --local

# 2. Subtitles with Whisper (local)
vac subtitle --audio audio/en-US/ --lang en-US --provider whisper-mlx --local

# 3. Render the video (embeds the Whisper subtitles)
vac slides video --input presentation.md --manifest audio/en-US/manifest.json \
  --subtitles subtitles/en-US.srt --subtitles-lang en-US --output video.mp4
```

`vac stt --manifest audio/en-US/manifest.json --provider whisper-mlx --local`
is an alternative to `vac subtitle` that reads the manifest directly.

## Flags

Both TTS and STT commands expose the same pair of local flags:

| Command | Flags |
|---------|-------|
| `vac slides tts`, `vac browser video` | `--provider f5tts-mlx`, `--local`, `--f5tts-endpoint` |
| `vac subtitle`, `vac stt` | `--provider whisper-mlx`, `--local`, `--whisper-endpoint` |

`--local` enables the local providers and drops the API-key requirement. The
`--*-endpoint` flags override the default Unix socket (for example, to reach a
server on a different socket or a TCP endpoint).

## Language codes

vac uses BCP-47 locales throughout (`en-US`, `fr-FR`, `zh-Hans`). ElevenLabs and
Deepgram accept these directly. Whisper expects the ISO-639-1 primary subtag
(`en`), so the `whisper-mlx` provider **strips the region/script subtag
automatically** (`en-US` → `en`, `zh-Hans` → `zh`). You can pass either form.

## Performance notes

- The F5-TTS server pre-loads its model at startup (`--auto-load`), so the
  first synthesis is ready immediately once `status` shows it up.
- The Whisper server loads its model lazily on the first transcription; the
  initial request also triggers the one-time model download.
- Inference runs on the GPU/Neural Engine via MLX. Short slides synthesize and
  transcribe in a few seconds each on an M-series Mac.

## Troubleshooting

**`incompatible architecture (have 'arm64', need 'x86_64')`**
: Your Python/venv is x86_64. Use `scripts/localvoice.sh`, which forces
  `arch -arm64`, or rebuild the venv with an arm64 Python.

**`connection refused` / `no such file` on the socket**
: The server isn't running. Check `scripts/localvoice.sh status` and the logs in
  `<repo>/.localvoice-logs/`.

**`Unsupported language: en-us`**
: You're on an older build. Current versions normalize BCP-47 locales to the
  ISO-639-1 code Whisper expects; update, or pass `--lang en`.

**First run is slow**
: Model weights are downloading from Hugging Face (~2 GB for F5-TTS, ~1.6 GB for
  Whisper `large-v3-turbo`). Subsequent runs use the cache.

## See Also

- [Subtitle Generation](subtitles.md) — STT-based and timing-based subtitles
- [Complete Workflow](complete-workflow.md) — the full decoupled pipeline
- [OmniVoice local providers](https://github.com/plexusone/omnivoice-core) —
  the underlying MLX TTS/STT implementations
