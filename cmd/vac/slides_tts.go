package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grokify/mogo/fmt/progress"
	omnitts "github.com/grokify/videoascode/pkg/omnivoice/tts"
	"github.com/grokify/videoascode/pkg/transcript"
	"github.com/grokify/videoascode/pkg/tts"
	"github.com/spf13/cobra"
)

var slidesTTSCmd = &cobra.Command{
	Use:   "tts",
	Short: "Generate audio from transcript",
	Long: `Generate audio files from a transcript JSON file using TTS providers.

Supports ElevenLabs and Deepgram TTS providers via OmniVoice.

This command processes a transcript.json file and generates:
  - One MP3 audio file per slide
  - A manifest.json with timing information

The manifest can be used by 'marp2video slides video' to synchronize
slide recordings with the pre-generated audio.

By default, existing audio files are skipped (not regenerated). This allows
resuming interrupted runs without re-generating already completed slides.
Use --force to regenerate all audio files.

Examples:
  # Generate audio using ElevenLabs (default if ELEVENLABS_API_KEY is set)
  marp2video slides tts --transcript transcript.json --output audio/

  # Generate audio using Deepgram
  marp2video slides tts --transcript transcript.json --output audio/ --provider deepgram

  # Force regeneration of all audio files
  marp2video slides tts --transcript transcript.json --output audio/ --force

  # Generate audio for specific language
  marp2video slides tts --transcript transcript.json --output audio/ --lang fr-FR`,
	RunE: runSlidesTTS,
}

var (
	stTranscriptFile   string
	stOutputDir        string
	stLanguage         string
	stElevenLabsAPIKey string
	stDeepgramAPIKey   string
	stProvider         string
	stForce            bool
	stLocal            bool
	stF5TTSEndpoint    string
)

func init() {
	slidesTTSCmd.Flags().StringVarP(&stTranscriptFile, "transcript", "t", "", "Transcript JSON file (required)")
	slidesTTSCmd.Flags().StringVarP(&stOutputDir, "output", "o", "audio", "Output directory for audio files")
	slidesTTSCmd.Flags().StringVarP(&stLanguage, "lang", "l", "", "Language/locale code (e.g., en-US, es-ES)")
	slidesTTSCmd.Flags().StringVar(&stElevenLabsAPIKey, "elevenlabs-api-key", "", "ElevenLabs API key (or use ELEVENLABS_API_KEY env var)")
	slidesTTSCmd.Flags().StringVar(&stDeepgramAPIKey, "deepgram-api-key", "", "Deepgram API key (or use DEEPGRAM_API_KEY env var)")
	slidesTTSCmd.Flags().StringVar(&stProvider, "provider", "", "TTS provider: elevenlabs, deepgram, or f5tts-mlx (overrides voice config if set)")
	slidesTTSCmd.Flags().BoolVarP(&stForce, "force", "f", false, "Regenerate audio even if files already exist")
	slidesTTSCmd.Flags().BoolVar(&stLocal, "local", false, "Enable local TTS providers (F5-TTS MLX); requires a local gRPC server running on Apple Silicon")
	slidesTTSCmd.Flags().StringVar(&stF5TTSEndpoint, "f5tts-endpoint", "", "F5-TTS MLX gRPC endpoint (default: unix:///tmp/omnivoice-f5tts.sock)")

	if err := slidesTTSCmd.MarkFlagRequired("transcript"); err != nil {
		panic(err)
	}

	slidesCmd.AddCommand(slidesTTSCmd)
}

func runSlidesTTS(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	// Validate transcript file exists
	if _, err := os.Stat(stTranscriptFile); os.IsNotExist(err) {
		return fmt.Errorf("transcript file does not exist: %s", stTranscriptFile)
	}

	// Get API keys from flags or environment
	elevenLabsKey := stElevenLabsAPIKey
	if elevenLabsKey == "" {
		elevenLabsKey = os.Getenv("ELEVENLABS_API_KEY")
	}

	deepgramKey := stDeepgramAPIKey
	if deepgramKey == "" {
		deepgramKey = os.Getenv("DEEPGRAM_API_KEY")
	}

	// Require at least one API key, unless using a local provider
	if elevenLabsKey == "" && deepgramKey == "" && !stLocal {
		return fmt.Errorf("TTS API key required: use --elevenlabs-api-key or --deepgram-api-key flag, set ELEVENLABS_API_KEY or DEEPGRAM_API_KEY env var, or use --local for F5-TTS MLX")
	}

	// Load transcript
	t, err := transcript.LoadFromFile(stTranscriptFile)
	if err != nil {
		return fmt.Errorf("failed to load transcript: %w", err)
	}

	// Determine language
	language := stLanguage
	if language == "" {
		language = t.Metadata.DefaultLanguage
	}
	if language == "" {
		return fmt.Errorf("no language specified and no default language in transcript")
	}

	fmt.Printf("Generating audio from transcript\n")
	fmt.Printf("  Transcript: %s\n", stTranscriptFile)
	fmt.Printf("  Language:   %s\n", language)
	fmt.Printf("  Output:     %s\n", stOutputDir)
	fmt.Printf("  Slides:     %d\n\n", len(t.Slides))

	// Create generator config
	genConfig := tts.TranscriptGeneratorConfig{
		ProviderConfig: omnitts.ProviderConfig{
			ElevenLabsAPIKey:     elevenLabsKey,
			DeepgramAPIKey:       deepgramKey,
			EnableLocalProviders: stLocal,
			F5TTSMLXEndpoint:     stF5TTSEndpoint,
		},
		DefaultProvider: stProvider,
		OutputDir:       stOutputDir,
		Force:           stForce,
	}

	// Only show progress bar when not in verbose mode (logs provide progress info)
	var renderer *progress.SingleStageRenderer
	if !verbose {
		renderer = progress.NewSingleStageRenderer(os.Stderr).WithBarWidth(30)
		genConfig.ProgressFunc = func(current, total int, name string) {
			renderer.Update(current, total, name)
		}
	}

	generator, err := tts.NewTranscriptGenerator(genConfig)
	if err != nil {
		return fmt.Errorf("failed to create TTS generator: %w", err)
	}

	// Generate audio
	manifest, err := generator.GenerateFromTranscript(ctx, t, language)

	// Clear progress line if we were showing it
	if renderer != nil {
		renderer.Done("")
	}

	if err != nil {
		return fmt.Errorf("failed to generate audio: %w", err)
	}

	// Save manifest
	manifestPath := filepath.Join(stOutputDir, "manifest.json")
	if err := manifest.SaveToFile(manifestPath); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Print summary
	fmt.Printf("\n✓ Generated %d audio files\n", len(manifest.Slides))
	fmt.Printf("✓ Manifest saved to: %s\n", manifestPath)
	fmt.Printf("✓ Total duration: %.1f seconds\n", float64(manifest.GetTotalDuration())/1000)

	return nil
}
