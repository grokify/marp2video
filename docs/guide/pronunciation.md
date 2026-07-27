# Pronunciation

TTS engines often mispronounce brand names, acronyms, and technical terms
("AAuth", "kubectl", "PlexusOne"). vac lets you supply a **pronunciation
dictionary** that rewrites those terms to a spoken form **before** synthesis —
so the generated audio says them correctly — while leaving the original spelling
untouched everywhere the text is *displayed* (subtitles, manifests, captions).

This is the opposite direction from [subtitle case correction](subtitles.md):

- **Pronunciation** rewrites text **before TTS** so the *audio* sounds right.
- **Case correction** fixes *subtitle text* **after STT** so the *display* looks right.

The two are independent and can be used together.

## How it works

Define a `pronunciations` map in your config: each term maps a BCP-47 language
to how it should be spoken. Terms with no entry for the active language are left
unchanged.

```yaml
pronunciations:
  AAuth:
    en-US: "ay auth"
  kubectl:
    en-US: "cube control"
  PlexusOne:
    en-US: "plexus one"
    fr-FR: "plexusse une"
```

Or in JSON:

```json
{
  "pronunciations": {
    "AAuth":     { "en-US": "ay auth" },
    "kubectl":   { "en-US": "cube control" },
    "PlexusOne": { "en-US": "plexus one", "fr-FR": "plexusse une" }
  }
}
```

When the config declares any pronunciations, vac applies them to the voiceover
text sent to the TTS provider (any provider). Matching is **case-insensitive
with word boundaries**, and **multi-word terms are matched before shorter
overlapping ones**, so a longer term isn't shadowed by a shorter one nested
inside it.

!!! important "Only the audio changes"
    Substitutions affect **only** the text handed to the TTS engine. The
    original term is preserved in voiceover text, timing-based subtitles,
    manifests, and captions — so what viewers *read* stays correct even though
    what they *hear* is the phonetic form.

## Where it applies

Pronunciations are read from the config used by `vac browser video`
(`VideoConfig.Pronunciations`). Any segment narrated through that pipeline has
its TTS input rewritten per the active language before synthesis.

## Canonical terms (advanced)

The substitution engine lives in
[`omnivoice-core/terminology`](https://github.com/plexusone/omnivoice-core), backed
by the canonical term IR in `terminology-spec`. Inline `pronunciations` config is
the convenient shape; projects with a canonical `terms/` directory can instead
compile a pronouncer directly from those terms (`NewPronouncerFromTerms`),
keeping pronunciation guidance in one shared source of truth across tools.

## See also

- [Voiceover Formats](voiceover-formats.md) — how narration text is authored
- [Subtitle Generation](subtitles.md) — the display-side counterpart (case correction)
