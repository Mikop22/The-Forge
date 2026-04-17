package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

func TestPreviewCanvasDrawSpriteClipsToBounds(t *testing.T) {
	canvas := newPreviewCanvas(3, 3, color.RGBA{A: 255})
	sprite := image.NewRGBA(image.Rect(0, 0, 2, 2))
	sprite.Set(0, 0, color.RGBA{R: 255, A: 255})
	sprite.Set(1, 0, color.RGBA{G: 255, A: 255})
	sprite.Set(0, 1, color.RGBA{B: 255, A: 255})
	sprite.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("drawSprite panicked: %v", r)
		}
	}()

	canvas.drawSprite(sprite, -0.5, -0.5, 1, 0)

	if got := canvas.pixels[0]; got != (color.RGBA{R: 255, G: 255, A: 255}) {
		t.Fatalf("pixel[0] = %#v, want bottom-right sprite color", got)
	}
	if got := canvas.pixels[1]; got != (color.RGBA{A: 255}) {
		t.Fatalf("pixel[1] = %#v, want background", got)
	}
	if got := canvas.pixels[3]; got != (color.RGBA{A: 255}) {
		t.Fatalf("pixel[3] = %#v, want background", got)
	}
}

func TestPreviewCanvasRenderHalfBlocksHasStableWidth(t *testing.T) {
	canvas := newPreviewCanvas(5, 4, color.RGBA{A: 255})
	canvas.set(0, 0, color.RGBA{R: 255, A: 255})
	canvas.set(1, 0, color.RGBA{G: 255, A: 255})
	canvas.set(2, 1, color.RGBA{B: 255, A: 255})
	canvas.set(4, 3, color.RGBA{R: 255, B: 255, A: 255})

	rendered := canvas.renderHalfBlocks()
	lines := strings.Split(rendered, "\n")
	if len(lines) != 2 {
		t.Fatalf("rendered line count = %d, want 2", len(lines))
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got > 5 {
			t.Fatalf("line %d width = %d, want <= 5", i, got)
		}
	}
}

func TestPreviewCanvasTransparentPixelsPreserveBackground(t *testing.T) {
	bg := color.RGBA{R: 12, G: 34, B: 56, A: 255}
	canvas := newPreviewCanvas(2, 2, bg)
	sprite := image.NewRGBA(image.Rect(0, 0, 1, 1))
	sprite.Set(0, 0, color.RGBA{R: 255, A: 0x7f})

	canvas.drawSprite(sprite, 0, 0, 1, 0)

	for i, got := range canvas.pixels {
		if got != bg {
			t.Fatalf("pixel[%d] = %#v, want background %#v", i, got, bg)
		}
	}
}

func TestCombatPreviewProfileForSwordUsesSwing(t *testing.T) {
	profile := combatPreviewProfileFor(craftedItem{subType: "Sword"}, nil)
	if profile.kind != combatPreviewSwing {
		t.Fatalf("profile.kind = %v, want swing", profile.kind)
	}
}

func TestCombatPreviewProfileForSpearUsesThrust(t *testing.T) {
	profile := combatPreviewProfileFor(craftedItem{subType: "Spear"}, nil)
	if profile.kind != combatPreviewThrust {
		t.Fatalf("profile.kind = %v, want thrust", profile.kind)
	}
}

func TestCombatPreviewProfileForGunBowStaffUsesShoot(t *testing.T) {
	for _, subType := range []string{"Gun", "Bow", "Staff"} {
		t.Run(subType, func(t *testing.T) {
			profile := combatPreviewProfileFor(craftedItem{subType: subType}, nil)
			if profile.kind != combatPreviewShoot {
				t.Fatalf("profile.kind = %v, want shoot", profile.kind)
			}
		})
	}
}

func TestCombatPreviewProfileUsesUseTimeForLoopLength(t *testing.T) {
	fast := combatPreviewProfileFor(craftedItem{stats: itemStats{UseTime: 8}}, nil)
	slow := combatPreviewProfileFor(craftedItem{stats: itemStats{UseTime: 40}}, nil)

	if fast.loopTicks != 8 {
		t.Fatalf("fast.loopTicks = %d, want 8", fast.loopTicks)
	}
	if slow.loopTicks != 24 {
		t.Fatalf("slow.loopTicks = %d, want 24", slow.loopTicks)
	}
	if fast.loopTicks >= slow.loopTicks {
		t.Fatalf("loopTicks did not increase with use time: fast=%d slow=%d", fast.loopTicks, slow.loopTicks)
	}
}
