package video

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Overlay positions for avatar overlays.
const (
	PositionBottomRight = "bottom-right"
	PositionBottomLeft  = "bottom-left"
	PositionTopRight    = "top-right"
	PositionTopLeft     = "top-left"
)

// OverlayOptions configures a talking-head avatar overlay.
type OverlayOptions struct {
	// Diameter is the avatar circle diameter in pixels.
	// Default: 320.
	Diameter int

	// Position anchors the overlay corner.
	// One of the Position* constants. Default: bottom-right.
	Position string

	// MarginX is the horizontal margin from the anchored edge in pixels.
	// Default: 56.
	MarginX int

	// MarginY is the vertical margin from the anchored edge in pixels.
	// Default: 56.
	MarginY int

	// BorderWidth draws a solid ring behind the avatar circle when > 0.
	BorderWidth int

	// BorderColor is the ring color (ffmpeg color name or 0xRRGGBB).
	// Default: white.
	BorderColor string

	// AudioPath, when set, replaces the output audio track with this
	// file (the authoritative narration), per the videoascode principle
	// that narration audio — not provider-returned audio — is canonical.
	// When empty, the base video's audio track is passed through.
	AudioPath string

	// FPS is the frame rate the base video is normalized to before
	// overlaying, so the avatar's motion isn't frozen between the base
	// video's (possibly sparse) native frames. Default: 30.
	FPS int
}

// applyDefaults fills zero values with defaults.
func (o *OverlayOptions) applyDefaults() {
	if o.Diameter <= 0 {
		o.Diameter = 320
	}
	if o.Position == "" {
		o.Position = PositionBottomRight
	}
	if o.MarginX <= 0 {
		o.MarginX = 56
	}
	if o.MarginY <= 0 {
		o.MarginY = 56
	}
	if o.BorderColor == "" {
		o.BorderColor = "white"
	}
	if o.FPS <= 0 {
		o.FPS = 30
	}
}

// OverlayAvatar composites a talking-head avatar video as a circular
// overlay onto a base (slides) video. The avatar video is center-cropped
// to a square, scaled to the configured diameter, circle-masked, and
// anchored at the configured corner. The avatar video's own audio track
// is always discarded.
func OverlayAvatar(ctx context.Context, basePath, avatarPath, outputPath string, opts OverlayOptions) error {
	opts.applyDefaults()

	filter, err := buildOverlayFilter(opts)
	if err != nil {
		return err
	}

	args := []string{
		"-y",
		"-i", basePath,
		"-i", avatarPath,
	}
	if opts.AudioPath != "" {
		args = append(args, "-i", opts.AudioPath)
	}

	args = append(args, "-filter_complex", filter, "-map", "[vout]")
	if opts.AudioPath != "" {
		args = append(args, "-map", "2:a", "-c:a", "aac", "-b:a", "192k")
	} else {
		args = append(args, "-map", "0:a?", "-c:a", "copy")
	}

	args = append(args,
		"-c:v", "libx264",
		"-shortest",
		"-movflags", "+faststart",
		outputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// buildOverlayFilter builds the ffmpeg filter_complex graph for the
// circular avatar overlay.
func buildOverlayFilter(opts OverlayOptions) (string, error) {
	avatarX, avatarY, err := overlayPosition(opts.Position, opts.MarginX, opts.MarginY)
	if err != nil {
		return "", err
	}

	// circleAlpha masks everything outside the inscribed circle.
	const circleAlpha = "a='if(lte((X-W/2)*(X-W/2)+(Y-H/2)*(Y-H/2),(W/2)*(W/2)),255,0)'"

	var filters []string

	// Center-crop the avatar to a square, scale, and circle-mask.
	filters = append(filters, fmt.Sprintf(
		"[1:v]crop='min(iw\\,ih)':'min(iw\\,ih)',scale=%d:%d,format=rgba,"+
			"geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':%s[avatar]",
		opts.Diameter, opts.Diameter, circleAlpha))

	// The overlay filter emits one output frame per BASE-input frame by
	// default. Slide/screen-recorded base videos commonly hold a single
	// frame for a long duration via PTS rather than encoding continuous
	// frames, which — left unnormalized — makes the overlay filter
	// composite the (continuously animated) avatar exactly once and then
	// visually freeze it for the rest of that hold. Forcing a constant
	// frame rate on the base stream first ensures the overlay filter
	// samples the avatar's current frame regularly, so its motion (lip
	// sync) actually shows through.
	filters = append(filters, fmt.Sprintf("[0:v]fps=%d[base]", opts.FPS))

	base := "[base]"
	if opts.BorderWidth > 0 {
		// Draw a solid disc behind the avatar; the visible ring is the
		// border-width margin around the avatar circle.
		discSize := opts.Diameter + 2*opts.BorderWidth
		discX, discY, err := overlayPosition(opts.Position, opts.MarginX-opts.BorderWidth, opts.MarginY-opts.BorderWidth)
		if err != nil {
			return "", err
		}
		filters = append(filters, fmt.Sprintf(
			"color=c=%s:s=%dx%d:d=1,format=rgba,geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':%s[disc]",
			opts.BorderColor, discSize, discSize, circleAlpha))
		filters = append(filters, fmt.Sprintf(
			"%s[disc]overlay=x=%s:y=%s[withdisc]", base, discX, discY))
		base = "[withdisc]"
	}

	filters = append(filters, fmt.Sprintf(
		"%s[avatar]overlay=x=%s:y=%s,format=yuv420p[vout]", base, avatarX, avatarY))

	return strings.Join(filters, ";"), nil
}

// overlayPosition returns the ffmpeg overlay x/y expressions for the
// given anchor position and margins.
func overlayPosition(position string, marginX, marginY int) (x, y string, err error) {
	right := fmt.Sprintf("main_w-overlay_w-%d", marginX)
	bottom := fmt.Sprintf("main_h-overlay_h-%d", marginY)
	left := fmt.Sprintf("%d", marginX)
	top := fmt.Sprintf("%d", marginY)

	switch position {
	case PositionBottomRight:
		return right, bottom, nil
	case PositionBottomLeft:
		return left, bottom, nil
	case PositionTopRight:
		return right, top, nil
	case PositionTopLeft:
		return left, top, nil
	default:
		return "", "", fmt.Errorf("unknown overlay position %q (valid: %s, %s, %s, %s)",
			position, PositionBottomRight, PositionBottomLeft, PositionTopRight, PositionTopLeft)
	}
}
