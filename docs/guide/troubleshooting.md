# Troubleshooting

Common issues and solutions when using vac.

## Audio Issues

### Audio Only Plays for Part of the Video

**Symptom**: Audio stops partway through the video, or audio duration doesn't match video duration.

**Cause**: This typically occurs when combining videos with audio from different TTS providers that use different sample rates (e.g., ElevenLabs at 44100 Hz, Deepgram at 22050 Hz).

**Solution**: This issue was fixed in version 0.2.1. Upgrade to the latest version:

```bash
go install github.com/grokify/videoascode/cmd/vac@latest
```

**Technical Details**: The fix uses FFmpeg's `filter_complex` concat filter instead of the concat demuxer. This properly decodes and re-encodes audio to ensure consistent sample rates across all segments.

### Mixed TTS Provider Support

vac supports using multiple TTS providers in a single video. For example, you might use:

- ElevenLabs for English (44100 Hz output)
- Deepgram for Chinese (22050 Hz output)

The video combiner automatically normalizes all audio to 44100 Hz AAC for consistent playback.

## Debug Mode

### Enabling Debug Output

To see detailed FFmpeg output during video generation:

```bash
MARP2VIDEO_DEBUG=1 vac video --input slides.md --output video.mp4
```

This streams FFmpeg's stderr/stdout to your terminal, showing encoding progress and any warnings.

### "Stdout already set" Error

**Symptom**: Error message `exec: Stdout already set` when using `MARP2VIDEO_DEBUG=1`.

**Cause**: This was a bug in earlier versions where debug mode conflicted with output capture.

**Solution**: Upgrade to the latest version where this is fixed.

## Video Generation

### Slow Video Generation

Video generation involves multiple steps that take time:

1. **TTS Generation**: Depends on API response time
2. **Slide Rendering**: Marp converts markdown to HTML
3. **Video Encoding**: FFmpeg encodes each slide
4. **Concatenation**: Final video assembly

Tips for faster generation:

- Use `--preset fast` for quicker encoding (slightly larger files)
- Pre-generate audio with `vac tts` and reuse the manifest
- Use lower resolution for drafts: `--width 1280 --height 720`

### VLC Playback Issues (Sluggish Controls)

**Symptom**: VLC becomes sluggish or unresponsive when playing generated videos.

**Cause**: This can happen when audio/video streams have mismatched durations.

**Solution**: Regenerate the video with the latest version of vac, which properly syncs all streams.

## Subtitle Issues

### Missing Words in Subtitles

**Symptom**: Some words appear in timestamps.json but not in the generated .srt/.vtt files.

**Cause**: The subtitle generator groups words into cues based on line limits. If the grouping logic doesn't account for word wrapping properly, some words may appear on a third line that gets visually cut off.

**Solution**: This was fixed in omnivoice v0.4.2. Ensure you have the latest dependencies:

```bash
go get -u github.com/plexusone/omnivoice@latest
go install github.com/grokify/videoascode/cmd/vac@latest
```

### Chinese/CJK Subtitle Spacing

**Symptom**: Chinese characters have spaces between them in subtitles.

**Cause**: Some STT providers (like Deepgram) tokenize Chinese text character-by-character, treating each character as a separate "word".

**Solution**: Post-process the subtitle files to remove spaces between CJK characters. A Python script example:

```python
import re

def fix_cjk_spacing(text):
    # Remove spaces between CJK characters
    cjk_pattern = r'([\u4e00-\u9fff])\s+([\u4e00-\u9fff])'
    while re.search(cjk_pattern, text):
        text = re.sub(cjk_pattern, r'\1\2', text)
    return text
```

## Local Provider Issues

### "incompatible architecture (have 'arm64', need 'x86_64')"

**Cause**: The Python/venv running the F5-TTS or Whisper MLX server is x86_64 (common when your shell is under Rosetta). MLX is arm64-only.

**Solution**: Use `scripts/localvoice.sh`, which always launches Python under `arch -arm64`, or rebuild the venv with an arm64 Python. See [Local Providers](local-providers.md).

### "connection refused" or missing socket

**Symptom**: `vac ... --local` fails to reach `unix:///tmp/omnivoice-f5tts.sock` or `.../omnivoice-whisper.sock`.

**Cause**: The local server isn't running.

**Solution**: Check `scripts/localvoice.sh status` and the logs in `.localvoice-logs/`. Start with `scripts/localvoice.sh start -d`.

### "Unsupported language: en-us"

**Cause**: An older `whisper-mlx` build passed the BCP-47 locale straight to Whisper, which expects the ISO-639-1 code (`en`).

**Solution**: Update to a build that normalizes locales automatically (`en-US` → `en`), or pass `--lang en`.

## Getting Help

If you encounter issues not covered here:

1. Check the [GitHub Issues](https://github.com/grokify/videoascode/issues)
2. Enable debug mode to gather detailed logs
3. Open a new issue with:
   - vac version (`vac version`)
   - FFmpeg version (`ffmpeg -version`)
   - Full error message and debug output
   - Minimal reproduction steps
