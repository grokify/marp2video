package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grokify/mogo/fmt/progress"
	omnistt "github.com/grokify/videoascode/pkg/omnivoice/stt"
	"github.com/grokify/videoascode/pkg/transcript"
	"github.com/grokify/videoascode/pkg/tts"
	"github.com/spf13/cobra"
)

var subtitleCmd = &cobra.Command{
	Use:   "subtitle",
	Short: "Generate subtitles from audio files",
	Long: `Generate subtitle files (SRT/VTT) from audio files using speech-to-text.

Supports Deepgram and ElevenLabs STT providers via OmniVoice.
Deepgram is recommended for STT due to better word-level timestamp accuracy.

This command reads audio files from a locale directory (audio/{lang}/) and generates:
  - Combined subtitle files: subtitles/{lang}.srt and subtitles/{lang}.vtt
  - Individual subtitle files per slide (optional with --individual)

The language is auto-detected from the manifest.json in the audio directory,
or from the directory name if it matches a locale pattern (e.g., "fr-FR").

Output follows the locale-based naming convention for automation:
  subtitles/
  ├── en-US.srt
  ├── en-US.vtt
  ├── fr-FR.srt
  └── fr-FR.vtt

Examples:
  # Generate French subtitles (language auto-detected)
  marp2video subtitle --audio audio/fr-FR/

  # Generate subtitles with explicit language
  marp2video subtitle --audio audio/fr-FR/ --lang fr-FR

  # Generate subtitles to custom output directory
  marp2video subtitle --audio audio/fr-FR/ --output subs/

  # Generate individual slide subtitles in addition to combined
  marp2video subtitle --audio audio/fr-FR/ --individual

  # Use ElevenLabs instead of Deepgram
  marp2video subtitle --audio audio/fr-FR/ --provider elevenlabs`,
	RunE: runSubtitle,
}

var (
	subtitleAudioDir        string
	subtitleOutputDir       string
	subtitleLanguage        string
	subtitleElevenLabsKey   string
	subtitleDeepgramKey     string
	subtitleProvider        string
	subtitleIndividual      bool
	subtitleUseOriginalText bool
	subtitleTranscript      string
	subtitleTimestamps      string
	subtitleDictionaries    []string
	subtitleNoBuiltIn       bool
	subtitleLocal           bool
	subtitleWhisperEndpoint string
)

func init() {
	subtitleCmd.Flags().StringVarP(&subtitleAudioDir, "audio", "a", "", "Audio directory containing manifest.json and slide_*.mp3 (required)")
	subtitleCmd.Flags().StringVarP(&subtitleOutputDir, "output", "o", "subtitles", "Output directory for subtitle files")
	subtitleCmd.Flags().StringVarP(&subtitleLanguage, "lang", "l", "", "Language code (auto-detected from manifest if not specified)")
	subtitleCmd.Flags().StringVar(&subtitleElevenLabsKey, "elevenlabs-api-key", "", "ElevenLabs API key (or use ELEVENLABS_API_KEY env var)")
	subtitleCmd.Flags().StringVar(&subtitleDeepgramKey, "deepgram-api-key", "", "Deepgram API key (or use DEEPGRAM_API_KEY env var)")
	subtitleCmd.Flags().StringVar(&subtitleProvider, "provider", "", "STT provider: deepgram, elevenlabs, or whisper-mlx (default: deepgram if available)")
	subtitleCmd.Flags().BoolVar(&subtitleLocal, "local", false, "Enable local STT providers (Whisper MLX); requires a local gRPC server running on Apple Silicon")
	subtitleCmd.Flags().StringVar(&subtitleWhisperEndpoint, "whisper-endpoint", "", "Whisper MLX gRPC endpoint (default: unix:///tmp/omnivoice-whisper.sock)")
	subtitleCmd.Flags().BoolVar(&subtitleIndividual, "individual", false, "Also generate individual subtitle files per slide")
	subtitleCmd.Flags().BoolVar(&subtitleUseOriginalText, "use-original-text", false, "Use original transcript text with STT timestamps for proper capitalization")
	subtitleCmd.Flags().StringVar(&subtitleTranscript, "transcript", "", "Path to transcript JSON file (required with --use-original-text)")
	subtitleCmd.Flags().StringVar(&subtitleTimestamps, "timestamps", "", "Load timestamps from JSON file instead of calling STT")
	subtitleCmd.Flags().StringArrayVar(&subtitleDictionaries, "dictionary", nil, "Additional dictionary files for case correction (can specify multiple)")
	subtitleCmd.Flags().BoolVar(&subtitleNoBuiltIn, "no-builtin-dictionary", false, "Disable built-in dictionary corrections")

	if err := subtitleCmd.MarkFlagRequired("audio"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(subtitleCmd)
}

func runSubtitle(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	// Validate audio directory exists
	if _, err := os.Stat(subtitleAudioDir); os.IsNotExist(err) {
		return fmt.Errorf("audio directory does not exist: %s", subtitleAudioDir)
	}

	// Validate --use-original-text requires --transcript
	if subtitleUseOriginalText && subtitleTranscript == "" {
		return fmt.Errorf("--transcript is required when using --use-original-text")
	}

	// Load transcript if using original text
	var transcriptData *transcript.Transcript
	if subtitleUseOriginalText {
		if _, err := os.Stat(subtitleTranscript); os.IsNotExist(err) {
			return fmt.Errorf("transcript file does not exist: %s", subtitleTranscript)
		}
		var err error
		transcriptData, err = transcript.LoadFromFile(subtitleTranscript)
		if err != nil {
			return fmt.Errorf("failed to load transcript: %w", err)
		}
	}

	// Check for manifest.json in audio directory
	manifestPath := filepath.Join(subtitleAudioDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest.json not found in audio directory: %s", subtitleAudioDir)
	}

	// Get API keys from flags or environment
	elevenLabsKey := subtitleElevenLabsKey
	if elevenLabsKey == "" {
		elevenLabsKey = os.Getenv("ELEVENLABS_API_KEY")
	}

	deepgramKey := subtitleDeepgramKey
	if deepgramKey == "" {
		deepgramKey = os.Getenv("DEEPGRAM_API_KEY")
	}

	// Check for auto-detected timestamps.json in audio directory
	hasTimestamps := subtitleTimestamps != ""
	if !hasTimestamps {
		autoTimestampsPath := filepath.Join(subtitleAudioDir, "timestamps.json")
		if _, err := os.Stat(autoTimestampsPath); err == nil {
			hasTimestamps = true
		}
	}

	// Require at least one API key (unless using pre-saved timestamps or a local provider)
	if !hasTimestamps && !subtitleLocal && elevenLabsKey == "" && deepgramKey == "" {
		return fmt.Errorf("STT API key required: use --deepgram-api-key or --elevenlabs-api-key flag, set DEEPGRAM_API_KEY or ELEVENLABS_API_KEY env var, or use --local for Whisper MLX\n(or use --timestamps to load pre-saved timestamps)")
	}

	// If Deepgram is not available but ElevenLabs is, prompt user
	// (skipped when using pre-saved timestamps or a local provider)
	if !hasTimestamps && !subtitleLocal && deepgramKey == "" && elevenLabsKey != "" && subtitleProvider == "" {
		fmt.Println("⚠️  DEEPGRAM_API_KEY not found. Deepgram is recommended for subtitle generation")
		fmt.Println("   due to better word-level timestamp accuracy.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  [1] Enter Deepgram API key now")
		fmt.Println("  [2] Continue with ElevenLabs (not recommended)")
		fmt.Println("  [3] Cancel")
		fmt.Println()
		fmt.Print("Choose option [1/2/3]: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Print("Enter Deepgram API key: ")
			keyInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}
			deepgramKey = strings.TrimSpace(keyInput)
			if deepgramKey == "" {
				return fmt.Errorf("no API key provided")
			}
			fmt.Println()
			fmt.Println("💡 Tip: Set DEEPGRAM_API_KEY environment variable to avoid this prompt:")
			fmt.Println("   export DEEPGRAM_API_KEY=\"" + deepgramKey + "\"")
			fmt.Println()
		case "2":
			fmt.Println()
			fmt.Println("Continuing with ElevenLabs STT...")
			fmt.Println()
			subtitleProvider = "elevenlabs"
		case "3", "":
			return fmt.Errorf("cancelled by user")
		default:
			return fmt.Errorf("invalid option: %s", input)
		}
	}

	// Load manifest
	manifest, err := tts.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Determine language (priority: flag > manifest > directory name)
	language := subtitleLanguage
	if language == "" {
		language = manifest.Language
	}
	if language == "" {
		// Try to extract from directory name (e.g., "audio/fr-FR" -> "fr-FR")
		language = detectLanguageFromPath(subtitleAudioDir)
	}
	if language == "" {
		return fmt.Errorf("could not determine language: specify with --lang flag")
	}

	// Create output directory
	if err := os.MkdirAll(subtitleOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Auto-detect timestamps.json if not explicitly provided
	timestampsFile := subtitleTimestamps
	if timestampsFile == "" {
		autoPath := filepath.Join(subtitleAudioDir, "timestamps.json")
		if _, err := os.Stat(autoPath); err == nil {
			timestampsFile = autoPath
			fmt.Printf("Using cached timestamps: %s\n", autoPath)
		}
	}

	fmt.Printf("Generating subtitles from audio\n")
	fmt.Printf("  Audio:    %s\n", subtitleAudioDir)
	fmt.Printf("  Language: %s\n", language)
	fmt.Printf("  Output:   %s\n", subtitleOutputDir)
	fmt.Printf("  Slides:   %d\n\n", len(manifest.Slides))

	// Create generator config
	genConfig := tts.SubtitleGeneratorConfig{
		ProviderConfig: omnistt.ProviderConfig{
			ElevenLabsAPIKey:     elevenLabsKey,
			DeepgramAPIKey:       deepgramKey,
			EnableLocalProviders: subtitleLocal,
			WhisperMLXEndpoint:   subtitleWhisperEndpoint,
		},
		DefaultProvider:     subtitleProvider,
		OutputDir:           subtitleOutputDir,
		AudioDir:            subtitleAudioDir, // For saving timestamps.json
		Format:              tts.FormatSRT,    // We'll generate both formats
		Language:            language,
		UseOriginalText:     subtitleUseOriginalText,
		OriginalTranscript:  transcriptData,
		SaveTimestamps:      true, // Always save timestamps for reuse
		TimestampsFile:      timestampsFile,
		DictionaryPaths:     subtitleDictionaries,
		NoBuiltInDictionary: subtitleNoBuiltIn,
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
	result, err := generator.GenerateFromManifest(ctx, manifest, subtitleAudioDir)

	// Clear progress line if we were showing it
	if renderer != nil {
		renderer.Done("")
	}

	if err != nil {
		return fmt.Errorf("failed to generate subtitles: %w", err)
	}

	// Rename combined files to locale-based naming
	srtPath := filepath.Join(subtitleOutputDir, language+".srt")
	vttPath := filepath.Join(subtitleOutputDir, language+".vtt")

	if result.CombinedSRT != "" {
		oldPath := filepath.Join(subtitleOutputDir, result.CombinedSRT)
		if err := os.Rename(oldPath, srtPath); err != nil {
			return fmt.Errorf("failed to rename SRT file: %w", err)
		}
	}

	if result.CombinedVTT != "" {
		oldPath := filepath.Join(subtitleOutputDir, result.CombinedVTT)
		if err := os.Rename(oldPath, vttPath); err != nil {
			return fmt.Errorf("failed to rename VTT file: %w", err)
		}
	}

	// Remove individual slide subtitles unless --individual flag is set
	if !subtitleIndividual {
		for _, sub := range result.Subtitles {
			slidePath := filepath.Join(subtitleOutputDir, sub.SubtitleFile)
			if err := os.Remove(slidePath); err != nil && !os.IsNotExist(err) {
				// Log but don't fail on cleanup errors
				fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", slidePath, err)
			}
		}
	}

	// Print summary
	fmt.Printf("\n✓ Generated subtitles for %d slides\n", len(result.Subtitles))
	fmt.Printf("✓ SRT: %s\n", srtPath)
	fmt.Printf("✓ VTT: %s\n", vttPath)

	if subtitleIndividual {
		fmt.Printf("✓ Individual slides: %d files\n", len(result.Subtitles))
	}

	// Calculate total words
	totalWords := 0
	for _, sub := range result.Subtitles {
		totalWords += sub.WordCount
	}
	fmt.Printf("✓ Total words transcribed: %d\n", totalWords)

	return nil
}

// detectLanguageFromPath tries to extract a language code from a directory path.
// Looks for common locale patterns like "en-US", "fr-FR", "zh-Hans", etc.
func detectLanguageFromPath(path string) string {
	// Get the last component of the path
	base := filepath.Base(strings.TrimSuffix(path, "/"))

	// Common locale patterns
	// BCP-47: language-region (en-US, fr-FR) or language-script (zh-Hans, zh-Hant)
	if isValidLocale(base) {
		return base
	}

	return ""
}

// isValidLocale checks if a string looks like a valid BCP-47 locale code.
func isValidLocale(s string) bool {
	// Basic validation for common patterns
	// Language codes: 2-3 lowercase letters
	// Region codes: 2 uppercase letters
	// Script codes: 4 letters (first uppercase)

	parts := strings.Split(s, "-")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	// First part should be 2-3 lowercase letters (language)
	lang := parts[0]
	if len(lang) < 2 || len(lang) > 3 {
		return false
	}
	for _, c := range lang {
		if c < 'a' || c > 'z' {
			return false
		}
	}

	// Second part can be region (2 uppercase) or script (4 mixed case)
	second := parts[1]
	if len(second) == 2 {
		// Region code (e.g., "US", "FR")
		for _, c := range second {
			if c < 'A' || c > 'Z' {
				return false
			}
		}
	} else if len(second) == 4 {
		// Script code (e.g., "Hans", "Hant")
		// First letter uppercase, rest lowercase
		if second[0] < 'A' || second[0] > 'Z' {
			return false
		}
		for _, c := range second[1:] {
			if c < 'a' || c > 'z' {
				return false
			}
		}
	} else {
		return false
	}

	return true
}
