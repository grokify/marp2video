package video

import (
	"strings"
	"testing"
)

func TestBuildOverlayFilterDefaults(t *testing.T) {
	opts := OverlayOptions{}
	opts.applyDefaults()

	filter, err := buildOverlayFilter(opts)
	if err != nil {
		t.Fatalf("buildOverlayFilter() error = %v", err)
	}

	for _, want := range []string{
		"crop='min(iw\\,ih)':'min(iw\\,ih)'",
		"scale=320:320",
		"format=rgba",
		"overlay=x=main_w-overlay_w-56:y=main_h-overlay_h-56",
		"format=yuv420p[vout]",
	} {
		if !strings.Contains(filter, want) {
			t.Errorf("filter missing %q:\n%s", want, filter)
		}
	}
	if strings.Contains(filter, "[disc]") {
		t.Errorf("filter contains border disc without border configured:\n%s", filter)
	}
}

// TestBuildOverlayFilterNormalizesBaseFPS guards against a regression where
// slide/screen-recorded base videos that hold a single frame for a long
// duration (rather than encoding continuous frames) made the overlay filter
// composite the avatar exactly once and then appear frozen: the overlay
// filter emits one output frame per BASE-input frame, so an un-normalized,
// frame-sparse base silently drops the avatar's motion.
func TestBuildOverlayFilterNormalizesBaseFPS(t *testing.T) {
	opts := OverlayOptions{}
	opts.applyDefaults()

	filter, err := buildOverlayFilter(opts)
	if err != nil {
		t.Fatalf("buildOverlayFilter() error = %v", err)
	}

	if !strings.Contains(filter, "[0:v]fps=30[base]") {
		t.Errorf("filter does not normalize the base stream's frame rate:\n%s", filter)
	}
	if strings.Contains(filter, "[0:v][avatar]overlay") || strings.Contains(filter, "[0:v][disc]overlay") {
		t.Errorf("filter overlays directly onto the un-normalized [0:v] instead of [base]:\n%s", filter)
	}
}

func TestBuildOverlayFilterWithBorder(t *testing.T) {
	opts := OverlayOptions{Diameter: 300, BorderWidth: 6, BorderColor: "0x336699", MarginX: 40, MarginY: 50}
	opts.applyDefaults()

	filter, err := buildOverlayFilter(opts)
	if err != nil {
		t.Fatalf("buildOverlayFilter() error = %v", err)
	}

	for _, want := range []string{
		"color=c=0x336699:s=312x312",
		// Disc anchored border-width closer to the corner than the avatar.
		"overlay=x=main_w-overlay_w-34:y=main_h-overlay_h-44[withdisc]",
		"[withdisc][avatar]overlay=x=main_w-overlay_w-40:y=main_h-overlay_h-50",
	} {
		if !strings.Contains(filter, want) {
			t.Errorf("filter missing %q:\n%s", want, filter)
		}
	}
}

func TestBuildOverlayFilterPositions(t *testing.T) {
	tests := []struct {
		position string
		wantX    string
		wantY    string
	}{
		{PositionBottomRight, "x=main_w-overlay_w-56", "y=main_h-overlay_h-56"},
		{PositionBottomLeft, "x=56", "y=main_h-overlay_h-56"},
		{PositionTopRight, "x=main_w-overlay_w-56", "y=56"},
		{PositionTopLeft, "x=56", "y=56"},
	}
	for _, tt := range tests {
		t.Run(tt.position, func(t *testing.T) {
			opts := OverlayOptions{Position: tt.position}
			opts.applyDefaults()

			filter, err := buildOverlayFilter(opts)
			if err != nil {
				t.Fatalf("buildOverlayFilter() error = %v", err)
			}
			if !strings.Contains(filter, tt.wantX) || !strings.Contains(filter, tt.wantY) {
				t.Errorf("filter missing %s/%s:\n%s", tt.wantX, tt.wantY, filter)
			}
		})
	}
}

func TestBuildOverlayFilterInvalidPosition(t *testing.T) {
	opts := OverlayOptions{Position: "center"}
	opts.applyDefaults()

	if _, err := buildOverlayFilter(opts); err == nil {
		t.Error("buildOverlayFilter() error = nil, want error for invalid position")
	}
}
