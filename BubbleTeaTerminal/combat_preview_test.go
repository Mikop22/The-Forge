package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTestSprite(t *testing.T, width, height int, paint func(*image.RGBA)) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	paint(img)

	path := filepath.Join(t.TempDir(), "sprite.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test sprite: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode test sprite: %v", err)
	}

	return path
}

func TestLoadPreviewSpriteCropsTransparentBounds(t *testing.T) {
	path := writeTestSprite(t, 6, 6, func(img *image.RGBA) {
		img.Set(2, 1, color.RGBA{R: 255, A: 255})
		img.Set(3, 4, color.RGBA{G: 255, A: 255})
	})

	spr, ok := loadPreviewSprite(path)
	if !ok {
		t.Fatal("loadPreviewSprite returned ok=false")
	}
	if spr.Bounds().Dx() != 2 || spr.Bounds().Dy() != 4 {
		t.Fatalf("cropped bounds = %v, want 2x4", spr.Bounds())
	}
}

func TestLoadPreviewSpriteMissingPathReturnsFalse(t *testing.T) {
	if _, ok := loadPreviewSprite("/does/not/exist.png"); ok {
		t.Fatal("missing sprite returned ok=true")
	}
}
