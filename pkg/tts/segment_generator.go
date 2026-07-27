package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grokify/videoascode/pkg/media"
	omnitts "github.com/grokify/videoascode/pkg/omnivoice/tts"
	"github.com/grokify/videoascode/pkg/segment"
	"github.com/grokify/videoascode/pkg/transcript"
)

// segmentMetadata stores per-voiceover duration info for caching
type segmentMetadata struct {
	SegmentID          string         `json:"segmentId"`
	TotalDuration      int            `json:"totalDuration"`
	VoiceoverDurations map[int]int    `json:"voiceoverDurations"`
	VoiceoverFiles     map[int]string `json:"voiceoverFiles"`
}

// SegmentTTSGenerator generates TTS audio for segments.
// It works with the segment.Segment interface, making it usable
// for both slides and browser demos.
type SegmentTTSGenerator struct {
	provider     *omnitts.Provider
	defaultVoice transcript.VoiceConfig
	progressFunc ProgressFunc
	pronouncer   *Pronouncer
}

// NewSegmentTTSGenerator creates a new segment-aware TTS generator.
func NewSegmentTTSGenerator(provider *omnitts.Provider, defaultVoice transcript.VoiceConfig) *SegmentTTSGenerator {
	return &SegmentTTSGenerator{
		provider:     provider,
		defaultVoice: defaultVoice,
	}
}

// SetProgressFunc sets a callback for progress updates during TTS generation.
func (g *SegmentTTSGenerator) SetProgressFunc(fn ProgressFunc) {
	g.progressFunc = fn
}

// SetPronunciations configures TTS pronunciation substitutions (see
// PronunciationDictionary). Applied only to the text sent to the TTS
// provider; voiceover/subtitle/manifest text elsewhere is unaffected. A nil
// dict clears any previously set pronunciations.
func (g *SegmentTTSGenerator) SetPronunciations(dict *PronunciationDictionary) {
	g.pronouncer = NewPronouncer(dict)
}

// reportProgress calls the progress function if set.
func (g *SegmentTTSGenerator) reportProgress(current, total int, desc string) {
	if g.progressFunc != nil {
		g.progressFunc(current, total, desc)
	}
}

// SegmentAudioResult contains TTS results for a segment.
type SegmentAudioResult struct {
	// SegmentID matches segment.GetID()
	SegmentID string

	// AudioFile is the path to the combined audio file for this segment
	AudioFile string

	// VoiceoverAudioFiles maps voiceover index to individual audio file
	VoiceoverAudioFiles map[int]string

	// Duration is the total audio duration in milliseconds
	Duration int

	// VoiceoverDurations maps voiceover index to duration in milliseconds
	VoiceoverDurations map[int]int
}

// GenerateForSegment generates TTS audio for a single segment.
// Audio files are stored as {outputDir}/{segmentID}.mp3 for easy caching and reuse.
func (g *SegmentTTSGenerator) GenerateForSegment(
	ctx context.Context,
	seg segment.Segment,
	language string,
	outputDir string,
) (*SegmentAudioResult, error) {
	voiceovers := seg.GetVoiceovers(language)
	if len(voiceovers) == 0 {
		return nil, fmt.Errorf("segment %s has no voiceovers for language %s", seg.GetID(), language)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &SegmentAudioResult{
		SegmentID:           seg.GetID(),
		VoiceoverAudioFiles: make(map[int]string),
		VoiceoverDurations:  make(map[int]int),
	}

	// Final combined audio path: {outputDir}/{segmentID}.mp3
	combinedPath := filepath.Join(outputDir, seg.GetID()+".mp3")
	metadataPath := filepath.Join(outputDir, seg.GetID()+".json")

	// Check for cached metadata with voiceover durations
	if meta, err := loadSegmentMetadata(metadataPath); err == nil {
		// Verify the combined audio file exists
		if info, err := os.Stat(combinedPath); err == nil && info.Size() > 0 {
			// Cache hit - use existing audio with metadata
			result.AudioFile = combinedPath
			result.Duration = meta.TotalDuration
			result.VoiceoverDurations = meta.VoiceoverDurations
			result.VoiceoverAudioFiles = meta.VoiceoverFiles
			return result, nil
		}
	}

	// For multiple voiceovers, use a subdirectory for intermediate files
	segDir := outputDir
	if len(voiceovers) > 1 {
		segDir = filepath.Join(outputDir, seg.GetID())
		if err := os.MkdirAll(segDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create segment directory: %w", err)
		}
	}

	// Generate audio for each voiceover
	for _, vo := range voiceovers {
		var audioPath string
		if len(voiceovers) == 1 {
			// Single voiceover: write directly to final path
			audioPath = combinedPath
		} else {
			// Multiple voiceovers: write to subdirectory
			audioPath = filepath.Join(segDir, fmt.Sprintf("voiceover_%03d.mp3", vo.Index))
		}

		var duration int
		var needsGeneration bool

		// Check if this voiceover audio already exists
		if info, err := os.Stat(audioPath); err == nil && info.Size() > 0 {
			if d, err := media.GetAudioDurationMs(audioPath); err == nil {
				duration = d
			} else {
				needsGeneration = true
			}
		} else {
			needsGeneration = true
		}

		if needsGeneration {
			voice := g.defaultVoice
			if vo.Voice != nil {
				voice = *vo.Voice
			}

			// Pronunciation substitutions affect only what's sent to TTS;
			// vo.Text itself (used for subtitles/manifests) is untouched.
			spokenText := g.pronouncer.Apply(vo.Text, language)
			audioData, err := g.synthesize(ctx, spokenText, voice)
			if err != nil {
				return nil, fmt.Errorf("TTS failed for voiceover %d: %w", vo.Index, err)
			}

			if err := os.WriteFile(audioPath, audioData, 0600); err != nil {
				return nil, fmt.Errorf("failed to write audio file: %w", err)
			}

			duration, err = media.GetAudioDurationMs(audioPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get audio duration: %w", err)
			}
		}

		result.VoiceoverAudioFiles[vo.Index] = audioPath
		result.VoiceoverDurations[vo.Index] = duration
		result.Duration += duration

		if vo.Pause > 0 {
			result.Duration += vo.Pause
		}
	}

	// Set the final audio file path
	if len(voiceovers) == 1 {
		// Single voiceover was written directly to combinedPath
		result.AudioFile = combinedPath
	} else {
		// Combine multiple voiceovers into the final combined path
		if err := g.combineAudioFiles(ctx, result.VoiceoverAudioFiles, voiceovers, combinedPath); err != nil {
			return nil, fmt.Errorf("failed to combine audio files: %w", err)
		}
		result.AudioFile = combinedPath
	}

	// Save metadata for future cache hits
	meta := segmentMetadata{
		SegmentID:          seg.GetID(),
		TotalDuration:      result.Duration,
		VoiceoverDurations: result.VoiceoverDurations,
		VoiceoverFiles:     result.VoiceoverAudioFiles,
	}
	if err := saveSegmentMetadata(metadataPath, meta); err != nil {
		// Log but don't fail - caching is optional
		fmt.Printf("Warning: failed to save audio metadata: %v\n", err)
	}

	return result, nil
}

// loadSegmentMetadata loads cached metadata from a JSON file
func loadSegmentMetadata(path string) (*segmentMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta segmentMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// saveSegmentMetadata saves metadata to a JSON file
func saveSegmentMetadata(path string, meta segmentMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GenerateForSegments generates TTS for multiple segments.
func (g *SegmentTTSGenerator) GenerateForSegments(
	ctx context.Context,
	segments []segment.Segment,
	language string,
	outputDir string,
) (map[string]*SegmentAudioResult, error) {
	results := make(map[string]*SegmentAudioResult)

	for _, seg := range segments {
		result, err := g.GenerateForSegment(ctx, seg, language, outputDir)
		if err != nil {
			return nil, fmt.Errorf("segment %s: %w", seg.GetID(), err)
		}
		results[seg.GetID()] = result
	}

	return results, nil
}

// GenerateMultiLanguage generates TTS for all segments in multiple languages.
// Returns a map of language -> segment ID -> result.
func (g *SegmentTTSGenerator) GenerateMultiLanguage(
	ctx context.Context,
	segments []segment.Segment,
	languages []string,
	outputDir string,
) (map[string]map[string]*SegmentAudioResult, error) {
	results := make(map[string]map[string]*SegmentAudioResult)

	// Count total voiceovers for progress reporting
	totalVoiceovers := 0
	for _, seg := range segments {
		for _, lang := range languages {
			totalVoiceovers += len(seg.GetVoiceovers(lang))
		}
	}

	currentVoiceover := 0
	for _, lang := range languages {
		langDir := filepath.Join(outputDir, lang)
		results[lang] = make(map[string]*SegmentAudioResult)

		for _, seg := range segments {
			result, err := g.generateForSegmentWithProgress(ctx, seg, lang, langDir, &currentVoiceover, totalVoiceovers)
			if err != nil {
				return nil, fmt.Errorf("language %s, segment %s: %w", lang, seg.GetID(), err)
			}
			results[lang][seg.GetID()] = result
		}
	}

	return results, nil
}

// generateForSegmentWithProgress generates TTS for a segment and updates progress.
func (g *SegmentTTSGenerator) generateForSegmentWithProgress(
	ctx context.Context,
	seg segment.Segment,
	language string,
	outputDir string,
	currentVoiceover *int,
	totalVoiceovers int,
) (*SegmentAudioResult, error) {
	voiceovers := seg.GetVoiceovers(language)
	if len(voiceovers) == 0 {
		return nil, fmt.Errorf("segment %s has no voiceovers for language %s", seg.GetID(), language)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &SegmentAudioResult{
		SegmentID:           seg.GetID(),
		VoiceoverAudioFiles: make(map[int]string),
		VoiceoverDurations:  make(map[int]int),
	}

	// Final combined audio path
	combinedPath := filepath.Join(outputDir, seg.GetID()+".mp3")
	metadataPath := filepath.Join(outputDir, seg.GetID()+".json")

	// Check for cached metadata
	if meta, err := loadSegmentMetadata(metadataPath); err == nil {
		if info, err := os.Stat(combinedPath); err == nil && info.Size() > 0 {
			result.AudioFile = combinedPath
			result.Duration = meta.TotalDuration
			result.VoiceoverDurations = meta.VoiceoverDurations
			result.VoiceoverAudioFiles = meta.VoiceoverFiles
			// Update progress for cached voiceovers
			*currentVoiceover += len(voiceovers)
			g.reportProgress(*currentVoiceover, totalVoiceovers, fmt.Sprintf("%s (cached)", seg.GetID()))
			return result, nil
		}
	}

	// For multiple voiceovers, use a subdirectory
	segDir := outputDir
	if len(voiceovers) > 1 {
		segDir = filepath.Join(outputDir, seg.GetID())
		if err := os.MkdirAll(segDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create segment directory: %w", err)
		}
	}

	// Generate audio for each voiceover
	for _, vo := range voiceovers {
		*currentVoiceover++
		g.reportProgress(*currentVoiceover, totalVoiceovers, fmt.Sprintf("%s vo%d", seg.GetID(), vo.Index))

		var audioPath string
		if len(voiceovers) == 1 {
			audioPath = combinedPath
		} else {
			audioPath = filepath.Join(segDir, fmt.Sprintf("voiceover_%03d.mp3", vo.Index))
		}

		var duration int
		var needsGeneration bool

		// Check if audio already exists
		if info, err := os.Stat(audioPath); err == nil && info.Size() > 0 {
			if d, err := media.GetAudioDurationMs(audioPath); err == nil {
				duration = d
			} else {
				needsGeneration = true
			}
		} else {
			needsGeneration = true
		}

		if needsGeneration {
			voice := g.defaultVoice
			if vo.Voice != nil {
				voice = *vo.Voice
			}

			// Pronunciation substitutions affect only what's sent to TTS;
			// vo.Text itself (used for subtitles/manifests) is untouched.
			spokenText := g.pronouncer.Apply(vo.Text, language)
			audioData, err := g.synthesize(ctx, spokenText, voice)
			if err != nil {
				return nil, fmt.Errorf("TTS failed for voiceover %d: %w", vo.Index, err)
			}

			if err := os.WriteFile(audioPath, audioData, 0600); err != nil {
				return nil, fmt.Errorf("failed to write audio file: %w", err)
			}

			duration, err = media.GetAudioDurationMs(audioPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get audio duration: %w", err)
			}
		}

		result.VoiceoverAudioFiles[vo.Index] = audioPath
		result.VoiceoverDurations[vo.Index] = duration
		result.Duration += duration

		if vo.Pause > 0 {
			result.Duration += vo.Pause
		}
	}

	// Set final audio path
	if len(voiceovers) == 1 {
		result.AudioFile = combinedPath
	} else {
		if err := g.combineAudioFiles(ctx, result.VoiceoverAudioFiles, voiceovers, combinedPath); err != nil {
			return nil, fmt.Errorf("failed to combine audio files: %w", err)
		}
		result.AudioFile = combinedPath
	}

	// Save metadata
	meta := segmentMetadata{
		SegmentID:          seg.GetID(),
		TotalDuration:      result.Duration,
		VoiceoverDurations: result.VoiceoverDurations,
		VoiceoverFiles:     result.VoiceoverAudioFiles,
	}
	if err := saveSegmentMetadata(metadataPath, meta); err != nil {
		fmt.Printf("Warning: failed to save audio metadata: %v\n", err)
	}

	return result, nil
}

// CalculateMaxDurations finds the maximum duration per segment across all languages.
// This is used to pace browser recordings so all languages fit.
func CalculateMaxDurations(results map[string]map[string]*SegmentAudioResult) map[string]int {
	maxDurations := make(map[string]int)

	for _, langResults := range results {
		for segID, result := range langResults {
			if result.Duration > maxDurations[segID] {
				maxDurations[segID] = result.Duration
			}
		}
	}

	return maxDurations
}

// CalculateMaxVoiceoverDurations finds the maximum duration per voiceover across languages.
// This is used for browser steps where each voiceover maps to a step.
func CalculateMaxVoiceoverDurations(results map[string]map[string]*SegmentAudioResult, segmentID string) map[int]int {
	maxDurations := make(map[int]int)

	for _, langResults := range results {
		if result, ok := langResults[segmentID]; ok {
			for idx, duration := range result.VoiceoverDurations {
				if duration > maxDurations[idx] {
					maxDurations[idx] = duration
				}
			}
		}
	}

	return maxDurations
}

// synthesize calls the TTS provider.
func (g *SegmentTTSGenerator) synthesize(ctx context.Context, text string, voice transcript.VoiceConfig) ([]byte, error) {
	// Use the omnitts.Provider wrapper which handles config conversion
	return g.provider.Synthesize(ctx, text, voice)
}

// combineAudioFiles concatenates multiple audio files with optional pauses.
func (g *SegmentTTSGenerator) combineAudioFiles(
	ctx context.Context,
	audioFiles map[int]string,
	voiceovers []segment.Voiceover,
	outputPath string,
) error {
	// Create a concat file for ffmpeg
	concatPath := outputPath + ".concat.txt"
	defer os.Remove(concatPath)

	var lines []string
	for _, vo := range voiceovers {
		audioPath, ok := audioFiles[vo.Index]
		if !ok {
			continue
		}

		// Use absolute path to avoid ffmpeg concat relative path issues
		absAudioPath, err := filepath.Abs(audioPath)
		if err != nil {
			absAudioPath = audioPath // Fall back to original if Abs fails
		}

		// Add audio file
		lines = append(lines, fmt.Sprintf("file '%s'", absAudioPath))

		// Add silence for pause if specified
		if vo.Pause > 0 {
			// Generate silence file
			silencePath := outputPath + fmt.Sprintf(".silence_%d.mp3", vo.Index)
			if err := generateSilence(ctx, vo.Pause, silencePath); err != nil {
				return fmt.Errorf("failed to generate silence: %w", err)
			}
			defer os.Remove(silencePath)
			// Use absolute path for silence too
			absSilencePath, err := filepath.Abs(silencePath)
			if err != nil {
				absSilencePath = silencePath
			}
			lines = append(lines, fmt.Sprintf("file '%s'", absSilencePath))
		}
	}

	concatContent := ""
	for _, line := range lines {
		concatContent += line + "\n"
	}

	if err := os.WriteFile(concatPath, []byte(concatContent), 0600); err != nil {
		return fmt.Errorf("failed to write concat file: %w", err)
	}

	// Run ffmpeg concat
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatPath,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	}

	if output, err := runFFmpeg(ctx, args...); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// generateSilence creates a silent audio file of the specified duration.
func generateSilence(ctx context.Context, durationMs int, outputPath string) error {
	durationSec := float64(durationMs) / 1000.0

	args := []string{
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%.3f", durationSec),
		"-c:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	}

	if output, err := runFFmpeg(ctx, args...); err != nil {
		return fmt.Errorf("ffmpeg silence generation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// runFFmpeg executes an ffmpeg command.
func runFFmpeg(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	return cmd.CombinedOutput()
}
