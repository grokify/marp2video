package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grokify/videoascode/pkg/config"
	omnitts "github.com/grokify/videoascode/pkg/omnivoice/tts"
	"github.com/grokify/videoascode/pkg/orchestrator"
	"github.com/grokify/videoascode/pkg/source"
	"github.com/grokify/videoascode/pkg/tts"
	"github.com/grokify/videoascode/pkg/video"
	"github.com/spf13/cobra"
)

var browserVideoCmd = &cobra.Command{
	Use:   "video",
	Short: "Record browser demo with TTS voiceover",
	Long: `Record a browser-driven demo and combine it with AI-generated voiceover.

Supports ElevenLabs and Deepgram TTS providers via OmniVoice.

This command:
1. Loads a config file defining browser steps with voiceover text
2. Generates TTS audio for each step (supports multiple languages)
3. Records the browser session with timing matched to audio
4. Combines video and audio into final output

For silent browser recording without voiceover, use 'vac browser record' instead.

The config file (YAML or JSON) should define browser segments with steps
that include voiceover text.

Examples:
  # Record demo using ElevenLabs (default if ELEVENLABS_API_KEY is set)
  vac browser video --config demo.yaml --output demo.mp4

  # Record demo using Deepgram
  vac browser video --config demo.yaml --output demo.mp4 --provider deepgram

  # Generate multiple languages
  vac browser video --config demo.yaml --output demo.mp4 --lang en-US,fr-FR

  # Use specific voice
  vac browser video --config demo.yaml --output demo.mp4 --voice pNInz6obpgDQGcFmaJgB`,
	RunE: runBrowserVideo,
}

var (
	bvConfigFile       string
	bvOutputFile       string
	bvAudioDir         string
	bvElevenLabsAPIKey string
	bvDeepgramAPIKey   string
	bvVoiceID          string
	bvLanguages        []string
	bvWidth            int
	bvHeight           int
	bvFPS              int
	bvWorkDir          string
	bvHeadless         bool
	bvTransition       float64
	bvProvider         string
	bvSubtitles        bool
	bvSubtitlesSTT     bool
	bvSubtitlesBurn    bool
	bvNoAudio          bool
	bvParallel         int
	bvFast             bool
	bvLimit            int
	bvLimitSteps       int
	bvLocal            bool
	bvF5TTSEndpoint    string
)

func init() {
	browserVideoCmd.Flags().StringVarP(&bvConfigFile, "config", "c", "", "Configuration file (YAML/JSON) with browser segments (required)")
	browserVideoCmd.Flags().StringVarP(&bvOutputFile, "output", "o", "output.mp4", "Output video file")
	browserVideoCmd.Flags().StringVarP(&bvAudioDir, "audio-dir", "a", "", "Save audio tracks to this directory (per-language subdirs)")
	browserVideoCmd.Flags().StringVar(&bvElevenLabsAPIKey, "elevenlabs-api-key", "", "ElevenLabs API key (or use ELEVENLABS_API_KEY env var)")
	browserVideoCmd.Flags().StringVar(&bvDeepgramAPIKey, "deepgram-api-key", "", "Deepgram API key (or use DEEPGRAM_API_KEY env var)")
	browserVideoCmd.Flags().StringVarP(&bvProvider, "provider", "p", "", "TTS provider: elevenlabs, deepgram, or f5tts-mlx (default: from config or auto-detect)")
	browserVideoCmd.Flags().BoolVar(&bvLocal, "local", false, "Enable local TTS providers (F5-TTS MLX); requires a local gRPC server running on Apple Silicon")
	browserVideoCmd.Flags().StringVar(&bvF5TTSEndpoint, "f5tts-endpoint", "", "F5-TTS MLX gRPC endpoint (default: unix:///tmp/omnivoice-f5tts.sock)")
	browserVideoCmd.Flags().StringVarP(&bvVoiceID, "voice", "v", "", "TTS voice ID (default: from config or provider default)")
	browserVideoCmd.Flags().StringSliceVarP(&bvLanguages, "lang", "l", []string{"en-US"}, "Languages to generate (comma-separated)")
	browserVideoCmd.Flags().IntVar(&bvWidth, "width", 1920, "Video width")
	browserVideoCmd.Flags().IntVar(&bvHeight, "height", 1080, "Video height")
	browserVideoCmd.Flags().IntVar(&bvFPS, "fps", 30, "Video frame rate")
	browserVideoCmd.Flags().StringVar(&bvWorkDir, "workdir", "", "Working directory for temp files")
	browserVideoCmd.Flags().BoolVar(&bvHeadless, "headless", false, "Run browser in headless mode")
	browserVideoCmd.Flags().Float64Var(&bvTransition, "transition", 0, "Transition duration between segments (seconds)")
	browserVideoCmd.Flags().BoolVar(&bvSubtitles, "subtitles", false, "Generate subtitles from voiceover timing (no STT)")
	browserVideoCmd.Flags().BoolVar(&bvSubtitlesSTT, "subtitles-stt", false, "Generate word-level subtitles using STT (requires API)")
	browserVideoCmd.Flags().BoolVar(&bvSubtitlesBurn, "subtitles-burn", false, "Burn subtitles into video (permanent, no toggle)")
	browserVideoCmd.Flags().BoolVar(&bvNoAudio, "no-audio", false, "Generate video without audio (TTS still used for timing/subtitles)")
	browserVideoCmd.Flags().IntVar(&bvParallel, "parallel", 1, "Number of segments to record in parallel (default 1 = sequential)")
	browserVideoCmd.Flags().BoolVar(&bvFast, "fast", false, "Use hardware-accelerated encoding (VideoToolbox/NVENC) for faster video generation")
	browserVideoCmd.Flags().IntVar(&bvLimit, "limit", 0, "Limit to first N segments (0 = no limit, useful for testing)")
	browserVideoCmd.Flags().IntVar(&bvLimitSteps, "limit-steps", 0, "Limit browser segments to first N steps (0 = no limit, useful for testing)")

	if err := browserVideoCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	browserParentCmd.AddCommand(browserVideoCmd)
}

func runBrowserVideo(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	// Load config first (needed to determine default provider)
	cfg, err := config.LoadFromFile(bvConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get API keys from flags or environment
	elevenLabsKey := bvElevenLabsAPIKey
	if elevenLabsKey == "" {
		elevenLabsKey = os.Getenv("ELEVENLABS_API_KEY")
	}

	deepgramKey := bvDeepgramAPIKey
	if deepgramKey == "" {
		deepgramKey = os.Getenv("DEEPGRAM_API_KEY")
	}

	// Require at least one API key, unless using a local provider
	if elevenLabsKey == "" && deepgramKey == "" && !bvLocal {
		return fmt.Errorf("TTS API key required: use --elevenlabs-api-key or --deepgram-api-key flag, set ELEVENLABS_API_KEY or DEEPGRAM_API_KEY env var, or use --local for F5-TTS MLX")
	}

	// Determine provider
	provider := bvProvider
	if provider == "" {
		// Check config first
		if cfg.DefaultVoice.Provider != "" {
			provider = cfg.DefaultVoice.Provider
		} else {
			// Auto-detect based on available API keys, then local providers
			if elevenLabsKey != "" {
				provider = "elevenlabs"
			} else if deepgramKey != "" {
				provider = "deepgram"
			} else if bvLocal {
				provider = "f5tts-mlx"
			}
		}
	}

	// Validate provider has corresponding API key (local providers need no key)
	switch provider {
	case "deepgram":
		if deepgramKey == "" {
			return fmt.Errorf("Deepgram provider selected but no API key: use --deepgram-api-key or set DEEPGRAM_API_KEY env var")
		}
	case "elevenlabs":
		if elevenLabsKey == "" {
			return fmt.Errorf("ElevenLabs provider selected but no API key: use --elevenlabs-api-key or set ELEVENLABS_API_KEY env var")
		}
	case "f5tts-mlx":
		if !bvLocal {
			return fmt.Errorf("F5-TTS MLX provider selected but local providers not enabled: use --local")
		}
	default:
		return fmt.Errorf("unknown TTS provider: %s (supported: elevenlabs, deepgram, f5tts-mlx)", provider)
	}

	// Validate config has browser segments
	hasBrowser := false
	for _, seg := range cfg.Segments {
		if seg.Type == config.SourceTypeBrowser {
			hasBrowser = true
			break
		}
	}
	if !hasBrowser {
		return fmt.Errorf("config has no browser segments")
	}

	// Set up working directory
	workDir := bvWorkDir
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "vac-browser-video")
	}

	// Create TTS provider config with both keys
	providerCfg := omnitts.ProviderConfig{
		ElevenLabsAPIKey:     elevenLabsKey,
		DeepgramAPIKey:       deepgramKey,
		EnableLocalProviders: bvLocal,
		F5TTSMLXEndpoint:     bvF5TTSEndpoint,
	}
	factory := omnitts.NewFactory(providerCfg)
	ttsProvider, err := factory.Get(provider)
	if err != nil {
		return fmt.Errorf("failed to create TTS provider: %w", err)
	}

	// Determine voice ID
	voiceID := bvVoiceID
	if voiceID == "" {
		// Check config
		if cfg.DefaultVoice.VoiceID != "" {
			voiceID = cfg.DefaultVoice.VoiceID
		} else {
			// Use provider defaults
			switch provider {
			case "elevenlabs":
				voiceID = "pNInz6obpgDQGcFmaJgB" // Adam
			case "deepgram":
				voiceID = "aura-asteria-en" // Default Deepgram voice
			}
		}
	}

	// Create default voice config, starting from the file config so fields
	// like OutputFormat/SampleRate/Stability/etc. aren't silently dropped,
	// then applying only the CLI-flag-or-auto-detected provider/voice ID.
	defaultVoice := cfg.DefaultVoice
	defaultVoice.Provider = provider
	defaultVoice.VoiceID = voiceID
	if provider == "elevenlabs" && defaultVoice.Model == "" {
		defaultVoice.Model = "eleven_multilingual_v2"
	}

	ttsGen := tts.NewSegmentTTSGenerator(ttsProvider, defaultVoice)
	if len(cfg.Pronunciations) > 0 {
		ttsGen.SetPronunciations(&tts.PronunciationDictionary{Terms: cfg.Pronunciations})
	}

	// Create content source
	contentSource := source.NewConfigSource(cfg)

	// Create orchestrator config
	orchConfig := orchestrator.SegmentConfig{
		Source:             contentSource,
		OutputFile:         bvOutputFile,
		WorkDir:            workDir,
		AudioOutputDir:     bvAudioDir,
		Languages:          bvLanguages,
		Width:              bvWidth,
		Height:             bvHeight,
		FrameRate:          bvFPS,
		TransitionDuration: bvTransition,
		Headless:           bvHeadless,
		ProgressWriter:     os.Stdout,
		Subtitles:          bvSubtitles,
		SubtitlesSTT:       bvSubtitlesSTT,
		SubtitlesBurn:      bvSubtitlesBurn,
		SubtitleFormat:     "srt",
		DeepgramAPIKey:     deepgramKey,
		NoAudio:            bvNoAudio,
		Parallel:           bvParallel,
		FastEncoding:       bvFast,
		SegmentLimit:       bvLimit,
		StepLimit:          bvLimitSteps,
	}

	// Print summary
	fmt.Printf("Browser Video Recording\n")
	fmt.Printf("=======================\n")
	fmt.Printf("Config:     %s\n", bvConfigFile)
	fmt.Printf("Output:     %s\n", bvOutputFile)
	if bvAudioDir != "" {
		fmt.Printf("Audio Dir:  %s\n", bvAudioDir)
	}
	fmt.Printf("Provider:   %s\n", provider)
	fmt.Printf("Voice ID:   %s\n", voiceID)
	fmt.Printf("Resolution: %dx%d @ %d fps\n", bvWidth, bvHeight, bvFPS)
	fmt.Printf("Languages:  %v\n", bvLanguages)
	fmt.Printf("Headless:   %v\n", bvHeadless)
	if bvParallel > 1 {
		fmt.Printf("Parallel:   %d segments\n", bvParallel)
	}
	if bvFast {
		fmt.Printf("Encoder:    %s (fast)\n", video.GetEncoderDescription(video.FastEncoderConfig()))
	}
	fmt.Println()

	// Create and run orchestrator
	orch := orchestrator.NewSegmentOrchestrator(orchConfig, ttsGen)

	if err := orch.Process(ctx); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	fmt.Printf("\n✓ Success! Video saved to: %s\n", bvOutputFile)

	// List additional language versions if generated
	if len(bvLanguages) > 1 {
		fmt.Println("\nLanguage versions:")
		ext := filepath.Ext(bvOutputFile)
		base := bvOutputFile[:len(bvOutputFile)-len(ext)]
		for _, lang := range bvLanguages[1:] {
			langFile := fmt.Sprintf("%s_%s%s", base, lang, ext)
			fmt.Printf("  - %s\n", langFile)
		}
	}

	// List saved audio tracks if audio-dir was specified
	if bvAudioDir != "" {
		fmt.Println("\nAudio tracks saved to:")
		for _, lang := range bvLanguages {
			fmt.Printf("  - %s/combined.mp3\n", filepath.Join(bvAudioDir, lang))
		}
	}

	return nil
}
