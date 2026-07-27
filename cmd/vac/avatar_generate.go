package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/plexusone/omniavatar"
	_ "github.com/plexusone/omniavatar/providers/all"

	"github.com/grokify/videoascode/pkg/avatar"
)

var avatarGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a talking-head presenter video from narration audio",
	Long: `Generate a talking-head presenter video from narration audio using an
AI avatar provider.

Audio input is one of:
  --manifest   TTS manifest from 'vac slides tts'; slide audio is
               concatenated with pause gaps so lip-sync matches the
               final video timeline
  --audio      a single narration audio file (MP3 recommended)
  --audio-url  a pre-hosted audio URL (required for providers without
               upload support, e.g. tavus)

API keys are read from provider environment variables:
  heygen    HEYGEN_API_KEY
  tavus     TAVUS_API_KEY
  bithuman  BITHUMAN_API_KEY

Generated videos are cached by narration content + avatar configuration;
re-runs with unchanged audio are free.

Examples:
  vac avatar generate --manifest audio/manifest.json --provider heygen \
    --avatar-id <id> --output presenter.mp4

  vac avatar generate --audio narration.mp3 --provider bithuman \
    --avatar-id <agent-id> --output presenter.mp4

  # Provider-specific options via --ext
  vac avatar generate --audio narration.mp3 --provider heygen \
    --avatar-id <id> --ext test=true --output presenter.mp4`,
	RunE: runAvatarGenerate,
}

var (
	agManifest   string
	agAudio      string
	agAudioURL   string
	agProvider   string
	agAvatarID   string
	agAPIKey     string
	agOutput     string
	agPoll       time.Duration
	agCacheDir   string
	agNoCache    bool
	agExtensions []string
)

// providerAPIKeyEnvs maps provider names to their API key environment variables.
// Local providers (liveportrait-joyvasa) have an empty string — no key required.
var providerAPIKeyEnvs = map[string]string{
	"heygen":               "HEYGEN_API_KEY",
	"tavus":                "TAVUS_API_KEY",
	"bithuman":             "BITHUMAN_API_KEY",
	"liveportrait-joyvasa": "", // local provider, no API key
}

// localProviders is the set of providers that run locally and don't need an API key.
var localProviders = map[string]bool{
	"liveportrait-joyvasa": true,
}

func init() {
	avatarGenerateCmd.Flags().StringVarP(&agManifest, "manifest", "m", "", "Audio manifest from 'vac slides tts' (concatenated with pause gaps)")
	avatarGenerateCmd.Flags().StringVar(&agAudio, "audio", "", "Narration audio file (MP3 recommended)")
	avatarGenerateCmd.Flags().StringVar(&agAudioURL, "audio-url", "", "Pre-hosted narration audio URL")
	avatarGenerateCmd.Flags().StringVarP(&agProvider, "provider", "p", "heygen", "Avatar provider: heygen, tavus, bithuman, or liveportrait-joyvasa (local)")
	avatarGenerateCmd.Flags().StringVar(&agAvatarID, "avatar-id", "", "Avatar identity (heygen avatar_id / tavus replica_id / bithuman agent_id) (required)")
	avatarGenerateCmd.Flags().StringVarP(&agAPIKey, "api-key", "k", "", "Provider API key (or use the provider's env var)")
	avatarGenerateCmd.Flags().StringVarP(&agOutput, "output", "o", "presenter.mp4", "Output presenter video file")
	avatarGenerateCmd.Flags().DurationVar(&agPoll, "poll", 5*time.Second, "Job status poll interval")
	avatarGenerateCmd.Flags().StringVar(&agCacheDir, "cache-dir", "", "Presenter video cache directory (default: user cache dir)")
	avatarGenerateCmd.Flags().BoolVar(&agNoCache, "no-cache", false, "Disable presenter video caching")
	avatarGenerateCmd.Flags().StringArrayVar(&agExtensions, "ext", nil, "Provider-specific request option as key=value (repeatable)")

	if err := avatarGenerateCmd.MarkFlagRequired("avatar-id"); err != nil {
		panic(err)
	}

	avatarCmd.AddCommand(avatarGenerateCmd)
}

func runAvatarGenerate(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	inputs := 0
	for _, v := range []string{agManifest, agAudio, agAudioURL} {
		if v != "" {
			inputs++
		}
	}
	if inputs != 1 {
		return fmt.Errorf("exactly one of --manifest, --audio, or --audio-url is required")
	}

	apiKey := agAPIKey
	envVar, ok := providerAPIKeyEnvs[agProvider]
	if !ok {
		return fmt.Errorf("unknown provider %q (available: heygen, tavus, bithuman, liveportrait-joyvasa)", agProvider)
	}
	// Local providers don't require an API key
	if !localProviders[agProvider] {
		if apiKey == "" {
			apiKey = os.Getenv(envVar)
		}
		if apiKey == "" {
			return fmt.Errorf("%s API key required: use --api-key flag or %s env var", agProvider, envVar)
		}
	}

	extensions, err := parseExtensions(agExtensions)
	if err != nil {
		return err
	}

	provider, err := omniavatar.GetRenderProvider(agProvider, omniavatar.WithAPIKey(apiKey))
	if err != nil {
		return err
	}

	audioPath := agAudio
	if agManifest != "" {
		// Concatenate per-slide narration (including pause gaps) so the
		// avatar lip-sync matches the final video timeline.
		audioPath = filepath.Join(os.TempDir(), fmt.Sprintf("vac-narration-%d.mp3", os.Getpid()))
		defer func() {
			// Best-effort temp file cleanup; nothing actionable on failure.
			_ = os.Remove(audioPath) //nolint:errcheck // see above
		}()

		fmt.Printf("Concatenating narration audio from %s...\n", agManifest)
		if err := avatar.ConcatManifestAudio(ctx, agManifest, audioPath); err != nil {
			return err
		}
	}

	cacheDir := ""
	if !agNoCache {
		cacheDir = agCacheDir
		if cacheDir == "" {
			userCache, err := os.UserCacheDir()
			if err != nil {
				return fmt.Errorf("resolve cache dir (use --cache-dir or --no-cache): %w", err)
			}
			cacheDir = filepath.Join(userCache, "vac", "avatar")
		}
	}

	fmt.Printf("Generating presenter video via %s...\n", agProvider)
	if err := avatar.Generate(ctx, agOutput, avatar.GenerateOptions{
		Provider:     provider,
		AvatarID:     agAvatarID,
		AudioPath:    audioPath,
		AudioURL:     agAudioURL,
		Extensions:   extensions,
		PollInterval: agPoll,
		CacheDir:     cacheDir,
	}); err != nil {
		return err
	}

	fmt.Printf("Presenter video written to %s\n", agOutput)
	return nil
}

// parseExtensions converts repeated key=value flags into a provider
// extensions map, coercing bool and integer literals.
func parseExtensions(pairs []string) (map[string]any, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	extensions := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --ext %q: expected key=value", pair)
		}
		switch {
		case value == "true" || value == "false":
			extensions[key] = value == "true"
		default:
			if n, err := strconv.Atoi(value); err == nil {
				extensions[key] = n
			} else {
				extensions[key] = value
			}
		}
	}
	return extensions, nil
}
