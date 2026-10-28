package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/webp"
)

const webpConvertFolder = "webp-converted"

// convertWebPToPNG decodes a WebP page and stores it as PNG inside a
// subfolder of its directory, returning the new path.
//
// Lossy WebP stores colors as limited-range YCbCr (BT.601, Y in 16..235), but
// golang.org/x/image/webp returns it as a full-range image.YCbCr. Passing that
// straight to pdfcpu washes the page out (white turns into RGB 235), so the
// colors are converted here with the limited-range formula instead.
func convertWebPToPNG(webpPath string) (string, error) {
	in, err := os.Open(webpPath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	img, err := webp.Decode(in)
	if err != nil {
		return "", err
	}

	switch im := img.(type) {
	case *image.YCbCr:
		img = limitedYCbCrToNRGBA(im, nil)
	case *image.NYCbCrA:
		img = limitedYCbCrToNRGBA(&im.YCbCr, im)
	}

	outDir := filepath.Join(filepath.Dir(webpPath), webpConvertFolder)
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(webpPath), filepath.Ext(webpPath))
	outPath := filepath.Join(outDir, base+".png")

	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(out, img); err != nil {
		return "", err
	}
	return outPath, nil
}

// limitedYCbCrToNRGBA converts limited-range BT.601 YCbCr to NRGBA. alpha
// is non-nil when src belongs to an image.NYCbCrA.
func limitedYCbCrToNRGBA(src *image.YCbCr, alpha *image.NYCbCrA) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			yi := src.YOffset(x, y)
			ci := src.COffset(x, y)

			yy := 1.164 * (float64(src.Y[yi]) - 16)
			cb := float64(src.Cb[ci]) - 128
			cr := float64(src.Cr[ci]) - 128

			a := uint8(255)
			if alpha != nil {
				a = alpha.A[alpha.AOffset(x, y)]
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: clampUint8(yy + 1.596*cr),
				G: clampUint8(yy - 0.392*cb - 0.813*cr),
				B: clampUint8(yy + 2.017*cb),
				A: a,
			})
		}
	}
	return dst
}

func clampUint8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}
