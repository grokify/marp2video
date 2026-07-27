package tts

import (
	"os"
	"path/filepath"
	"testing"

	octerm "github.com/plexusone/omnivoice-core/terminology"
)

func TestCaseCorrector_Correct(t *testing.T) {
	dict := &Dictionary{
		Corrections: map[string]string{
			"ai":               "AI",
			"openai":           "OpenAI",
			"claude code":      "Claude Code",
			"frontier workers": "Frontier Workers",
		},
	}

	cc := NewCaseCorrector(dict)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single word",
			input:    "Using ai for coding",
			expected: "Using AI for coding",
		},
		{
			name:     "multiple matches",
			input:    "openai makes ai tools",
			expected: "OpenAI makes AI tools",
		},
		{
			name:     "multi-word phrase",
			input:    "claude code is great",
			expected: "Claude Code is great",
		},
		{
			name:     "preserves other case",
			input:    "The AI from OpenAI",
			expected: "The AI from OpenAI",
		},
		{
			name:     "frontier workers phrase",
			input:    "frontier workers use ai aggressively",
			expected: "Frontier Workers use AI aggressively",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cc.Correct(tt.input)
			if result != tt.expected {
				t.Errorf("Correct(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCaseCorrector_CorrectWord(t *testing.T) {
	dict := &Dictionary{
		Corrections: map[string]string{
			"ai":     "AI",
			"openai": "OpenAI",
		},
	}

	cc := NewCaseCorrector(dict)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "ai", "AI"},
		{"with comma", "ai,", "AI,"},
		{"with period", "openai.", "OpenAI."},
		{"no match", "hello", "hello"},
		{"already correct", "AI", "AI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cc.CorrectWord(tt.input)
			if result != tt.expected {
				t.Errorf("CorrectWord(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuiltInCorrections(t *testing.T) {
	// Verify some key built-in corrections exist, sourced from the
	// terminology-spec-embedded layer via omnivoice-core/terminology.
	builtin, err := octerm.BuiltinCorrections()
	if err != nil {
		t.Fatalf("BuiltinCorrections() error = %v", err)
	}

	expectedCorrections := map[string]string{
		"ai":          "AI",
		"openai":      "OpenAI",
		"claude":      "Claude",
		"claude code": "Claude Code",
		"github":      "GitHub",
		"api":         "API",
		"json":        "JSON",
		"i":           "I",
	}

	for key, expected := range expectedCorrections {
		if actual, ok := builtin[key]; !ok {
			t.Errorf("missing built-in correction for %q", key)
		} else if actual != expected {
			t.Errorf("builtin[%q] = %q, want %q", key, actual, expected)
		}
	}
}

func TestDictionaryLoader_Load(t *testing.T) {
	// Create temp directory with test dictionaries
	tmpDir, err := os.MkdirTemp("", "dict-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Create project directory
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Write config dictionary
	configDict := `{
		"name": "config",
		"corrections": {
			"mycompany": "MyCompany",
			"ai": "A.I."
		}
	}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configDict), 0600); err != nil {
		t.Fatalf("failed to write config dict: %v", err)
	}

	// Write project dictionary (should override)
	projectDict := `{
		"name": "project",
		"corrections": {
			"myproject": "MyProject",
			"ai": "AI"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(projectDict), 0600); err != nil {
		t.Fatalf("failed to write project dict: %v", err)
	}

	// Load with custom directories
	loader := NewDictionaryLoader().
		WithBuiltIn(false).
		WithConfigDir(configDir).
		WithProjectDir(projectDir)

	dict, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Check that config values are loaded
	if dict.Corrections["mycompany"] != "MyCompany" {
		t.Errorf("expected mycompany -> MyCompany, got %q", dict.Corrections["mycompany"])
	}

	// Check that project values are loaded
	if dict.Corrections["myproject"] != "MyProject" {
		t.Errorf("expected myproject -> MyProject, got %q", dict.Corrections["myproject"])
	}

	// Check that project overrides config
	if dict.Corrections["ai"] != "AI" {
		t.Errorf("expected ai -> AI (project override), got %q", dict.Corrections["ai"])
	}
}

func TestDictionaryLoader_WithBuiltIn(t *testing.T) {
	// Test with built-in enabled
	loader := NewDictionaryLoader().
		WithBuiltIn(true).
		WithConfigDir("").
		WithProjectDir("")

	dict, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should have built-in corrections
	if dict.Corrections["ai"] != "AI" {
		t.Errorf("expected ai -> AI from built-in, got %q", dict.Corrections["ai"])
	}

	// Test with built-in disabled
	loader = NewDictionaryLoader().
		WithBuiltIn(false).
		WithConfigDir("").
		WithProjectDir("")

	dict, err = loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should NOT have built-in corrections
	if _, ok := dict.Corrections["ai"]; ok {
		t.Errorf("expected no ai correction when built-in disabled")
	}
}

func TestPreservePunctuation(t *testing.T) {
	tests := []struct {
		original    string
		replacement string
		expected    string
	}{
		{"hello,", "Hello", "Hello,"},
		{"world.", "World", "World."},
		{"test!", "Test", "Test!"},
		{"ai?", "AI", "AI?"},
		{"word", "Word", "Word"},
		{"end...", "End", "End..."},
	}

	for _, tt := range tests {
		t.Run(tt.original, func(t *testing.T) {
			result := preservePunctuation(tt.original, tt.replacement)
			if result != tt.expected {
				t.Errorf("preservePunctuation(%q, %q) = %q, want %q",
					tt.original, tt.replacement, result, tt.expected)
			}
		})
	}
}
