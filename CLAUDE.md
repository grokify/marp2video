# CLAUDE.md

Agent guidelines for videoascode.

## Project Overview

CLI tool for generating videos from Marp presentations with AI voiceovers,
avatar presenters, and subtitles. Supports both cloud APIs (ElevenLabs, HeyGen)
and fully local providers (F5-TTS, Whisper, LivePortrait+JoyVASA).

## Pipeline

```
presentation.md (Marp)
       │
       ▼
┌─────────────────┐
│      TTS        │  ElevenLabs or F5-TTS (local)
└────────┬────────┘
         │ audio/
         ▼
┌─────────────────┐
│    Avatar       │  HeyGen/Tavus/bitHuman or LivePortrait (local)
└────────┬────────┘
         │ presenter.mp4
         ▼
┌─────────────────┐
│   Subtitles     │  Whisper (local) or cloud STT
└────────┬────────┘
         │ subtitles.srt
         ▼
┌─────────────────┐
│ Slides + FFmpeg │  Browser recording + composition
└────────┬────────┘
         │
         ▼
    output.mp4
```

## Key Directories

| Path | Purpose |
|------|---------|
| `cmd/vac/` | CLI commands |
| `pkg/orchestrator/` | Pipeline orchestration |
| `pkg/tts/` | TTS abstraction |
| `pkg/avatar/` | Avatar generation + caching |
| `pkg/video/` | FFmpeg operations, overlays |
| `pkg/omnivoice/` | Local TTS/STT factory |
| `examples/` | Example presentations |
| `docs/` | MkDocs site |

## Local Providers

### Flags

```bash
# Local TTS
--tts-provider f5tts-mlx
--f5tts-endpoint unix:///custom/path.sock  # optional

# Local STT
--stt-provider whisper-mlx
--whisper-endpoint unix:///custom/path.sock  # optional

# Local Avatar
--avatar-provider liveportrait-joyvasa
--avatar-id <bundle-name>  # from ~/.omniavatar/avatars/
```

### Provider Maps

In `cmd/vac/avatar_generate.go`:

```go
var providerAPIKeyEnvs = map[string]string{
    "heygen":               "HEYGEN_API_KEY",
    "liveportrait-joyvasa": "",  // local, no key
}

var localProviders = map[string]bool{
    "liveportrait-joyvasa": true,
}
```

When adding a new local provider, add to both maps.

## Common Commands

```bash
# Full pipeline with local providers
vac slides video \
  --input slides.md \
  --tts-provider f5tts-mlx \
  --avatar-provider liveportrait-joyvasa \
  --avatar-id example \
  --output video.mp4

# Avatar only
vac avatar generate \
  --provider liveportrait-joyvasa \
  --avatar-id example \
  --audio narration.wav \
  --output presenter.mp4

# List avatars
vac avatar list-avatars --provider liveportrait-joyvasa
```

## Testing

```bash
go test -v ./...
golangci-lint run
```

## Dependencies

- `omniavatar` / `omniavatar-core` — avatar providers
- `omnivoice` / `omnivoice-core` — TTS/STT providers
- `ffutil` — FFmpeg wrapper
- `mogo` — utilities

## Adding a New Avatar Provider

1. Add to `providerAPIKeyEnvs` in `cmd/vac/avatar_generate.go`
2. If local (no API key), add to `localProviders` map
3. Update help text in flag definitions
4. Test with `vac avatar generate --provider <name>`

## Examples

- `examples/intro/` — Standard cloud workflow
- `examples/local-avatar/` — Fully offline pipeline

## Common Tasks

| Task | Command |
|------|---------|
| Build | `go build ./...` |
| Test | `go test -v ./...` |
| Lint | `golangci-lint run` |
| Run | `go run ./cmd/vac <command>` |
| Install | `go install ./cmd/vac` |
