package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grokify/mogo/fmt/progress"
	omnistt "github.com/grokify/videoascode/pkg/omnivoice/stt"
	"github.com/grokify/videoascode/pkg/tts"
	"github.com/spf13/cobra"
)

var sttCmd = &cobra.Command{
	Use:   "stt",
	Short: "Generate subtitles from audio files",
	Long: `Generate subtitle files (SRT/VTT) from audio files using speech-to-text.

Supports ElevenLabs and Deepgram STT providers via OmniVoice.
Deepgram is recommended for STT due to better word-level timestamp accuracy.

This command reads a TTS manifest (from 'marp2video tts') and generates:
  - Individual subtitle files per slide (slide_001.srt, etc.)
  - Combined subtitle file for the full presentation (combined.srt, combined.vtt)

Examples:
  # Generate subtitles using Deepgram (default if DEEPGRAM_API_KEY is set)
  marp2video stt --manifest audio/manifest.json --output subtitles/

  # Generate subtitles using ElevenLabs
  marp2video stt --manifest audio/manifest.json --output subtitles/ --provider elevenlabs

  # Generate VTT format instead of SRT
  marp2video stt --manifest audio/manifest.json --output subtitles/ --format vtt

  # Specify language for better accuracy
  marp2video stt --manifest audio/manifest.json --output subtitles/ --lang en-US`,
	RunE: runSTT,
}

var (
	sttManifestFile     string
	sttOutputDir        string
	sttFormat           string
	sttLanguage         string
	sttElevenLabsAPIKey string
	sttDeepgramAPIKey   string
	sttProvider         string
	sttLocal            bool
	sttWhisperEndpoint  string
)

func init() {
	sttCmd.Flags().StringVarP(&sttManifestFile, "manifest", "m", "", "TTS manifest JSON file (required)")
	sttCmd.Flags().StringVarP(&sttOutputDir, "output", "o", "subtitles", "Output directory for subtitle files")
	sttCmd.Flags().StringVarP(&sttFormat, "format", "f", "srt", "Subtitle format: srt or vtt")
	sttCmd.Flags().StringVarP(&sttLanguage, "lang", "l", "", "Language code for transcription (e.g., en-US)")
	sttCmd.Flags().StringVar(&sttElevenLabsAPIKey, "elevenlabs-api-key", "", "ElevenLabs API key (or use ELEVENLABS_API_KEY env var)")
	sttCmd.Flags().StringVar(&sttDeepgramAPIKey, "deepgram-api-key", "", "Deepgram API key (or use DEEPGRAM_API_KEY env var)")
	sttCmd.Flags().StringVar(&sttProvider, "provider", "", "STT provider: elevenlabs, deepgram, or whisper-mlx (default: deepgram if available)")
	sttCmd.Flags().BoolVar(&sttLocal, "local", false, "Enable local STT providers (Whisper MLX); requires a local gRPC server running on Apple Silicon")
	sttCmd.Flags().StringVar(&sttWhisperEndpoint, "whisper-endpoint", "", "Whisper MLX gRPC endpoint (default: unix:///tmp/omnivoice-whisper.sock)")

	if err := sttCmd.MarkFlagRequired("manifest"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(sttCmd)
}

func runSTT(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	// Validate manifest file exists
	if _, err := os.Stat(sttManifestFile); os.IsNotExist(err) {
		return fmt.Errorf("manifest file does not exist: %s", sttManifestFile)
	}

	// Get API keys from flags or environment
	elevenLabsKey := sttElevenLabsAPIKey
	if elevenLabsKey == "" {
		elevenLabsKey = os.Getenv("ELEVENLABS_API_KEY")
	}

	deepgramKey := sttDeepgramAPIKey
	if deepgramKey == "" {
		deepgramKey = os.Getenv("DEEPGRAM_API_KEY")
	}

	// Require at least one API key, unless using a local provider
	if elevenLabsKey == "" && deepgramKey == "" && !sttLocal {
		return fmt.Errorf("STT API key required: use --elevenlabs-api-key or --deepgram-api-key flag, set ELEVENLABS_API_KEY or DEEPGRAM_API_KEY env var, or use --local for Whisper MLX")
	}

	// Load manifest
	manifest, err := tts.LoadManifest(sttManifestFile)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get audio directory from manifest path
	audioDir := filepath.Dir(sttManifestFile)

	// Parse format
	var format tts.SubtitleFormat
	switch sttFormat {
	case "srt":
		format = tts.FormatSRT
	case "vtt":
		format = tts.FormatVTT
	default:
		return fmt.Errorf("invalid format: %s (use 'srt' or 'vtt')", sttFormat)
	}

	// Determine language from manifest if not specified
	language := sttLanguage
	if language == "" {
		language = manifest.Language
	}

	fmt.Printf("Generating subtitles from audio\n")
	fmt.Printf("  Manifest: %s\n", sttManifestFile)
	fmt.Printf("  Language: %s\n", language)
	fmt.Printf("  Format:   %s\n", sttFormat)
	fmt.Printf("  Output:   %s\n", sttOutputDir)
	fmt.Printf("  Slides:   %d\n\n", len(manifest.Slides))

	// Create generator config
	genConfig := tts.SubtitleGeneratorConfig{
		ProviderConfig: omnistt.ProviderConfig{
			ElevenLabsAPIKey:     elevenLabsKey,
			DeepgramAPIKey:       deepgramKey,
			EnableLocalProviders: sttLocal,
			WhisperMLXEndpoint:   sttWhisperEndpoint,
		},
		DefaultProvider: sttProvider,
		OutputDir:       sttOutputDir,
		Format:          format,
		Language:        language,
	}

	// Only show progress bar when not in verbose mode
	var renderer *progress.SingleStageRenderer
	if !verbose {
		renderer = progress.NewSingleStageRenderer(os.Stderr).WithBarWidth(30)
		genConfig.ProgressFunc = func(current, total int, name string) {
			renderer.Update(current, total, name)
		}
	}

	generator, err := tts.NewSubtitleGenerator(genConfig)
	if err != nil {
		return fmt.Errorf("failed to create subtitle generator: %w", err)
	}

	// Generate subtitles
	result, err := generator.GenerateFromManifest(ctx, manifest, audioDir)

	// Clear progress line if we were showing it
	if renderer != nil {
		renderer.Done("")
	}

	if err != nil {
		return fmt.Errorf("failed to generate subtitles: %w", err)
	}

	// Print summary
	fmt.Printf("\n✓ Generated %d subtitle files\n", len(result.Subtitles))
	if result.CombinedSRT != "" {
		fmt.Printf("✓ Combined SRT: %s\n", filepath.Join(sttOutputDir, result.CombinedSRT))
	}
	if result.CombinedVTT != "" {
		fmt.Printf("✓ Combined VTT: %s\n", filepath.Join(sttOutputDir, result.CombinedVTT))
	}

	// Calculate total words
	totalWords := 0
	for _, sub := range result.Subtitles {
		totalWords += sub.WordCount
	}
	fmt.Printf("✓ Total words transcribed: %d\n", totalWords)

	return nil
}
