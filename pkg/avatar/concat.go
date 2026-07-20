// Package avatar generates talking-head presenter videos from narration
// audio using OmniAvatar render providers, and prepares narration audio
// for avatar generation.
//
// This feature is optional: presentations render exactly as before unless
// the avatar commands are used.
package avatar

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grokify/videoascode/pkg/tts"
)

// audioSegment is one input clip plus trailing silence padding in
// milliseconds, applied before concatenation.
type audioSegment struct {
	path  string
	padMs int
}

// ConcatManifestAudio concatenates per-slide narration audio from a TTS
// manifest into a single MP3 file, inserting the manifest's pause gaps as
// silence so the combined narration matches the final video timeline.
//
// The output is normalized to 44.1kHz mono MP3, which is compatible with
// all avatar provider upload APIs (HeyGen documents MP3 as its supported
// audio asset type).
func ConcatManifestAudio(ctx context.Context, manifestPath, outputPath string) error {
	manifest, err := tts.LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	if len(manifest.Slides) == 0 {
		return fmt.Errorf("manifest %s contains no slides", manifestPath)
	}

	segments := manifestSegments(manifest, filepath.Dir(manifestPath))
	return runFFmpeg(ctx, buildConcatArgs(segments, outputPath))
}

// ConcatAudioFiles concatenates already-prepared audio files (e.g. the
// per-slide padded audio produced by the video pipeline) into a single
// normalized MP3. No additional padding is applied — the inputs are
// assumed to already carry any pause timing.
func ConcatAudioFiles(ctx context.Context, inputs []string, outputPath string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no audio files to concatenate")
	}

	segments := make([]audioSegment, len(inputs))
	for i, p := range inputs {
		segments[i] = audioSegment{path: p}
	}
	return runFFmpeg(ctx, buildConcatArgs(segments, outputPath))
}

// manifestSegments converts manifest slides into audio segments, resolving
// relative audio paths against baseDir and deriving each slide's trailing
// pause as silence padding.
func manifestSegments(manifest *tts.Manifest, baseDir string) []audioSegment {
	segments := make([]audioSegment, 0, len(manifest.Slides))
	for _, slide := range manifest.Slides {
		audioPath := slide.AudioFile
		if !filepath.IsAbs(audioPath) {
			audioPath = filepath.Join(baseDir, audioPath)
		}

		padMs := slide.TotalDuration - slide.AudioDuration
		if padMs < 0 {
			padMs = 0
		}

		segments = append(segments, audioSegment{path: audioPath, padMs: padMs})
	}
	return segments
}

// buildConcatArgs builds the ffmpeg arguments that normalize each segment
// to mono 44.1kHz, apply trailing silence padding, and concatenate them
// into a single MP3.
func buildConcatArgs(segments []audioSegment, outputPath string) []string {
	args := []string{"-y"}

	var filters []string
	var labels []string
	for i, seg := range segments {
		args = append(args, "-i", seg.path)

		label := fmt.Sprintf("a%d", i)
		filters = append(filters, fmt.Sprintf(
			"[%d:a]aformat=sample_fmts=fltp:sample_rates=44100:channel_layouts=mono,apad=pad_dur=%.3f[%s]",
			i, float64(seg.padMs)/1000.0, label))
		labels = append(labels, "["+label+"]")
	}

	filters = append(filters, fmt.Sprintf("%sconcat=n=%d:v=0:a=1[out]",
		strings.Join(labels, ""), len(segments)))

	args = append(args,
		"-filter_complex", strings.Join(filters, ";"),
		"-map", "[out]",
		"-c:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	return args
}

// runFFmpeg executes ffmpeg with the given arguments, surfacing combined
// output on failure.
func runFFmpeg(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}
