package config

import (
	"strings"

	"github.com/grokify/videoascode/pkg/browser"
	"github.com/grokify/videoascode/pkg/transcript"
)

// SourceType identifies the segment content source
type SourceType string

const (
	// SourceTypeSlide indicates a Marp slide segment
	SourceTypeSlide SourceType = "slide"
	// SourceTypeBrowser indicates a browser-driven demo segment
	SourceTypeBrowser SourceType = "browser"
)

// VideoConfig is the top-level configuration for video generation
type VideoConfig struct {
	// Version is the config schema version
	Version string `json:"version" yaml:"version"`

	// Title is the video title
	Title string `json:"title" yaml:"title"`

	// Description is an optional video description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// DefaultLanguage is the BCP-47 language code (e.g., "en-US")
	DefaultLanguage string `json:"defaultLanguage" yaml:"defaultLanguage"`

	// DefaultVoice is the default TTS voice configuration
	DefaultVoice transcript.VoiceConfig `json:"defaultVoice" yaml:"defaultVoice"`

	// Resolution is the output video resolution
	Resolution Resolution `json:"resolution" yaml:"resolution"`

	// FrameRate is the output video frame rate
	FrameRate int `json:"frameRate" yaml:"frameRate"`

	// Segments is the ordered list of video segments
	Segments []SegmentConfig `json:"segments" yaml:"segments"`

	// OutputDir is the output directory for generated files
	OutputDir string `json:"outputDir,omitempty" yaml:"outputDir,omitempty"`

	// Languages is the list of languages to generate (uses DefaultLanguage if empty)
	Languages []string `json:"languages,omitempty" yaml:"languages,omitempty"`

	// TransitionDuration is the crossfade duration between segments (seconds)
	TransitionDuration float64 `json:"transitionDuration,omitempty" yaml:"transitionDuration,omitempty"`

	// Metadata is arbitrary key-value pairs for custom use
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Pronunciations maps terms to how they should be spoken by TTS, keyed by
	// BCP-47 language (e.g. {"AAuth": {"en-US": "ay auth"}}). Applied to text
	// immediately before synthesis, for any provider; the original spelling
	// is left untouched everywhere else (timing-based subtitles, manifests),
	// so displayed text stays correct even though the audio says it
	// phonetically. Matching is case-insensitive with word boundaries, and
	// multi-word terms are matched before shorter overlapping ones.
	Pronunciations map[string]map[string]string `json:"pronunciations,omitempty" yaml:"pronunciations,omitempty"`
}

// Resolution defines video dimensions
type Resolution struct {
	Width  int `json:"width" yaml:"width"`
	Height int `json:"height" yaml:"height"`
}

// Common resolutions
var (
	Resolution720p  = Resolution{Width: 1280, Height: 720}
	Resolution1080p = Resolution{Width: 1920, Height: 1080}
	Resolution4K    = Resolution{Width: 3840, Height: 2160}
)

// SegmentConfig defines a segment in the video
type SegmentConfig struct {
	// Type is the segment type (slide or browser)
	Type SourceType `json:"type" yaml:"type"`

	// Name is an optional identifier for this segment
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Slide-specific fields

	// Source is the Marp markdown file path (for slide segments)
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// Slides is the list of slide indices to include (for slide segments)
	// If empty, all slides from the source are included
	Slides []int `json:"slides,omitempty" yaml:"slides,omitempty"`

	// Browser-specific fields

	// URL is the starting URL (for browser segments)
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Steps is the list of browser actions (for browser segments)
	Steps []browser.Step `json:"steps,omitempty" yaml:"steps,omitempty"`

	// Headless runs the browser without UI (for browser segments)
	Headless bool `json:"headless,omitempty" yaml:"headless,omitempty"`

	// Common fields

	// Transcripts contains per-language voiceover content
	Transcripts map[string]SegmentTranscript `json:"transcripts,omitempty" yaml:"transcripts,omitempty"`

	// Voice overrides the default voice for this segment
	Voice *transcript.VoiceConfig `json:"voice,omitempty" yaml:"voice,omitempty"`

	// MinDuration is the minimum segment duration in milliseconds
	MinDuration int `json:"minDuration,omitempty" yaml:"minDuration,omitempty"`
}

// SegmentTranscript contains transcript content for a specific language
type SegmentTranscript struct {
	// Text is the voiceover text for the entire segment
	Text string `json:"text,omitempty" yaml:"text,omitempty"`

	// Segments is the list of text segments (for finer control)
	Segments []transcript.Segment `json:"segments,omitempty" yaml:"segments,omitempty"`

	// Voice overrides the voice for this language
	Voice *transcript.VoiceConfig `json:"voice,omitempty" yaml:"voice,omitempty"`
}

// DefaultVideoConfig returns a default configuration
func DefaultVideoConfig() VideoConfig {
	return VideoConfig{
		Version:         "1.0",
		DefaultLanguage: "en-US",
		Resolution:      Resolution1080p,
		FrameRate:       30,
		Segments:        make([]SegmentConfig, 0),
	}
}

// GetLanguages returns the list of languages to generate
func (c *VideoConfig) GetLanguages() []string {
	if len(c.Languages) > 0 {
		return c.Languages
	}
	return []string{c.DefaultLanguage}
}

// GetOutputDir returns the output directory, with a default if not set
func (c *VideoConfig) GetOutputDir() string {
	if c.OutputDir != "" {
		return c.OutputDir
	}
	return "output"
}

// IsSlideSegment returns true if the segment is a slide segment
func (s *SegmentConfig) IsSlideSegment() bool {
	return s.Type == SourceTypeSlide
}

// IsBrowserSegment returns true if the segment is a browser segment
func (s *SegmentConfig) IsBrowserSegment() bool {
	return s.Type == SourceTypeBrowser
}

// GetVoiceover returns the voiceover text for a language
func (s *SegmentConfig) GetVoiceover(language string) string {
	if t, ok := s.Transcripts[language]; ok {
		if t.Text != "" {
			return t.Text
		}
		// Combine segments into single text
		texts := make([]string, len(t.Segments))
		for i, seg := range t.Segments {
			texts[i] = seg.Text
		}
		return strings.Join(texts, " ")
	}
	return ""
}

// GetStepVoiceovers extracts voiceover text from browser steps
func (s *SegmentConfig) GetStepVoiceovers() []string {
	voiceovers := make([]string, 0, len(s.Steps))
	for _, step := range s.Steps {
		if step.Voiceover != "" {
			voiceovers = append(voiceovers, step.Voiceover)
		}
	}
	return voiceovers
}
