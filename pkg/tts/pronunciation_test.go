package tts

import "testing"

func TestPronouncerApply(t *testing.T) {
	dict := &PronunciationDictionary{Terms: map[string]map[string]string{
		"AAuth": {"en-US": "ay auth"},
		"OAuth": {"en-US": "oh auth"},
	}}
	p := NewPronouncer(dict)

	got := p.Apply("AAuth fixes what OAuth breaks for agents.", "en-US")
	want := "ay auth fixes what oh auth breaks for agents."
	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}
}

func TestPronouncerApply_CaseInsensitiveAndWordBoundary(t *testing.T) {
	dict := &PronunciationDictionary{Terms: map[string]map[string]string{
		"AAuth": {"en-US": "ay auth"},
	}}
	p := NewPronouncer(dict)

	// Matches regardless of case...
	if got := p.Apply("aauth and AAUTH and AAuth", "en-US"); got != "ay auth and ay auth and ay auth" {
		t.Errorf("case-insensitive Apply() = %q", got)
	}
	// ...but not as a substring of a longer word.
	if got := p.Apply("AAuthentication is unrelated", "en-US"); got != "AAuthentication is unrelated" {
		t.Errorf("word-boundary Apply() = %q, want unchanged", got)
	}
}

func TestPronouncerApply_NoEntryForLanguage(t *testing.T) {
	dict := &PronunciationDictionary{Terms: map[string]map[string]string{
		"AAuth": {"en-US": "ay auth"},
	}}
	p := NewPronouncer(dict)

	got := p.Apply("AAuth fixes agents.", "fr-FR")
	if got != "AAuth fixes agents." {
		t.Errorf("Apply() with no fr-FR entry = %q, want unchanged", got)
	}
}

func TestPronouncerApply_LongerTermsMatchFirst(t *testing.T) {
	// "AAuth" is itself a valid, word-bounded match inside "AAuth Protocol",
	// so whichever rule runs first wins the overlapping span. Multi-word
	// terms must be tried before their single-word prefixes/substrings.
	dict := &PronunciationDictionary{Terms: map[string]map[string]string{
		"AAuth":          {"en-US": "SHOULD-NOT-WIN"},
		"AAuth Protocol": {"en-US": "ay auth protocol"},
	}}
	p := NewPronouncer(dict)

	got := p.Apply("The AAuth Protocol works well.", "en-US")
	want := "The ay auth protocol works well."
	if got != want {
		t.Errorf("Apply() = %q, want %q (multi-word term should win over a shorter overlapping match)", got, want)
	}
}

func TestPronouncerApply_NilReceiver(t *testing.T) {
	var p *Pronouncer
	if got := p.Apply("unchanged", "en-US"); got != "unchanged" {
		t.Errorf("nil *Pronouncer Apply() = %q, want unchanged", got)
	}
}

func TestNewPronouncer_NilDictionary(t *testing.T) {
	p := NewPronouncer(nil)
	if got := p.Apply("unchanged", "en-US"); got != "unchanged" {
		t.Errorf("NewPronouncer(nil) Apply() = %q, want unchanged", got)
	}
}
