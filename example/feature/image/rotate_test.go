package image

import (
	"encoding/json"
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestRotate90CW(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := range 2 {
		for x := range 3 {
			src.Set(x, y, color.RGBA{R: uint8(x + 1), G: uint8(y + 1), B: 9, A: 255})
		}
	}

	got := rotate90CW(src)

	if w, h := got.Bounds().Dx(), got.Bounds().Dy(); w != 2 || h != 3 {
		t.Fatalf("bounds = %dx%d, want 2x3 with width and height swapped", w, h)
	}

	tests := []struct {
		sx, sy int
	}{
		{0, 0},
		{2, 0},
		{0, 1},
		{2, 1},
	}

	for _, tt := range tests {
		dx, dy := 2-1-tt.sy, tt.sx
		if gotColor, wantColor := got.At(dx, dy), src.At(tt.sx, tt.sy); gotColor != wantColor {
			t.Errorf("src(%d,%d) -> dst(%d,%d) = %v, want %v", tt.sx, tt.sy, dx, dy, gotColor, wantColor)
		}
	}
}

// Discord requires dot-prefixed extensions, never MIME types like "image/png".
func TestRotateModalFileTypes(t *testing.T) {
	data, err := json.Marshal(rotateModal())
	if err != nil {
		t.Fatalf("Marshal modal: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, `"file_types":[".png"]`) {
		t.Fatalf("modal = %s, want file_types [\".png\"]", got)
	}

	if strings.Contains(got, "image/png") {
		t.Fatalf("modal = %s, must not send a MIME type in file_types", got)
	}
}
