package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testLogo(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds().Inset(5), image.NewUniform(color.RGBA{25, 100, 190, 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(w/3, h/4, 2*w/3, 3*h/4), image.NewUniform(color.White), image.Point{}, draw.Src)
	return img
}

func TestCLIProducesReadablePNG(t *testing.T) {
	for _, tc := range []struct {
		name, text, format string
		w, h               int
		flags              bool
	}{
		{"url-png", "https://example.org/formulaire?a=1&b=2", "png", 240, 240, false},
		{"unicode-jpeg", "Bonjour, été à Paris ! 日本語 😀\nDeuxième ligne", "jpeg", 400, 120, true},
		{"long-portrait", strings.Repeat("Texte avec espaces et accents é. ", 25), "png", 120, 400, false},
		{"short", "A", "png", 200, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			logoPath := filepath.Join(dir, "logo."+tc.format)
			f, err := os.Create(logoPath)
			if err != nil {
				t.Fatal(err)
			}
			if tc.format == "png" {
				err = png.Encode(f, testLogo(tc.w, tc.h))
			} else {
				err = jpeg.Encode(f, testLogo(tc.w, tc.h), nil)
			}
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "result.png")
			args := []string{"-o", output, tc.text, logoPath}
			if tc.flags {
				args = []string{"-o", output, "-text", tc.text, "-logo", logoPath}
			}
			if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width != 2048 || cfg.Height != 2048 {
				t.Fatalf("dimensions: %+v, %v", cfg, err)
			}
			if err := verifyPNG(data, tc.text); err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			colored := false
			for y := 850; y < 1200; y++ {
				for x := 850; x < 1200; x++ {
					r, _, b, _ := img.At(x, y).RGBA()
					if b > r+1000 {
						colored = true
					}
				}
			}
			if !colored {
				t.Fatal("logo color absent from center")
			}
			if r, g, b, _ := img.At(0, 0).RGBA(); r != 65535 || g != 65535 || b != 65535 {
				t.Fatal("missing white margin")
			}
		})
	}
}

func TestInvalidInputs(t *testing.T) {
	dir := t.TempDir()
	logoPath := filepath.Join(dir, "logo.png")
	f, err := os.Create(logoPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, testLogo(100, 100)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	for _, args := range [][]string{
		{}, {"text"}, {"text", "missing.png"}, {"-text", "text"},
		{"-o", logoPath, "text", logoPath},
		{"-o", filepath.Join(dir, "result.png"), strings.Repeat("x", 10000), logoPath},
	} {
		if err := run(args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("expected error: %v", args)
		}
	}
	if err := run([]string{"-h"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "result.png")); !os.IsNotExist(err) {
		t.Fatal("invalid input wrote output")
	}
}
