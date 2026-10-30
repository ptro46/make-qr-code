package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/makiuchi-d/gozxing"
	zxingqr "github.com/makiuchi-d/gozxing/qrcode"
	qrcode "github.com/skip2/go-qrcode"
	xdraw "golang.org/x/image/draw"
)

const outputSize = 2048

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur :", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("make-qr-code", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "qrcode.png", "fichier PNG de sortie (2048 × 2048)")
	text := flags.String("text", "", "texte ou URL à encoder (alternative au premier argument)")
	logoPath := flags.String("logo", "", "logo PNG ou JPEG (alternative au second argument)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage : make-qr-code [-o sortie.png] <texte-ou-URL> <logo.png|logo.jpg>")
		fmt.Fprintln(stderr, "    ou : make-qr-code [-o sortie.png] -text <texte> -logo <logo>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() == 2 && *text == "" && *logoPath == "" {
		*text, *logoPath = flags.Arg(0), flags.Arg(1)
	} else if flags.NArg() != 0 || *text == "" || *logoPath == "" {
		flags.Usage()
		return errors.New("fournir le texte et le logo, soit en arguments, soit avec -text et -logo")
	}
	if *output == "" {
		return errors.New("le chemin de sortie est vide")
	}
	// Never replace the input logo, even when the paths differ through a symlink.
	inputInfo, err := os.Stat(*logoPath)
	if err != nil {
		return fmt.Errorf("logo : %w", err)
	}
	if outputInfo, err := os.Stat(*output); err == nil && os.SameFile(inputInfo, outputInfo) {
		return errors.New("le fichier de sortie doit être différent du logo")
	}
	logo, err := loadLogo(*logoPath)
	if err != nil {
		return err
	}
	data, percent, err := generate(*text, logo)
	if err != nil {
		return err
	}
	if err := savePNG(*output, data); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "QR-Code créé : %s (2048 × 2048, logo %d %%, décodage vérifié)\n", *output, percent)
	return nil
}

func loadLogo(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ouvrir le logo : %w", err)
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("lire le logo PNG/JPEG : %w", err)
	}
	if format != "png" && format != "jpeg" {
		return nil, errors.New("le logo doit être un PNG ou JPEG")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return nil, errors.New("dimensions du logo invalides ou supérieures à 40 millions de pixels")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	logo, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("décoder le logo : %w", err)
	}
	return logo, nil
}

func generate(text string, logo image.Image) ([]byte, int, error) {
	if text == "" || !utf8.ValidString(text) {
		return nil, 0, errors.New("le texte doit être non vide et en UTF-8 valide")
	}
	code, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		return nil, 0, fmt.Errorf("encoder le texte : %w", err)
	}
	bitmap := code.Bitmap() // Includes the four-module quiet zone.
	scale := outputSize / len(bitmap)
	side := scale * len(bitmap)
	offset := (outputSize - side) / 2
	base := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
	draw.Draw(base, base.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for y, row := range bitmap {
		for x, black := range row {
			if black {
				r := image.Rect(offset+x*scale, offset+y*scale, offset+(x+1)*scale, offset+(y+1)*scale)
				draw.Draw(base, r, image.NewUniform(color.Black), image.Point{}, draw.Src)
			}
		}
	}
	// Check the finished PNG and shrink the logo if it prevents decoding.
	for percent := 18; percent >= 10; percent -= 2 {
		result := image.NewRGBA(base.Bounds())
		draw.Draw(result, result.Bounds(), base, image.Point{}, draw.Src)
		addLogo(result, logo, (len(bitmap)-8)*scale*percent/100, scale)
		var buf bytes.Buffer
		if err := png.Encode(&buf, result); err != nil {
			return nil, 0, fmt.Errorf("encoder le PNG : %w", err)
		}
		if err := verifyPNG(buf.Bytes(), text); err == nil {
			return buf.Bytes(), percent, nil
		}
	}
	return nil, 0, errors.New("QR-Code avec logo illisible après vérification ; essayer un texte plus court ou un logo plus simple")
}

func addLogo(dst *image.RGBA, logo image.Image, maxSide, padding int) {
	b := logo.Bounds()
	w, h := maxSide, maxSide
	if b.Dx() >= b.Dy() {
		h = max(1, maxSide*b.Dy()/b.Dx())
	} else {
		w = max(1, maxSide*b.Dx()/b.Dy())
	}
	x, y := (outputSize-w)/2, (outputSize-h)/2
	r := image.Rect(x, y, x+w, y+h)
	draw.Draw(dst, r.Inset(-padding), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, r, logo, b, draw.Over, nil)
}

func verifyPNG(data []byte, expected string) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	binary, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return err
	}
	result, err := zxingqr.NewQRCodeReader().Decode(binary, map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_CHARACTER_SET: "UTF-8",
		gozxing.DecodeHintType_TRY_HARDER:    true,
	})
	if err != nil {
		return err
	}
	if result.GetText() != expected {
		return errors.New("le contenu décodé diffère du texte fourni")
	}
	return nil
}

func savePNG(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".make-qr-code-*.png")
	if err != nil {
		return fmt.Errorf("créer la sortie : %w", err)
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("enregistrer la sortie : %w", err)
	}
	return nil
}
