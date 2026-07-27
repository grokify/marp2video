package tts

import (
	octerm "github.com/plexusone/omnivoice-core/terminology"
)

// PronunciationDictionary maps terms to how they should be SPOKEN by TTS,
// keyed by BCP-47 language (e.g. {"AAuth": {"en-US": "ay auth"}}).
//
// This is distinct from Dictionary/CaseCorrector, which fixes subtitle TEXT
// casing after STT transcription. A PronunciationDictionary is applied to
// text BEFORE synthesis, so the generated audio says a term correctly; the
// original spelling is left untouched everywhere else (timing-based
// subtitles, manifests, captions) so displayed text stays correct even
// though the audio pronounces it phonetically.
//
// The substitution engine itself lives in
// github.com/plexusone/omnivoice-core/terminology (backed by the canonical
// term IR in github.com/plexusone/terminology-spec) — this type is just the
// inline-JSON-config shape videoascode's VideoConfig.Pronunciations already
// uses. Prefer loading terms from terminology-spec via
// NewPronouncerFromTerms when a project has a canonical terms/ directory.
type PronunciationDictionary struct {
	Terms map[string]map[string]string `json:"terms"`
}

// Pronouncer applies compiled pronunciation substitutions to TTS input text.
// A nil *Pronouncer is valid and Apply is a no-op, so callers don't need to
// nil-check before use.
type Pronouncer struct {
	inner *octerm.Pronouncer
}

// NewPronouncer compiles a dictionary into a Pronouncer. Multi-word terms are
// matched first, so a longer term isn't shadowed by a shorter one nested
// inside it.
func NewPronouncer(dict *PronunciationDictionary) *Pronouncer {
	if dict == nil {
		return &Pronouncer{}
	}
	return &Pronouncer{inner: octerm.NewPronouncer(dictionaryToTerms(dict))}
}

// NewPronouncerFromTerms builds a Pronouncer directly from canonical
// terminology-spec terms (e.g. loaded via octerm.LoadDir), skipping the
// PronunciationDictionary/VideoConfig JSON shape entirely.
func NewPronouncerFromTerms(terms []octerm.Term) *Pronouncer {
	return &Pronouncer{inner: octerm.NewPronouncer(terms)}
}

// Apply substitutes pronunciation-guide terms in text for the given
// language. Terms with no entry for language are left unmodified. Callers
// should feed the result only to TTS synthesis — keep the original text for
// subtitles/captions/manifests, since the substitution reflects how a term
// sounds, not how it's spelled.
func (p *Pronouncer) Apply(text, language string) string {
	if p == nil {
		return text
	}
	return p.inner.Apply(text, language)
}

func dictionaryToTerms(dict *PronunciationDictionary) []octerm.Term {
	terms := make([]octerm.Term, 0, len(dict.Terms))
	for term, byLang := range dict.Terms {
		terms = append(terms, octerm.Term{CanonicalForm: term, Pronunciations: byLang})
	}
	return terms
}
