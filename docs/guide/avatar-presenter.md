# Avatar Presenter Overlay

Add an optional talking-head presenter to your presentation videos — a
circular overlay in the corner whose lip-sync is driven by the **exact
same narration audio** used in the final video.

vac uses [OmniAvatar](https://github.com/plexusone/omniavatar) render
providers to generate the presenter video:

| Provider | Avatar Identity | Local Audio Upload |
|----------|-----------------|--------------------|
| HeyGen | `avatar_id` | Yes (MP3) |
| Tavus | `replica_id` | No — requires `--audio-url` |
| bitHuman | `agent_id` | Yes |

This feature is **optional**: presentations render exactly as before
unless the `vac avatar` commands are used.

## How It Works

```
vac slides tts
      │
      ├── per-slide audio + manifest.json
      ▼
vac avatar generate          # concat narration (incl. pause gaps)
      │                      # -> upload -> provider job -> download
      └── presenter.mp4
      ▼
vac slides video
      │
      └── slides.mp4
      ▼
vac avatar compose           # local FFmpeg circle mask + overlay
      │
      └── final.mp4
```

Two design principles:

1. **One authoritative audio track.** The avatar's mouth movements are
   generated from the concatenated narration (including per-slide pause
   gaps), and `vac avatar compose --audio` maps that same file into the
   final video. The provider-returned audio is always discarded, so
   there is no drift between narration, slides, and lips.
2. **Provider-neutral composition.** The circle mask, positioning, and
   border are applied locally with FFmpeg, so switching avatar providers
   changes nothing visually except the presenter.

## Finding an Avatar ID

`--avatar-id` must be a **provider-native** avatar ID. For HeyGen this is
a v2 avatar ID (e.g. `Abigail_expressive_2024112501`) — note that a
provider's dashboard or group-listing endpoints may show different IDs
that the generation API rejects. List usable IDs with:

```bash
export HEYGEN_API_KEY=...
vac avatar list-avatars --provider heygen --search abigail
```

Head-and-shoulders / "upper body" avatars work best, since everything
outside the circle crop is discarded.

## One-Shot Mode

The quickest path: add `--avatar-id` (and any `--avatar-*` options) to
`vac slides video`. The avatar overlay then runs as an integrated final
stage — narration is concatenated from the slide audio, the presenter is
generated, and the circle is composited automatically.

```bash
export HEYGEN_API_KEY=...
vac slides video --input slides.md --manifest audio/en-US/manifest.json \
  --output final.mp4 --avatar-id <avatar-id> --avatar-border 6
```

Because one-shot mode generates narration locally and uploads it, the
provider must support audio upload (`heygen` or `bithuman`). For Tavus,
use the decoupled workflow below with `vac avatar generate --audio-url`.

## Decoupled Workflow

The four commands run explicitly — useful for caching intermediates,
using Tavus (hosted audio URL), or inspecting the presenter before
compositing:

```bash
# 1. Generate narration audio + manifest
vac slides tts --transcript transcript.json --output audio/

# 2. Generate the presenter video (cached by audio + avatar config)
export HEYGEN_API_KEY=...
vac avatar generate --manifest audio/en-US/manifest.json \
  --provider heygen --avatar-id <avatar-id> --output presenter.mp4

# 3. Render the slides video
vac slides video --input slides.md \
  --manifest audio/en-US/manifest.json --output slides.mp4

# 4. Composite the presenter circle onto the slides
vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
  --output final.mp4 --border 6
```

## Layout Options

```bash
vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
  --output final.mp4 \
  --diameter 280 \            # circle size in pixels
  --position bottom-left \    # bottom-right, bottom-left, top-right, top-left
  --margin-x 48 --margin-y 48 \
  --border 6 --border-color 0x336699
```

For a typical 1920×1080 output, a 280–360px diameter with 48–64px
margins works well. Avoid placing important slide content in the
anchored corner, or switch `--position` per presentation.

## Caching

Presenter videos are cached by a hash of the narration audio content
plus the provider, avatar ID, and `--ext` options — the same philosophy
as TTS audio caching. Re-running `vac avatar generate` with unchanged
narration is free. Use `--no-cache` to force regeneration or
`--cache-dir` to relocate the cache.

## Provider Notes

- **HeyGen** uses the `HEYGEN_API_KEY` (the video-generation key, not
  the LiveAvatar streaming key). Its upload API documents MP3
  (`audio/mpeg`) as the supported audio type — `vac avatar generate`
  always produces MP3 narration, so this is handled automatically.
  Use `--ext test=true` for watermarked test videos that consume no
  credits.
- **Tavus** has no audio upload API; host the narration yourself and
  pass `--audio-url`.
- **bitHuman** uploads narration through its file API automatically.

## Presenter Framing

Request or create the avatar as head-and-shoulders with a centered face
and minimal side-to-side movement. Since everything outside the circle
is discarded, a plain background is sufficient — full-body avatars will
render the face too small at circle sizes.
