package stt

import (
	"fmt"

	"github.com/plexusone/omnivoice"
	_ "github.com/plexusone/omnivoice-core/providers/whisper-mlx" // Register Whisper MLX local provider
	_ "github.com/plexusone/omnivoice/providers/all"              // Register all cloud providers
)

// ProviderConfig holds configuration for creating STT providers.
type ProviderConfig struct {
	// ElevenLabsAPIKey is the API key for ElevenLabs.
	ElevenLabsAPIKey string

	// DeepgramAPIKey is the API key for Deepgram.
	DeepgramAPIKey string

	// WhisperMLXEndpoint is the gRPC endpoint for the Whisper MLX local server.
	// Default: "unix:///tmp/omnivoice-whisper.sock"
	WhisperMLXEndpoint string

	// EnableLocalProviders enables local STT providers (Whisper MLX, etc).
	// Requires the corresponding Python/MLX gRPC server to be running locally
	// (Apple Silicon only). See providers/whisper-mlx/server in omnivoice-core.
	EnableLocalProviders bool
}

// Factory creates STT providers based on configuration.
type Factory struct {
	config    ProviderConfig
	providers map[string]*Provider
	fallback  string
}

// NewFactory creates a new STT provider factory.
func NewFactory(config ProviderConfig) *Factory {
	return &Factory{
		config:    config,
		providers: make(map[string]*Provider),
	}
}

// Get returns a provider by name, creating it if necessary.
// If name is empty, returns the fallback provider.
func (f *Factory) Get(name string) (*Provider, error) {
	if name == "" {
		name = f.fallback
	}
	if name == "" {
		// Default to deepgram for STT if available (better batch support)
		if f.config.DeepgramAPIKey != "" {
			name = "deepgram"
		} else if f.config.ElevenLabsAPIKey != "" {
			name = "elevenlabs"
		} else if f.config.EnableLocalProviders {
			name = "whisper-mlx"
		} else {
			return nil, fmt.Errorf("no provider specified and no API keys or local providers configured")
		}
	}

	// Return cached provider if available
	if provider, ok := f.providers[name]; ok {
		return provider, nil
	}

	// Create provider
	provider, err := f.createProvider(name)
	if err != nil {
		return nil, err
	}

	f.providers[name] = provider
	if f.fallback == "" {
		f.fallback = name
	}

	return provider, nil
}

// SetFallback sets the default provider name.
func (f *Factory) SetFallback(name string) {
	f.fallback = name
}

// createProvider creates a new provider instance using the omnivoice registry.
func (f *Factory) createProvider(name string) (*Provider, error) {
	var opts []omnivoice.ProviderOption

	switch name {
	case "elevenlabs":
		if f.config.ElevenLabsAPIKey == "" {
			return nil, fmt.Errorf("ElevenLabs API key not configured")
		}
		opts = append(opts, omnivoice.WithAPIKey(f.config.ElevenLabsAPIKey))

	case "deepgram":
		if f.config.DeepgramAPIKey == "" {
			return nil, fmt.Errorf("Deepgram API key not configured")
		}
		opts = append(opts, omnivoice.WithAPIKey(f.config.DeepgramAPIKey))

	case "whisper-mlx":
		if !f.config.EnableLocalProviders {
			return nil, fmt.Errorf("local providers not enabled")
		}
		if f.config.WhisperMLXEndpoint != "" {
			opts = append(opts, omnivoice.WithEndpoint(f.config.WhisperMLXEndpoint))
		}

	default:
		return nil, fmt.Errorf("unknown STT provider: %s", name)
	}

	// Use the omnivoice registry to create the provider
	provider, err := omnivoice.GetSTTProvider(name, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s STT provider: %w", name, err)
	}

	return New(provider), nil
}

// Available returns a list of available provider names based on configured API keys.
func (f *Factory) Available() []string {
	var names []string
	if f.config.ElevenLabsAPIKey != "" {
		names = append(names, "elevenlabs")
	}
	if f.config.DeepgramAPIKey != "" {
		names = append(names, "deepgram")
	}
	if f.config.EnableLocalProviders {
		names = append(names, "whisper-mlx")
	}
	return names
}
