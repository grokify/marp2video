package avatar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/grokify/mogo/log/slogutil"
	"github.com/plexusone/omniavatar-core/render"
)

// GenerateOptions configures presenter video generation.
type GenerateOptions struct {
	// Provider is the OmniAvatar render provider to use.
	// Required.
	Provider render.Provider

	// AvatarID identifies the presenter with the provider
	// (heygen avatar_id / tavus replica_id / bithuman agent_id).
	// Required.
	AvatarID string

	// AudioPath is a local narration audio file. It is uploaded via the
	// provider's audio upload capability. Exactly one of AudioPath or
	// AudioURL must be set.
	AudioPath string

	// AudioURL is a pre-hosted narration audio URL, for providers
	// without upload support or externally hosted audio.
	AudioURL string

	// Extensions holds provider-specific request options
	// (e.g., "test", "avatar_style", "voice_id").
	Extensions map[string]any

	// PollInterval is the job status poll interval.
	// Default: 5s.
	PollInterval time.Duration

	// CacheDir enables caching of generated presenter videos keyed by
	// the narration audio content and avatar configuration. Empty
	// disables caching.
	CacheDir string
}

// Generate produces a talking-head presenter video at outputPath from
// narration audio, using an OmniAvatar render provider. Generated videos
// are cached by (audio content, provider, avatar config) when CacheDir
// is set, mirroring vac's TTS caching behavior.
func Generate(ctx context.Context, outputPath string, opts GenerateOptions) error {
	logger := slogutil.LoggerFromContext(ctx, slogutil.Null())

	if opts.Provider == nil {
		return fmt.Errorf("avatar: Provider is required")
	}
	if opts.AvatarID == "" {
		return fmt.Errorf("avatar: AvatarID is required")
	}
	if (opts.AudioPath == "") == (opts.AudioURL == "") {
		return fmt.Errorf("avatar: exactly one of AudioPath or AudioURL is required")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 5 * time.Second
	}

	// Check the cache before any provider calls.
	cachePath := ""
	if opts.CacheDir != "" {
		key, err := cacheKey(opts)
		if err != nil {
			return err
		}
		cachePath = filepath.Join(opts.CacheDir, key+".mp4")
		if _, err := os.Stat(cachePath); err == nil {
			logger.Info("avatar cache hit", "cache", cachePath)
			return copyFile(cachePath, outputPath)
		}
	}

	audioURL := opts.AudioURL
	if audioURL == "" {
		uploader, ok := opts.Provider.(render.AudioUploader)
		if !ok {
			return fmt.Errorf("%w: provider %s cannot host local files; supply a hosted audio URL",
				render.ErrAudioUploadUnsupported, opts.Provider.Name())
		}

		f, err := os.Open(opts.AudioPath)
		if err != nil {
			return err
		}
		logger.Info("uploading narration audio", "provider", opts.Provider.Name(), "audio", opts.AudioPath)
		audioURL, err = uploader.UploadAudio(ctx, filepath.Base(opts.AudioPath), f)
		// Read-only file; a close error after a successful upload is unactionable.
		_ = f.Close() //nolint:errcheck // see above
		if err != nil {
			return err
		}
	}

	logger.Info("submitting avatar generation job", "provider", opts.Provider.Name(), "avatar", opts.AvatarID)
	job, err := opts.Provider.Generate(ctx, render.GenerateRequest{
		AvatarID:   opts.AvatarID,
		AudioURL:   audioURL,
		Extensions: opts.Extensions,
	})
	if err != nil {
		return err
	}

	logger.Info("waiting for avatar video", "job", job.ID, "interval", opts.PollInterval)
	status, err := render.Wait(ctx, opts.Provider, job.ID, opts.PollInterval)
	if err != nil {
		return err
	}
	logger.Info("avatar video ready", "job", job.ID, "duration", status.Duration)

	if err := download(ctx, opts.Provider, job.ID, outputPath); err != nil {
		return err
	}

	if cachePath != "" {
		if err := os.MkdirAll(opts.CacheDir, 0750); err != nil {
			return err
		}
		if err := copyFile(outputPath, cachePath); err != nil {
			return err
		}
		logger.Info("cached avatar video", "cache", cachePath)
	}
	return nil
}

// cacheKey derives a stable cache key from the narration audio content
// and the avatar configuration.
func cacheKey(opts GenerateOptions) (string, error) {
	h := sha256.New()

	if opts.AudioPath != "" {
		f, err := os.Open(opts.AudioPath)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		// Read-only file; a close error after a successful read is unactionable.
		_ = f.Close() //nolint:errcheck // see above
		if err != nil {
			return "", err
		}
	} else {
		fmt.Fprintf(h, "url:%s", opts.AudioURL)
	}

	fmt.Fprintf(h, "|provider:%s|avatar:%s", opts.Provider.Name(), opts.AvatarID)

	keys := make([]string, 0, len(opts.Extensions))
	for k := range opts.Extensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "|%s:%v", k, opts.Extensions[k])
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// download streams the completed job video to outputPath.
func download(ctx context.Context, provider render.Provider, jobID, outputPath string) (err error) {
	f, err := os.Create(outputPath) //nolint:gosec // G304: operator-supplied output path
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return provider.Download(ctx, jobID, f)
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) (err error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return err
	}

	in, err := os.Open(src) //nolint:gosec // G304: operator-supplied path
	if err != nil {
		return err
	}
	defer func() {
		// Read-only file; a close error after a successful copy is unactionable.
		_ = in.Close() //nolint:errcheck // see above
	}()

	out, err := os.Create(dst) //nolint:gosec // G304: operator-supplied path
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
