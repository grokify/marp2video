package tts

import (
	"fmt"

	"github.com/plexusone/omnivoice"
	_ "github.com/plexusone/omnivoice-core/providers/f5tts-mlx" // Register F5-TTS MLX local provider
	_ "github.com/plexusone/omnivoice/providers/all"            // Register all cloud providers
)

// ProviderConfig holds configuration for creating TTS providers.
type ProviderConfig struct {
	// ElevenLabsAPIKey is the API key for ElevenLabs.
	ElevenLabsAPIKey string

	// DeepgramAPIKey is the API key for Deepgram.
	DeepgramAPIKey string

	// F5TTSMLXEndpoint is the gRPC endpoint for the F5-TTS MLX local server.
	// Default: "unix:///tmp/omnivoice-f5tts.sock"
	F5TTSMLXEndpoint string

	// EnableLocalProviders enables local TTS providers (F5-TTS MLX, etc).
	// Requires the corresponding Python/MLX gRPC server to be running locally
	// (Apple Silicon only). See providers/f5tts-mlx/server in omnivoice-core.
	EnableLocalProviders bool
}

// Factory creates TTS providers based on configuration.
type Factory struct {
	config    ProviderConfig
	providers map[string]*Provider
	fallback  string
}

// NewFactory creates a new TTS provider factory.
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
		// Default to elevenlabs if available, then other cloud providers, then local
		if f.config.ElevenLabsAPIKey != "" {
			name = "elevenlabs"
		} else if f.config.DeepgramAPIKey != "" {
			name = "deepgram"
		} else if f.config.EnableLocalProviders {
			name = "f5tts-mlx"
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

	case "f5tts-mlx":
		if !f.config.EnableLocalProviders {
			return nil, fmt.Errorf("local providers not enabled")
		}
		if f.config.F5TTSMLXEndpoint != "" {
			opts = append(opts, omnivoice.WithEndpoint(f.config.F5TTSMLXEndpoint))
		}

	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	// Use the omnivoice registry to create the provider
	provider, err := omnivoice.GetTTSProvider(name, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s provider: %w", name, err)
	}

	return New(provider), nil
}

// Available returns a list of available provider names based on configuration.
func (f *Factory) Available() []string {
	var names []string
	if f.config.ElevenLabsAPIKey != "" {
		names = append(names, "elevenlabs")
	}
	if f.config.DeepgramAPIKey != "" {
		names = append(names, "deepgram")
	}
	if f.config.EnableLocalProviders {
		names = append(names, "f5tts-mlx")
	}
	return names
}
