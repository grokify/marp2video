package avatar

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/plexusone/omniavatar-core/render"
)

// fakeProvider is a render.Provider with optional upload support.
type fakeProvider struct {
	uploads   int
	generates int
	video     []byte
	canUpload bool
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Generate(_ context.Context, req render.GenerateRequest) (*render.Job, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	f.generates++
	return &render.Job{ID: "job-1", Provider: "fake"}, nil
}

func (f *fakeProvider) Status(_ context.Context, jobID string) (*render.JobStatus, error) {
	return &render.JobStatus{ID: jobID, State: render.JobStateCompleted, VideoURL: "https://x/v.mp4"}, nil
}

func (f *fakeProvider) Download(_ context.Context, _ string, dst io.Writer) error {
	_, err := dst.Write(f.video)
	return err
}

// fakeUploader adds AudioUploader to fakeProvider.
type fakeUploader struct{ *fakeProvider }

func (f *fakeUploader) UploadAudio(_ context.Context, _ string, r io.Reader) (string, error) {
	if _, err := io.ReadAll(r); err != nil {
		return "", err
	}
	f.uploads++
	return "https://x/narration.mp3", nil
}

func writeTempAudio(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "narration.mp3")
	if err := os.WriteFile(path, []byte("fake-audio"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGenerateWithUploadAndCache(t *testing.T) {
	provider := &fakeUploader{&fakeProvider{video: []byte("fake-mp4"), canUpload: true}}
	audioPath := writeTempAudio(t)
	cacheDir := t.TempDir()
	outputPath := filepath.Join(t.TempDir(), "presenter.mp4")

	opts := GenerateOptions{
		Provider:     provider,
		AvatarID:     "avatar-1",
		AudioPath:    audioPath,
		PollInterval: time.Millisecond,
		CacheDir:     cacheDir,
	}

	if err := Generate(context.Background(), outputPath, opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fake-mp4" {
		t.Errorf("output content = %q, want %q", got, "fake-mp4")
	}
	if provider.uploads != 1 || provider.generates != 1 {
		t.Errorf("uploads=%d generates=%d, want 1/1", provider.uploads, provider.generates)
	}

	// Second run with identical audio + config must be served from cache.
	outputPath2 := filepath.Join(t.TempDir(), "presenter2.mp4")
	if err := Generate(context.Background(), outputPath2, opts); err != nil {
		t.Fatalf("Generate() cached error = %v", err)
	}
	if provider.uploads != 1 || provider.generates != 1 {
		t.Errorf("cache miss: uploads=%d generates=%d, want 1/1", provider.uploads, provider.generates)
	}
	got2, err := os.ReadFile(outputPath2)
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != "fake-mp4" {
		t.Errorf("cached output content = %q, want %q", got2, "fake-mp4")
	}
}

func TestGenerateLocalAudioWithoutUploader(t *testing.T) {
	provider := &fakeProvider{video: []byte("x")}
	err := Generate(context.Background(), filepath.Join(t.TempDir(), "p.mp4"), GenerateOptions{
		Provider:  provider,
		AvatarID:  "avatar-1",
		AudioPath: writeTempAudio(t),
	})
	if !errors.Is(err, render.ErrAudioUploadUnsupported) {
		t.Fatalf("Generate() error = %v, want errors.Is ErrAudioUploadUnsupported", err)
	}
}

func TestGenerateWithAudioURL(t *testing.T) {
	// A provider without upload support works fine with a hosted URL.
	provider := &fakeProvider{video: []byte("x")}
	outputPath := filepath.Join(t.TempDir(), "p.mp4")

	err := Generate(context.Background(), outputPath, GenerateOptions{
		Provider: provider,
		AvatarID: "avatar-1",
		AudioURL: "https://example.com/narration.mp3",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if provider.generates != 1 {
		t.Errorf("generates = %d, want 1", provider.generates)
	}
}

func TestGenerateValidation(t *testing.T) {
	provider := &fakeProvider{}
	tests := []struct {
		name string
		opts GenerateOptions
	}{
		{"missing provider", GenerateOptions{AvatarID: "a", AudioURL: "https://x/a.mp3"}},
		{"missing avatar", GenerateOptions{Provider: provider, AudioURL: "https://x/a.mp3"}},
		{"no audio", GenerateOptions{Provider: provider, AvatarID: "a"}},
		{"both audio inputs", GenerateOptions{Provider: provider, AvatarID: "a", AudioPath: "x.mp3", AudioURL: "https://x/a.mp3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Generate(context.Background(), "out.mp4", tt.opts); err == nil {
				t.Error("Generate() error = nil, want validation error")
			}
		})
	}
}

func TestCacheKeyChangesWithConfig(t *testing.T) {
	provider := &fakeProvider{}
	audioPath := writeTempAudio(t)

	base := GenerateOptions{Provider: provider, AvatarID: "a1", AudioPath: audioPath}
	key1, err := cacheKey(base)
	if err != nil {
		t.Fatal(err)
	}

	changed := base
	changed.AvatarID = "a2"
	key2, err := cacheKey(changed)
	if err != nil {
		t.Fatal(err)
	}
	if key1 == key2 {
		t.Error("cacheKey identical for different avatar IDs")
	}

	withExt := base
	withExt.Extensions = map[string]any{"test": true}
	key3, err := cacheKey(withExt)
	if err != nil {
		t.Fatal(err)
	}
	if key1 == key3 {
		t.Error("cacheKey identical for different extensions")
	}
}
