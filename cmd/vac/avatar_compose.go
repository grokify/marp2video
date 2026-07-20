package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grokify/videoascode/pkg/video"
)

var avatarComposeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Composite a presenter circle overlay onto a slides video",
	Long: `Composite a talking-head presenter video as a circular overlay onto a
slides video.

The presenter video is center-cropped to a square, scaled to the circle
diameter, circle-masked, and anchored at the configured corner. The
presenter video's own audio track is always discarded; pass --audio to
use the narration file as the authoritative audio track, otherwise the
slides video's audio is preserved.

Examples:
  vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
    --audio narration.mp3 --output final.mp4

  # Smaller circle in the bottom-left with a border
  vac avatar compose --slides slides.mp4 --avatar presenter.mp4 \
    --output final.mp4 --diameter 280 --position bottom-left --border 6`,
	RunE: runAvatarCompose,
}

var (
	acSlides      string
	acAvatar      string
	acAudio       string
	acOutput      string
	acDiameter    int
	acPosition    string
	acMarginX     int
	acMarginY     int
	acBorder      int
	acBorderColor string
)

func init() {
	avatarComposeCmd.Flags().StringVar(&acSlides, "slides", "", "Slides (base) video file (required)")
	avatarComposeCmd.Flags().StringVar(&acAvatar, "avatar", "", "Presenter (avatar) video file (required)")
	avatarComposeCmd.Flags().StringVar(&acAudio, "audio", "", "Narration audio file to use as the authoritative audio track")
	avatarComposeCmd.Flags().StringVarP(&acOutput, "output", "o", "final.mp4", "Output video file")
	avatarComposeCmd.Flags().IntVar(&acDiameter, "diameter", 320, "Avatar circle diameter in pixels")
	avatarComposeCmd.Flags().StringVar(&acPosition, "position", video.PositionBottomRight, "Overlay position: bottom-right, bottom-left, top-right, top-left")
	avatarComposeCmd.Flags().IntVar(&acMarginX, "margin-x", 56, "Horizontal margin in pixels")
	avatarComposeCmd.Flags().IntVar(&acMarginY, "margin-y", 56, "Vertical margin in pixels")
	avatarComposeCmd.Flags().IntVar(&acBorder, "border", 0, "Border ring width in pixels (0 disables)")
	avatarComposeCmd.Flags().StringVar(&acBorderColor, "border-color", "white", "Border ring color (ffmpeg color name or 0xRRGGBB)")

	for _, flag := range []string{"slides", "avatar"} {
		if err := avatarComposeCmd.MarkFlagRequired(flag); err != nil {
			panic(err)
		}
	}

	avatarCmd.AddCommand(avatarComposeCmd)
}

func runAvatarCompose(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	for _, path := range []string{acSlides, acAvatar} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("input file does not exist: %s", path)
		}
	}
	if acAudio != "" {
		if _, err := os.Stat(acAudio); err != nil {
			return fmt.Errorf("audio file does not exist: %s", acAudio)
		}
	}

	fmt.Printf("Compositing %s onto %s...\n", acAvatar, acSlides)
	if err := video.OverlayAvatar(ctx, acSlides, acAvatar, acOutput, video.OverlayOptions{
		Diameter:    acDiameter,
		Position:    acPosition,
		MarginX:     acMarginX,
		MarginY:     acMarginY,
		BorderWidth: acBorder,
		BorderColor: acBorderColor,
		AudioPath:   acAudio,
	}); err != nil {
		return err
	}

	fmt.Printf("Composited video written to %s\n", acOutput)
	return nil
}
