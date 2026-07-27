package tts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	octerm "github.com/plexusone/omnivoice-core/terminology"
)

// Dictionary contains case corrections for subtitle text.
type Dictionary struct {
	Name        string            `json:"name,omitempty"`
	Version     string            `json:"version,omitempty"`
	Corrections map[string]string `json:"corrections"`
}

// DictionaryLoader handles loading and merging multiple dictionaries.
type DictionaryLoader struct {
	configDir       string
	projectDir      string
	additionalPaths []string
	includeBuiltIn  bool
}

// NewDictionaryLoader creates a new dictionary loader.
func NewDictionaryLoader() *DictionaryLoader {
	// Default config directory
	configDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(home, ".config", "videoascode", "dictionaries")
	}

	return &DictionaryLoader{
		configDir:      configDir,
		projectDir:     "./dictionaries",
		includeBuiltIn: true,
	}
}

// WithConfigDir sets a custom config directory.
func (dl *DictionaryLoader) WithConfigDir(dir string) *DictionaryLoader {
	dl.configDir = dir
	return dl
}

// WithProjectDir sets the project dictionary directory.
func (dl *DictionaryLoader) WithProjectDir(dir string) *DictionaryLoader {
	dl.projectDir = dir
	return dl
}

// WithAdditionalPaths adds extra dictionary paths.
func (dl *DictionaryLoader) WithAdditionalPaths(paths []string) *DictionaryLoader {
	dl.additionalPaths = paths
	return dl
}

// WithBuiltIn controls whether to include built-in corrections.
func (dl *DictionaryLoader) WithBuiltIn(include bool) *DictionaryLoader {
	dl.includeBuiltIn = include
	return dl
}

// Load loads and merges all dictionaries in order.
// Order: built-in → user config → additional paths → project local
func (dl *DictionaryLoader) Load() (*Dictionary, error) {
	merged := &Dictionary{
		Name:        "merged",
		Corrections: make(map[string]string),
	}

	// 1. Built-in corrections — the generic industry-technology term layer
	// embedded in github.com/plexusone/terminology-spec (AI/ML terms,
	// companies, dev tools, languages, frameworks, cloud/infra, etc.),
	// available as a Go dependency rather than a hardcoded local map.
	if dl.includeBuiltIn {
		builtin, err := octerm.BuiltinCorrections()
		if err != nil {
			return nil, fmt.Errorf("load builtin terminology: %w", err)
		}
		for k, v := range builtin {
			merged.Corrections[k] = v
		}
	}

	// 2. User config directory (~/.config/videoascode/dictionaries/*.json)
	if dl.configDir != "" {
		if err := dl.loadFromDir(dl.configDir, merged); err != nil {
			// Don't fail if config dir doesn't exist
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load config dictionaries: %w", err)
			}
		}
	}

	// 3. Additional paths (--dictionary flags)
	for _, path := range dl.additionalPaths {
		if err := dl.loadFromPath(path, merged); err != nil {
			return nil, fmt.Errorf("failed to load dictionary %s: %w", path, err)
		}
	}

	// 4. Project local directory (./dictionaries/*.json)
	if dl.projectDir != "" {
		if err := dl.loadFromDir(dl.projectDir, merged); err != nil {
			// Don't fail if project dir doesn't exist
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load project dictionaries: %w", err)
			}
		}
	}

	return merged, nil
}

// loadFromDir loads all .json files from a directory.
func (dl *DictionaryLoader) loadFromDir(dir string, merged *Dictionary) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Sort entries to ensure consistent order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := dl.loadFromPath(path, merged); err != nil {
			return err
		}
	}

	return nil
}

// loadFromPath loads a single dictionary file and merges it.
func (dl *DictionaryLoader) loadFromPath(path string, merged *Dictionary) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var dict Dictionary
	if err := json.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	// Merge corrections (later values override earlier)
	for k, v := range dict.Corrections {
		// Normalize key to lowercase for consistent lookup
		merged.Corrections[strings.ToLower(k)] = v
	}

	return nil
}

// CaseCorrector applies dictionary-based case corrections to text. The bulk
// regex substitution engine lives in
// github.com/plexusone/omnivoice-core/terminology (backed by the canonical
// term IR in github.com/plexusone/terminology-spec); this type adapts a
// videoascode Dictionary (flat lowercase -> corrected-form map, as loaded by
// DictionaryLoader) into that engine, and keeps the exact-word lookup
// (CorrectWord) that needs the flat map directly for punctuation handling.
type CaseCorrector struct {
	dictionary *Dictionary
	inner      *octerm.CaseCorrector
}

// NewCaseCorrector creates a new case corrector from a dictionary.
func NewCaseCorrector(dict *Dictionary) *CaseCorrector {
	terms := make([]octerm.Term, 0, len(dict.Corrections))
	for original, replacement := range dict.Corrections {
		terms = append(terms, octerm.Term{CanonicalForm: replacement, Aliases: []string{original}})
	}
	return &CaseCorrector{
		dictionary: dict,
		inner:      octerm.NewCaseCorrector(terms),
	}
}

// Correct applies all dictionary corrections to the text.
func (cc *CaseCorrector) Correct(text string) string {
	return cc.inner.Correct(text)
}

// CorrectWord corrects a single word using the dictionary.
func (cc *CaseCorrector) CorrectWord(word string) string {
	lower := strings.ToLower(word)

	// Check for exact match
	if replacement, ok := cc.dictionary.Corrections[lower]; ok {
		// Preserve trailing punctuation
		return preservePunctuation(word, replacement)
	}

	// Check without punctuation
	stripped := stripTrailingPunctuation(lower)
	if replacement, ok := cc.dictionary.Corrections[stripped]; ok {
		return preservePunctuation(word, replacement)
	}

	return word
}

// stripTrailingPunctuation removes trailing punctuation from a word.
func stripTrailingPunctuation(word string) string {
	return strings.TrimRight(word, ".,!?;:'\"")
}

// preservePunctuation applies the replacement but keeps original punctuation.
func preservePunctuation(original, replacement string) string {
	// Find trailing punctuation in original
	trailing := ""
	for i := len(original) - 1; i >= 0; i-- {
		c := original[i]
		if c == '.' || c == ',' || c == '!' || c == '?' || c == ';' || c == ':' || c == '\'' || c == '"' {
			trailing = string(c) + trailing
		} else {
			break
		}
	}
	return replacement + trailing
}
