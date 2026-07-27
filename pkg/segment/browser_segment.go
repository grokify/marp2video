package segment

import (
	"fmt"

	"github.com/grokify/videoascode/pkg/browser"
	"github.com/grokify/videoascode/pkg/transcript"
)

// BrowserSegment represents a browser-driven demo
type BrowserSegment struct {
	index       int
	title       string
	url         string
	steps       []browser.Step
	transcripts map[string]transcript.LanguageContent
	videoPath   string
}

// NewBrowserSegment creates a new browser segment
func NewBrowserSegment(index int, title, url string, steps []browser.Step) *BrowserSegment {
	return &BrowserSegment{
		index:       index,
		title:       title,
		url:         url,
		steps:       steps,
		transcripts: make(map[string]transcript.LanguageContent),
	}
}

// NewBrowserSegmentFromTranscript creates a browser segment from a transcript.Slide
func NewBrowserSegmentFromTranscript(slide transcript.Slide) *BrowserSegment {
	// Convert transcript.BrowserStep to browser.Step
	steps := make([]browser.Step, len(slide.BrowserSteps))
	for i, ts := range slide.BrowserSteps {
		steps[i] = browser.Step{
			Action:         browser.StepAction(ts.Action),
			Selector:       ts.Selector,
			Value:          ts.Value,
			URL:            ts.URL,
			Duration:       ts.Duration,
			Script:         ts.Script,
			Voiceover:      ts.Voiceover,
			Description:    ts.Description,
			ScrollX:        ts.ScrollX,
			ScrollY:        ts.ScrollY,
			ScrollMode:     browser.ScrollMode(ts.ScrollMode),
			ScrollBehavior: browser.ScrollBehavior(ts.ScrollBehavior),
		}
	}

	return &BrowserSegment{
		index:       slide.Index,
		title:       slide.Title,
		url:         slide.BrowserURL,
		steps:       steps,
		transcripts: slide.Transcripts,
	}
}

// GetID returns the segment identifier
func (b *BrowserSegment) GetID() string {
	return fmt.Sprintf("segment_%03d", b.index)
}

// GetIndex returns the segment position
func (b *BrowserSegment) GetIndex() int {
	return b.index
}

// GetSourceType returns SourceTypeBrowser
func (b *BrowserSegment) GetSourceType() SourceType {
	return SourceTypeBrowser
}

// GetTitle returns the segment title
func (b *BrowserSegment) GetTitle() string {
	return b.title
}

// GetVoiceovers returns voiceovers for a specific language.
// Browser segments may have multiple voiceovers - one per step that has voiceover text.
// If transcripts are defined, they override step voiceovers.
func (b *BrowserSegment) GetVoiceovers(language string) []Voiceover {
	// Check for explicit transcripts first (multi-language support)
	if content, ok := b.transcripts[language]; ok && len(content.Segments) > 0 {
		voiceovers := make([]Voiceover, len(content.Segments))
		for i, seg := range content.Segments {
			voiceovers[i] = Voiceover{
				Index:    i,
				Text:     seg.Text,
				Language: language,
				Voice:    seg.Voice,
				Pause:    seg.Pause,
			}
		}
		return voiceovers
	}

	// Fall back to step-embedded voiceovers (single language)
	voiceovers := make([]Voiceover, 0, len(b.steps))
	for i, step := range b.steps {
		if step.Voiceover != "" {
			voiceovers = append(voiceovers, Voiceover{
				Index:       len(voiceovers),
				Text:        step.Voiceover,
				Language:    language,
				MinDuration: step.MinDuration,
				StepIndex:   i,
			})
		}
	}
	return voiceovers
}

// GetLanguages returns all available language codes
func (b *BrowserSegment) GetLanguages() []string {
	if len(b.transcripts) > 0 {
		languages := make([]string, 0, len(b.transcripts))
		for lang := range b.transcripts {
			languages = append(languages, lang)
		}
		return languages
	}

	// If no explicit transcripts, check if any steps have voiceovers
	for _, step := range b.steps {
		if step.Voiceover != "" {
			// Return empty slice to indicate "use default language"
			return []string{}
		}
	}
	return nil
}

// GetTranscripts returns the raw transcript data
func (b *BrowserSegment) GetTranscripts() map[string]transcript.LanguageContent {
	return b.transcripts
}

// GetURL returns the starting URL for the browser session
func (b *BrowserSegment) GetURL() string {
	return b.url
}

// GetSteps returns the browser automation steps
func (b *BrowserSegment) GetSteps() []browser.Step {
	return b.steps
}

// SetVideoPath sets the path to the recorded video
func (b *BrowserSegment) SetVideoPath(path string) {
	b.videoPath = path
}

// GetVideoPath returns the path to the recorded video
func (b *BrowserSegment) GetVideoPath() string {
	return b.videoPath
}

// SetTranscripts sets multi-language transcripts for the segment
func (b *BrowserSegment) SetTranscripts(transcripts map[string]transcript.LanguageContent) {
	b.transcripts = transcripts
}

// GetStepVoiceovers returns voiceover texts extracted from steps
func (b *BrowserSegment) GetStepVoiceovers() []string {
	voiceovers := make([]string, 0, len(b.steps))
	for _, step := range b.steps {
		if step.Voiceover != "" {
			voiceovers = append(voiceovers, step.Voiceover)
		}
	}
	return voiceovers
}

// LimitSteps truncates the steps to the first n steps (for testing), and
// correspondingly truncates any segment-level transcripts to the same n.
// GetVoiceovers prefers segment-level transcripts over step-embedded
// voiceovers whenever both are present, so without this, limiting steps had
// no effect on TTS generation, pacing, or final video duration for
// transcript-based (multi-language) segments — the full narration was
// synthesized and timed regardless of --limit-steps.
func (b *BrowserSegment) LimitSteps(n int) {
	if n <= 0 {
		return
	}
	if n < len(b.steps) {
		fmt.Printf("Limiting browser segment to first %d of %d steps\n", n, len(b.steps))
		b.steps = b.steps[:n]
	}
	for lang, content := range b.transcripts {
		if n < len(content.Segments) {
			content.Segments = content.Segments[:n]
			b.transcripts[lang] = content
		}
	}
}

// UpdateStepMinDurations updates minDuration for steps based on TTS durations.
// It handles two cases:
// 1. Step-embedded voiceovers: each step with a voiceover gets the corresponding duration
// 2. Transcript-based voiceovers: total duration is distributed across all steps
func (b *BrowserSegment) UpdateStepMinDurations(durations map[int]int) {
	// Count steps with embedded voiceovers
	stepsWithVoiceover := 0
	for _, step := range b.steps {
		if step.Voiceover != "" {
			stepsWithVoiceover++
		}
	}

	if stepsWithVoiceover > 0 {
		// Case 1: Step-embedded voiceovers - map durations to steps with voiceovers
		voiceoverIdx := 0
		for i := range b.steps {
			if b.steps[i].Voiceover != "" {
				if duration, ok := durations[voiceoverIdx]; ok {
					// Add buffer for natural pacing
					b.steps[i].MinDuration = duration + 500
				}
				voiceoverIdx++
			}
		}
	} else if len(durations) > 0 && len(b.steps) > 0 {
		// Case 2: Transcript-based voiceovers - distribute total duration across steps
		// Calculate total duration from all voiceovers
		totalDuration := 0
		for _, dur := range durations {
			totalDuration += dur
		}

		// Distribute duration across steps proportionally
		// Steps with longer existing Duration get more time
		totalExisting := 0
		for _, step := range b.steps {
			if step.Duration > 0 {
				totalExisting += step.Duration
			} else {
				totalExisting += 1000 // Default 1 second per step
			}
		}

		if totalExisting == 0 {
			totalExisting = len(b.steps) * 1000
		}

		// Distribute: each step gets (its proportion of existing time) * totalDuration
		for i := range b.steps {
			stepProportion := 1000 // Default
			if b.steps[i].Duration > 0 {
				stepProportion = b.steps[i].Duration
			}

			// Calculate this step's share of total audio duration
			stepShare := max((totalDuration*stepProportion)/totalExisting, 1000)
			b.steps[i].MinDuration = stepShare
		}
	}
}
