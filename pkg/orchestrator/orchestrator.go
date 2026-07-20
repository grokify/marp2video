package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grokify/mogo/fmt/progress"
	"github.com/grokify/videoascode/pkg/audio"
	"github.com/grokify/videoascode/pkg/avatar"
	omnitts "github.com/grokify/videoascode/pkg/omnivoice/tts"
	"github.com/grokify/videoascode/pkg/parser"
	"github.com/grokify/videoascode/pkg/renderer"
	"github.com/grokify/videoascode/pkg/tts"
	"github.com/grokify/videoascode/pkg/video"
	"github.com/plexusone/omniavatar-core/render"
)

// writeFileSecure validates that path contains no ".." traversal sequences,
// then writes data to the cleaned path with mode 0600.
func writeFileSecure(path string, data []byte) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..' traversal sequence: %s", path)
	}
	cleanPath := filepath.Clean(path)
	return os.WriteFile(cleanPath, data, 0600) //nolint:gosec // G703: Path validated above - no '..' allowed
}

// Config holds orchestrator configuration
type Config struct {
	InputFile           string
	OutputFile          string
	WorkDir             string
	ElevenLabsAPIKey    string
	VoiceID             string
	Width               int
	Height              int
	FrameRate           int
	OutputIndividualDir string        // Directory for individual slide videos (Udemy)
	TransitionDuration  float64       // Duration of transitions between slides in seconds
	ScreenDevice        string        // Screen capture device (macOS, auto-detected if empty)
	AudioManifest       string        // Path to audio manifest file (from 'marp2video tts')
	ProgressWriter      io.Writer     // Writer for progress output (nil to disable)
	Avatar              *AvatarConfig // Optional talking-head presenter overlay (nil to disable)
}

// AvatarConfig configures the optional avatar presenter overlay stage.
// When set, after the slides video is combined, narration audio is
// concatenated, a presenter video is generated via the render provider,
// and the presenter circle is composited onto the final output.
type AvatarConfig struct {
	// Provider is the constructed OmniAvatar render provider.
	// The provider must support audio upload (render.AudioUploader),
	// since the orchestrator generates narration locally.
	Provider render.Provider

	// AvatarID identifies the presenter with the provider.
	AvatarID string

	// Extensions holds provider-specific request options.
	Extensions map[string]any

	// PollInterval is the generation job status poll interval.
	PollInterval time.Duration

	// CacheDir enables presenter video caching (empty disables).
	CacheDir string

	// Overlay configures the circular overlay composition.
	Overlay video.OverlayOptions
}

// Orchestrator coordinates the entire video generation process
type Orchestrator struct {
	config   Config
	progress *progress.MultiStageRenderer
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(config Config) *Orchestrator {
	// Set defaults
	if config.Width == 0 {
		config.Width = 1920
	}
	if config.Height == 0 {
		config.Height = 1080
	}
	if config.FrameRate == 0 {
		config.FrameRate = 30
	}
	if config.WorkDir == "" {
		config.WorkDir = filepath.Join(os.TempDir(), "videoascode")
	}

	o := &Orchestrator{config: config}

	// Initialize progress renderer if writer is provided
	if config.ProgressWriter != nil {
		o.progress = progress.NewMultiStageRenderer(config.ProgressWriter).
			WithBarWidth(20).
			WithDescWidth(30)
	}

	return o
}

// totalStages returns the number of pipeline stages: 5 base, plus one each
// for individual-video export and the avatar presenter overlay.
func (o *Orchestrator) totalStages() int {
	n := 5
	if o.config.OutputIndividualDir != "" {
		n++
	}
	if o.config.Avatar != nil {
		n++
	}
	return n
}

// updateProgress updates the progress display if enabled
func (o *Orchestrator) updateProgress(stage int, desc string, current, total int, done bool, text ...string) {
	if o.progress == nil {
		return
	}
	info := progress.StageInfo{
		Stage:       stage,
		TotalStages: o.totalStages(),
		Description: desc,
		Current:     current,
		Total:       total,
		Done:        done,
	}
	if len(text) > 0 {
		info.Text = text[0]
	}
	o.progress.Update(info)
}

// Process orchestrates the entire conversion process
func (o *Orchestrator) Process(ctx context.Context) error {
	// Create working directories
	audioDir := filepath.Join(o.config.WorkDir, "audio")
	videoDir := filepath.Join(o.config.WorkDir, "video")
	htmlDir := filepath.Join(o.config.WorkDir, "html")

	for _, dir := range []string{audioDir, videoDir, htmlDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create output directory if needed
	if outputDir := filepath.Dir(o.config.OutputFile); outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	// Step 1: Parse Marp markdown
	o.updateProgress(1, "Parsing Marp markdown", 0, 0, false, o.config.InputFile)
	content, err := os.ReadFile(o.config.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	presentation, err := parser.ParseMarpFile(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse Marp file: %w", err)
	}
	o.updateProgress(1, "Parsing Marp markdown", 0, 0, true, fmt.Sprintf("%d slides", len(presentation.Slides)))

	// Step 2: Get audio for each slide (from manifest or generate)
	numSlides := len(presentation.Slides)
	audioFiles := make([]string, numSlides)
	slideDurations := make([]time.Duration, numSlides)

	if o.config.AudioManifest != "" {
		// Use pre-generated audio from manifest
		o.updateProgress(2, "Loading audio manifest", 0, 0, false, o.config.AudioManifest)

		manifest, err := tts.LoadManifest(o.config.AudioManifest)
		if err != nil {
			return fmt.Errorf("failed to load audio manifest: %w", err)
		}

		manifestDir := filepath.Dir(o.config.AudioManifest)
		for i := 0; i < numSlides; i++ {
			slideAudio, err := manifest.GetSlide(i)
			if err != nil {
				o.updateProgress(2, "Loading audio manifest", i+1, numSlides, false, fmt.Sprintf("slide %d: no audio", i))
				continue
			}

			audioFiles[i] = filepath.Join(manifestDir, slideAudio.AudioFile)
			slideDurations[i] = time.Duration(slideAudio.TotalDuration) * time.Millisecond

			o.updateProgress(2, "Loading audio manifest", i+1, numSlides, false, fmt.Sprintf("slide %d: %.1fs", i, slideDurations[i].Seconds()))
		}
		o.updateProgress(2, "Loading audio manifest", numSlides, numSlides, true)
	} else {
		// Generate audio with OmniVoice TTS provider
		ttsGen, err := tts.NewGenerator(tts.Config{
			ProviderConfig: omnitts.ProviderConfig{
				ElevenLabsAPIKey: o.config.ElevenLabsAPIKey,
			},
			VoiceID:   o.config.VoiceID,
			OutputDir: audioDir,
		})
		if err != nil {
			return fmt.Errorf("failed to create TTS generator: %w", err)
		}

		audioPlayer := audio.NewPlayer()

		for i, slide := range presentation.Slides {
			if slide.Voiceover == "" {
				o.updateProgress(2, "Generating audio", i+1, numSlides, false, fmt.Sprintf("slide %d: skipped", i))
				continue
			}

			o.updateProgress(2, "Generating audio", i+1, numSlides, false, fmt.Sprintf("slide %d: generating...", i))
			audioResult, err := ttsGen.GenerateAudio(ctx, slide.Voiceover, i)
			if err != nil {
				return fmt.Errorf("failed to generate audio for slide %d: %w", i, err)
			}

			audioFiles[i] = audioResult.FilePath

			// Get actual audio duration using ffprobe
			duration, err := audioPlayer.GetDuration(audioResult.FilePath)
			if err != nil {
				return fmt.Errorf("failed to get audio duration for slide %d: %w", i, err)
			}

			// Add pause durations
			totalDuration := duration + time.Duration(slide.TotalPauseDuration)*time.Millisecond
			slideDurations[i] = totalDuration

			o.updateProgress(2, "Generating audio", i+1, numSlides, false, fmt.Sprintf("slide %d: %.1fs", i, totalDuration.Seconds()))
		}
		o.updateProgress(2, "Generating audio", numSlides, numSlides, true)
	}

	// Step 3: Render Marp to HTML
	o.updateProgress(3, "Rendering HTML", 0, 0, false)
	marpRenderer := renderer.NewMarpRenderer()
	if err := marpRenderer.CheckMarpCLI(); err != nil {
		return err
	}

	htmlPath, err := marpRenderer.RenderToHTML(o.config.InputFile, htmlDir)
	if err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}
	_ = htmlPath // HTML created for reference; using images for video
	o.updateProgress(3, "Rendering HTML", 0, 0, true, htmlPath)

	// Step 4: Render slides to images and create videos
	o.updateProgress(4, "Creating slide videos", 0, numSlides, false, "rendering images...")

	// Render Marp to PNG images (one per slide)
	imagesDir := filepath.Join(o.config.WorkDir, "images")
	imagePaths, err := marpRenderer.RenderToImages(o.config.InputFile, imagesDir)
	if err != nil {
		return fmt.Errorf("failed to render images: %w", err)
	}
	o.updateProgress(4, "Creating slide videos", 0, numSlides, false, fmt.Sprintf("%d images rendered", len(imagePaths)))

	// Sort image paths to ensure correct order
	sort.Strings(imagePaths)

	// Create video converter
	converter := video.NewImageVideoConverter(video.ImageVideoConfig{
		OutputDir: videoDir,
		Width:     o.config.Width,
		Height:    o.config.Height,
		FrameRate: o.config.FrameRate,
	})

	// Create directory for padded audio files
	paddedAudioDir := filepath.Join(o.config.WorkDir, "audio_padded")
	if err := os.MkdirAll(paddedAudioDir, 0755); err != nil {
		return fmt.Errorf("failed to create padded audio directory: %w", err)
	}

	audioPlayer := audio.NewPlayer()
	videoFiles := make([]string, 0, len(presentation.Slides))
	// paddedAudioFiles tracks per-slide padded audio in the same order as
	// videoFiles, so the avatar stage can concatenate a narration track
	// that matches the combined video's timeline exactly.
	paddedAudioFiles := make([]string, 0, len(presentation.Slides))

	// Create video for each slide with audio
	for i := range presentation.Slides {
		if audioFiles[i] == "" {
			o.updateProgress(4, "Creating slide videos", i+1, numSlides, false, fmt.Sprintf("slide %d: skipped", i))
			continue
		}

		// Find corresponding image (marp generates slide.001.png, slide.002.png, etc.)
		if i >= len(imagePaths) {
			return fmt.Errorf("no image found for slide %d", i)
		}
		imagePath := imagePaths[i]

		// Pad audio with silence to match target duration (includes pause time)
		o.updateProgress(4, "Creating slide videos", i+1, numSlides, false, fmt.Sprintf("slide %d: padding audio...", i))
		paddedAudioPath, err := audioPlayer.PadToLength(audioFiles[i], slideDurations[i], paddedAudioDir)
		if err != nil {
			return fmt.Errorf("failed to pad audio for slide %d: %w", i, err)
		}

		// Create video from image + padded audio (use -shortest so video matches audio exactly)
		o.updateProgress(4, "Creating slide videos", i+1, numSlides, false, fmt.Sprintf("slide %d: encoding %.1fs", i, slideDurations[i].Seconds()))
		videoPath, err := converter.CreateSlideVideoWithSize(ctx, i, imagePath, paddedAudioPath, slideDurations[i], o.config.Width, o.config.Height)
		if err != nil {
			return fmt.Errorf("failed to create video for slide %d: %w", i, err)
		}

		videoFiles = append(videoFiles, videoPath)
		paddedAudioFiles = append(paddedAudioFiles, paddedAudioPath)
	}
	o.updateProgress(4, "Creating slide videos", numSlides, numSlides, true, fmt.Sprintf("%d videos", len(videoFiles)))

	// Step 5: Combine all videos (for YouTube).
	// When an avatar overlay is requested, combine to an intermediate file
	// so the overlay stage can read it and write the final output.
	combineStage := 5
	combineTarget := o.config.OutputFile
	if o.config.Avatar != nil {
		combineTarget = filepath.Join(videoDir, "slides_combined.mp4")
	}
	o.updateProgress(combineStage, "Combining videos", 0, 0, false, fmt.Sprintf("%d videos", len(videoFiles)))
	combiner := video.NewCombiner(videoDir)

	var combineErr error
	if o.config.TransitionDuration > 0 {
		combineErr = combiner.CombineVideosWithTransitions(ctx, videoFiles, combineTarget, o.config.TransitionDuration)
	} else {
		combineErr = combiner.CombineVideos(ctx, videoFiles, combineTarget)
	}
	if combineErr != nil {
		return fmt.Errorf("failed to combine videos: %w", combineErr)
	}
	o.updateProgress(combineStage, "Combining videos", 0, 0, true, combineTarget)

	// Step 6: Save individual videos if requested (for Udemy)
	if o.config.OutputIndividualDir != "" {
		exportStage := 6
		numVideos := len(videoFiles)
		o.updateProgress(exportStage, "Exporting individual videos", 0, numVideos, false)

		if err := os.MkdirAll(o.config.OutputIndividualDir, 0755); err != nil {
			return fmt.Errorf("failed to create individual output directory: %w", err)
		}

		for i, videoPath := range videoFiles {
			// Find the slide index from the video filename
			baseName := filepath.Base(videoPath)
			destPath := filepath.Join(o.config.OutputIndividualDir, baseName)

			o.updateProgress(exportStage, "Exporting individual videos", i+1, numVideos, false, baseName)

			// Copy the video file
			data, err := os.ReadFile(videoPath)
			if err != nil {
				return fmt.Errorf("failed to read video %d: %w", i, err)
			}
			if err := writeFileSecure(destPath, data); err != nil {
				return fmt.Errorf("failed to write video %d: %w", i, err)
			}
		}
		o.updateProgress(exportStage, "Exporting individual videos", numVideos, numVideos, true, o.config.OutputIndividualDir)
	}

	// Optional stage: avatar presenter overlay.
	if o.config.Avatar != nil {
		if err := o.processAvatar(ctx, combineTarget, paddedAudioFiles); err != nil {
			return err
		}
	}

	return nil
}

// processAvatar runs the optional avatar presenter overlay: concatenate
// narration matching the final timeline, generate the presenter video via
// the render provider, and composite the presenter circle onto the slides
// video, writing the final output.
func (o *Orchestrator) processAvatar(ctx context.Context, slidesVideoPath string, paddedAudioFiles []string) error {
	av := o.config.Avatar

	stage := 6
	if o.config.OutputIndividualDir != "" {
		stage = 7
	}
	const steps = 3

	if len(paddedAudioFiles) == 0 {
		return fmt.Errorf("avatar overlay requires narration audio, but no slides had audio")
	}

	// Step 1: concatenate per-slide padded audio into one narration track
	// that matches the combined video's timeline.
	o.updateProgress(stage, "Adding avatar presenter", 0, steps, false, "concatenating narration...")
	narrationPath := filepath.Join(o.config.WorkDir, "narration.mp3")
	if err := avatar.ConcatAudioFiles(ctx, paddedAudioFiles, narrationPath); err != nil {
		return fmt.Errorf("failed to concatenate narration audio: %w", err)
	}

	// Step 2: generate the presenter video (cached by narration + config).
	o.updateProgress(stage, "Adding avatar presenter", 1, steps, false, "generating presenter...")
	presenterPath := filepath.Join(o.config.WorkDir, "presenter.mp4")
	if err := avatar.Generate(ctx, presenterPath, avatar.GenerateOptions{
		Provider:     av.Provider,
		AvatarID:     av.AvatarID,
		AudioPath:    narrationPath,
		Extensions:   av.Extensions,
		PollInterval: av.PollInterval,
		CacheDir:     av.CacheDir,
	}); err != nil {
		return fmt.Errorf("failed to generate presenter video: %w", err)
	}

	// Step 3: composite the presenter circle onto the slides video, using
	// the concatenated narration as the authoritative audio track.
	o.updateProgress(stage, "Adding avatar presenter", 2, steps, false, "compositing overlay...")
	overlay := av.Overlay
	overlay.AudioPath = narrationPath
	if err := video.OverlayAvatar(ctx, slidesVideoPath, presenterPath, o.config.OutputFile, overlay); err != nil {
		return fmt.Errorf("failed to composite avatar overlay: %w", err)
	}
	o.updateProgress(stage, "Adding avatar presenter", steps, steps, true, o.config.OutputFile)
	return nil
}
