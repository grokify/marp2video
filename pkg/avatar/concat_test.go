package avatar

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/grokify/videoascode/pkg/tts"
)

func TestManifestSegments(t *testing.T) {
	manifest := &tts.Manifest{
		Version: "1.0",
		Slides: []tts.SlideAudio{
			{Index: 0, AudioFile: "slide-000.mp3", AudioDuration: 4000, PauseDuration: 1500, TotalDuration: 5500},
			{Index: 1, AudioFile: "/abs/slide-001.mp3", AudioDuration: 3000, TotalDuration: 3000},
			// TotalDuration < AudioDuration must not produce a negative pad.
			{Index: 2, AudioFile: "slide-002.mp3", AudioDuration: 4000, TotalDuration: 3000},
		},
	}

	segments := manifestSegments(manifest, "/base")

	if want := filepath.Join("/base", "slide-000.mp3"); segments[0].path != want {
		t.Errorf("segment[0].path = %q, want %q", segments[0].path, want)
	}
	if segments[0].padMs != 1500 {
		t.Errorf("segment[0].padMs = %d, want 1500", segments[0].padMs)
	}
	if segments[1].path != "/abs/slide-001.mp3" {
		t.Errorf("segment[1].path = %q, want absolute path unchanged", segments[1].path)
	}
	if segments[1].padMs != 0 {
		t.Errorf("segment[1].padMs = %d, want 0", segments[1].padMs)
	}
	if segments[2].padMs != 0 {
		t.Errorf("segment[2].padMs = %d, want 0 (negative pad clamped)", segments[2].padMs)
	}
}

func TestBuildConcatArgs(t *testing.T) {
	segments := []audioSegment{
		{path: "/base/slide-000.mp3", padMs: 1500},
		{path: "/abs/slide-001.mp3", padMs: 0},
	}

	args := buildConcatArgs(segments, "/out/narration.mp3")
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-i /base/slide-000.mp3") {
		t.Errorf("args missing first input: %s", joined)
	}
	if !strings.Contains(joined, "-i /abs/slide-001.mp3") {
		t.Errorf("args missing second input: %s", joined)
	}
	if !strings.Contains(joined, "apad=pad_dur=1.500[a0]") {
		t.Errorf("args missing pause padding for segment 0: %s", joined)
	}
	if !strings.Contains(joined, "apad=pad_dur=0.000[a1]") {
		t.Errorf("args missing zero padding for segment 1: %s", joined)
	}
	if !strings.Contains(joined, "[a0][a1]concat=n=2:v=0:a=1[out]") {
		t.Errorf("args missing concat filter: %s", joined)
	}
	if args[len(args)-1] != "/out/narration.mp3" {
		t.Errorf("last arg = %q, want output path", args[len(args)-1])
	}
}

func TestBuildConcatArgsPlainFiles(t *testing.T) {
	// ConcatAudioFiles builds zero-pad segments; the concat still normalizes.
	segments := []audioSegment{{path: "a.wav"}, {path: "b.wav"}}

	args := buildConcatArgs(segments, "out.mp3")
	joined := strings.Join(args, " ")

	if strings.Count(joined, "apad=pad_dur=0.000") != 2 {
		t.Errorf("expected two zero-pad segments: %s", joined)
	}
	if !strings.Contains(joined, "concat=n=2:v=0:a=1[out]") {
		t.Errorf("args missing concat filter: %s", joined)
	}
	if !strings.Contains(joined, "-c:a libmp3lame") {
		t.Errorf("args missing mp3 encoder: %s", joined)
	}
}
