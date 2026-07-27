package segment

import (
	"testing"

	"github.com/grokify/videoascode/pkg/browser"
	"github.com/grokify/videoascode/pkg/transcript"
)

func TestSlideSegment_GetID(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "segment_000"},
		{1, "segment_001"},
		{10, "segment_010"},
		{100, "segment_100"},
	}

	for _, tc := range tests {
		seg := NewSlideSegment(tc.index, "Test", nil)
		if got := seg.GetID(); got != tc.expected {
			t.Errorf("GetID() for index %d = %s, want %s", tc.index, got, tc.expected)
		}
	}
}

func TestSlideSegment_BasicGetters(t *testing.T) {
	seg := NewSlideSegment(5, "Test Title", nil)

	if got := seg.GetIndex(); got != 5 {
		t.Errorf("GetIndex() = %d, want 5", got)
	}
	if got := seg.GetSourceType(); got != SourceTypeSlide {
		t.Errorf("GetSourceType() = %s, want %s", got, SourceTypeSlide)
	}
	if got := seg.GetTitle(); got != "Test Title" {
		t.Errorf("GetTitle() = %s, want 'Test Title'", got)
	}
}

func TestSlideSegment_GetVoiceovers(t *testing.T) {
	transcripts := map[string]transcript.LanguageContent{
		"en-US": {
			Segments: []transcript.Segment{
				{Text: "Hello"},
				{Text: "World", Pause: 500},
			},
		},
	}
	seg := NewSlideSegment(0, "Test", transcripts)

	voiceovers := seg.GetVoiceovers("en-US")
	if len(voiceovers) != 1 {
		t.Fatalf("GetVoiceovers() returned %d items, want 1", len(voiceovers))
	}
	if voiceovers[0].Text != "Hello World" {
		t.Errorf("Voiceover text = %s, want 'Hello World'", voiceovers[0].Text)
	}
	if voiceovers[0].Language != "en-US" {
		t.Errorf("Voiceover language = %s, want 'en-US'", voiceovers[0].Language)
	}
	if voiceovers[0].Pause != 500 {
		t.Errorf("Voiceover pause = %d, want 500", voiceovers[0].Pause)
	}

	// Non-existent language
	if got := seg.GetVoiceovers("fr-FR"); got != nil {
		t.Errorf("GetVoiceovers(fr-FR) = %v, want nil", got)
	}
}

func TestSlideSegment_GetLanguages(t *testing.T) {
	transcripts := map[string]transcript.LanguageContent{
		"en-US": {},
		"es-ES": {},
	}
	seg := NewSlideSegment(0, "Test", transcripts)

	languages := seg.GetLanguages()
	if len(languages) != 2 {
		t.Errorf("GetLanguages() returned %d items, want 2", len(languages))
	}
}

func TestSlideSegment_ImagePath(t *testing.T) {
	seg := NewSlideSegment(0, "Test", nil)

	if got := seg.GetImagePath(); got != "" {
		t.Errorf("GetImagePath() initial = %s, want empty", got)
	}

	seg.SetImagePath("/path/to/image.png")
	if got := seg.GetImagePath(); got != "/path/to/image.png" {
		t.Errorf("GetImagePath() = %s, want '/path/to/image.png'", got)
	}
}

func TestSlideSegment_GetFullText(t *testing.T) {
	transcripts := map[string]transcript.LanguageContent{
		"en-US": {
			Segments: []transcript.Segment{
				{Text: "Hello"},
				{Text: "World"},
			},
		},
	}
	seg := NewSlideSegment(0, "Test", transcripts)

	if got := seg.GetFullText("en-US"); got != "Hello World" {
		t.Errorf("GetFullText(en-US) = %s, want 'Hello World'", got)
	}
	if got := seg.GetFullText("fr-FR"); got != "" {
		t.Errorf("GetFullText(fr-FR) = %s, want ''", got)
	}
}

func TestBrowserSegment_GetID(t *testing.T) {
	seg := NewBrowserSegment(42, "Test", "https://example.com", nil)
	if got := seg.GetID(); got != "segment_042" {
		t.Errorf("GetID() = %s, want 'segment_042'", got)
	}
}

func TestBrowserSegment_BasicGetters(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick, Selector: "#btn"},
	}
	seg := NewBrowserSegment(3, "Demo", "https://example.com", steps)

	if got := seg.GetIndex(); got != 3 {
		t.Errorf("GetIndex() = %d, want 3", got)
	}
	if got := seg.GetSourceType(); got != SourceTypeBrowser {
		t.Errorf("GetSourceType() = %s, want %s", got, SourceTypeBrowser)
	}
	if got := seg.GetTitle(); got != "Demo" {
		t.Errorf("GetTitle() = %s, want 'Demo'", got)
	}
	if got := seg.GetURL(); got != "https://example.com" {
		t.Errorf("GetURL() = %s, want 'https://example.com'", got)
	}
	if len(seg.GetSteps()) != 1 {
		t.Errorf("GetSteps() returned %d items, want 1", len(seg.GetSteps()))
	}
}

func TestBrowserSegment_GetVoiceovers_FromSteps(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick, Voiceover: "Click the button"},
		{Action: browser.ActionWait},
		{Action: browser.ActionInput, Voiceover: "Enter your name"},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)

	voiceovers := seg.GetVoiceovers("en-US")
	if len(voiceovers) != 2 {
		t.Fatalf("GetVoiceovers() returned %d items, want 2", len(voiceovers))
	}
	if voiceovers[0].Text != "Click the button" {
		t.Errorf("Voiceover[0].Text = %s, want 'Click the button'", voiceovers[0].Text)
	}
	if voiceovers[0].StepIndex != 0 {
		t.Errorf("Voiceover[0].StepIndex = %d, want 0", voiceovers[0].StepIndex)
	}
	if voiceovers[1].Text != "Enter your name" {
		t.Errorf("Voiceover[1].Text = %s, want 'Enter your name'", voiceovers[1].Text)
	}
	if voiceovers[1].StepIndex != 2 {
		t.Errorf("Voiceover[1].StepIndex = %d, want 2", voiceovers[1].StepIndex)
	}
}

func TestBrowserSegment_GetVoiceovers_FromTranscripts(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick, Voiceover: "This should be overridden"},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)

	seg.SetTranscripts(map[string]transcript.LanguageContent{
		"en-US": {
			Segments: []transcript.Segment{
				{Text: "From transcript"},
			},
		},
	})

	voiceovers := seg.GetVoiceovers("en-US")
	if len(voiceovers) != 1 {
		t.Fatalf("GetVoiceovers() returned %d items, want 1", len(voiceovers))
	}
	if voiceovers[0].Text != "From transcript" {
		t.Errorf("Voiceover[0].Text = %s, want 'From transcript'", voiceovers[0].Text)
	}
}

func TestBrowserSegment_VideoPath(t *testing.T) {
	seg := NewBrowserSegment(0, "Test", "https://example.com", nil)

	if got := seg.GetVideoPath(); got != "" {
		t.Errorf("GetVideoPath() initial = %s, want empty", got)
	}

	seg.SetVideoPath("/path/to/video.mp4")
	if got := seg.GetVideoPath(); got != "/path/to/video.mp4" {
		t.Errorf("GetVideoPath() = %s, want '/path/to/video.mp4'", got)
	}
}

func TestBrowserSegment_GetStepVoiceovers(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick, Voiceover: "First"},
		{Action: browser.ActionWait},
		{Action: browser.ActionInput, Voiceover: "Second"},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)

	voiceovers := seg.GetStepVoiceovers()
	expected := []string{"First", "Second"}
	if len(voiceovers) != len(expected) {
		t.Fatalf("GetStepVoiceovers() returned %d items, want %d", len(voiceovers), len(expected))
	}
	for i, v := range voiceovers {
		if v != expected[i] {
			t.Errorf("GetStepVoiceovers()[%d] = %s, want %s", i, v, expected[i])
		}
	}
}

func TestBrowserSegment_LimitSteps(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick},
		{Action: browser.ActionWait},
		{Action: browser.ActionInput},
		{Action: browser.ActionNavigate},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)

	seg.LimitSteps(2)
	if len(seg.GetSteps()) != 2 {
		t.Errorf("LimitSteps(2) resulted in %d steps, want 2", len(seg.GetSteps()))
	}

	// Limiting to more than available should not change
	seg.LimitSteps(10)
	if len(seg.GetSteps()) != 2 {
		t.Errorf("LimitSteps(10) changed step count to %d", len(seg.GetSteps()))
	}
}

// TestBrowserSegment_LimitSteps_TruncatesTranscripts guards against a
// regression where LimitSteps truncated b.steps but GetVoiceovers, which
// prefers segment-level transcripts whenever present, kept returning every
// original transcript segment — so --limit-steps silently had no effect on
// TTS generation/pacing/duration for transcript-based (multi-language)
// segments.
func TestBrowserSegment_LimitSteps_TruncatesTranscripts(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionWait},
		{Action: browser.ActionKeypress},
		{Action: browser.ActionKeypress},
		{Action: browser.ActionKeypress},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)
	seg.SetTranscripts(map[string]transcript.LanguageContent{
		"en-US": {Segments: []transcript.Segment{
			{Text: "slide one"}, {Text: "slide two"}, {Text: "slide three"}, {Text: "slide four"},
		}},
	})

	seg.LimitSteps(2)

	voiceovers := seg.GetVoiceovers("en-US")
	if len(voiceovers) != 2 {
		t.Fatalf("GetVoiceovers() after LimitSteps(2) returned %d items, want 2 (got: %+v)", len(voiceovers), voiceovers)
	}
	if voiceovers[0].Text != "slide one" || voiceovers[1].Text != "slide two" {
		t.Errorf("GetVoiceovers() after LimitSteps(2) = %+v, want [slide one, slide two]", voiceovers)
	}
}

func TestBrowserSegment_UpdateStepMinDurations(t *testing.T) {
	steps := []browser.Step{
		{Action: browser.ActionClick, Voiceover: "First"},
		{Action: browser.ActionWait},
		{Action: browser.ActionInput, Voiceover: "Second"},
	}
	seg := NewBrowserSegment(0, "Test", "https://example.com", steps)

	// Update durations: voiceover 0 = 2000ms, voiceover 1 = 3000ms
	durations := map[int]int{0: 2000, 1: 3000}
	seg.UpdateStepMinDurations(durations)

	updatedSteps := seg.GetSteps()
	// Step 0 has voiceover, should get 2000 + 500 buffer
	if updatedSteps[0].MinDuration != 2500 {
		t.Errorf("Step 0 MinDuration = %d, want 2500", updatedSteps[0].MinDuration)
	}
	// Step 1 has no voiceover, should be unchanged
	if updatedSteps[1].MinDuration != 0 {
		t.Errorf("Step 1 MinDuration = %d, want 0", updatedSteps[1].MinDuration)
	}
	// Step 2 has voiceover, should get 3000 + 500 buffer
	if updatedSteps[2].MinDuration != 3500 {
		t.Errorf("Step 2 MinDuration = %d, want 3500", updatedSteps[2].MinDuration)
	}
}

func TestSourceTypeConstants(t *testing.T) {
	if SourceTypeSlide != "slide" {
		t.Errorf("SourceTypeSlide = %s, want 'slide'", SourceTypeSlide)
	}
	if SourceTypeBrowser != "browser" {
		t.Errorf("SourceTypeBrowser = %s, want 'browser'", SourceTypeBrowser)
	}
}

func TestVoiceover_Struct(t *testing.T) {
	v := Voiceover{
		Index:       0,
		Text:        "Hello",
		Language:    "en-US",
		Pause:       500,
		MinDuration: 2000,
		StepIndex:   1,
	}

	if v.Index != 0 {
		t.Errorf("Index = %d, want 0", v.Index)
	}
	if v.Text != "Hello" {
		t.Errorf("Text = %s, want 'Hello'", v.Text)
	}
	if v.Pause != 500 {
		t.Errorf("Pause = %d, want 500", v.Pause)
	}
}

func TestAudioResult_Struct(t *testing.T) {
	r := AudioResult{
		SegmentID:      "segment_001",
		AudioFiles:     map[string]string{"en-US": "/path/to/audio.wav"},
		Durations:      map[string]int{"en-US": 5000},
		MaxDuration:    5000,
		VoiceoverCount: 1,
	}

	if r.SegmentID != "segment_001" {
		t.Errorf("SegmentID = %s, want 'segment_001'", r.SegmentID)
	}
	if r.MaxDuration != 5000 {
		t.Errorf("MaxDuration = %d, want 5000", r.MaxDuration)
	}
}

func TestVideoResult_Struct(t *testing.T) {
	r := VideoResult{
		SegmentID:  "segment_002",
		VideoPath:  "/path/to/video.mp4",
		Duration:   10000,
		FrameCount: 300,
	}

	if r.VideoPath != "/path/to/video.mp4" {
		t.Errorf("VideoPath = %s, want '/path/to/video.mp4'", r.VideoPath)
	}
	if r.FrameCount != 300 {
		t.Errorf("FrameCount = %d, want 300", r.FrameCount)
	}
}

func TestTimingInfo_Struct(t *testing.T) {
	ti := TimingInfo{
		StartMs:  1000,
		EndMs:    5000,
		Duration: 4000,
		VoiceoverTimings: []VoiceoverTiming{
			{Index: 0, StartMs: 0, EndMs: 2000, Text: "First"},
			{Index: 1, StartMs: 2000, EndMs: 4000, Text: "Second"},
		},
	}

	if ti.Duration != 4000 {
		t.Errorf("Duration = %d, want 4000", ti.Duration)
	}
	if len(ti.VoiceoverTimings) != 2 {
		t.Errorf("VoiceoverTimings count = %d, want 2", len(ti.VoiceoverTimings))
	}
}
