package main

import (
	"github.com/spf13/cobra"
)

var avatarCmd = &cobra.Command{
	Use:   "avatar",
	Short: "Generate and composite talking-head presenter overlays (optional)",
	Long: `Generate a talking-head presenter video from narration audio using an
AI avatar provider (HeyGen, Tavus, or bitHuman via OmniAvatar), and
composite it as a circular overlay onto a slides video.

This feature is optional: presentations render exactly as before unless
these commands are used.

Typical workflow:

  1. vac avatar list-avatars - find a provider avatar ID (--avatar-id)
  2. vac slides tts          - generate per-slide narration + manifest
  3. vac avatar generate     - narration audio -> presenter.mp4
  4. vac slides video        - render slides.mp4
  5. vac avatar compose      - overlay presenter circle onto slides.mp4

Note: --avatar-id must be a provider-native avatar ID. For HeyGen this is
a v2 avatar ID (e.g. Abigail_expressive_2024112501) — run
'vac avatar list-avatars' to find one.

Examples:
  # Generate presenter video from a TTS manifest (concatenates slide audio)
  vac avatar generate --manifest audio/manifest.json --provider heygen \
    --avatar-id <id> --output presenter.mp4

  # Composite the presenter circle onto the slides video
  vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
    --audio narration.mp3 --output final.mp4`,
}

func init() {
	rootCmd.AddCommand(avatarCmd)
}
