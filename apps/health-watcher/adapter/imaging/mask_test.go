package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"apps/health-watcher/domain"
)

func whiteJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func lum(img image.Image, x, y int) uint32 {
	r, g, b, _ := img.At(x, y).RGBA()
	return (r + g + b) / 3 >> 8
}

func TestGivenARotatedBoxWhenMaskingThenItsInsideIsBlackAndTheRestStaysWhite(t *testing.T) {
	// A diamond around the centre of a 200x200 image.
	box := domain.Quad{{X: 0.5, Y: 0.3}, {X: 0.7, Y: 0.5}, {X: 0.5, Y: 0.7}, {X: 0.3, Y: 0.5}}
	out, err := Masker{Quality: 95}.Mask(whiteJPEG(t, 200, 200), []domain.Quad{box})
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{100, 100}, {100, 65}, {135, 100}, {100, 135}, {65, 100}} {
		if l := lum(img, p[0], p[1]); l > 40 {
			t.Errorf("inside pixel %v has luminance %d", p, l)
		}
	}
	for _, p := range [][2]int{{10, 10}, {65, 65}, {135, 135}, {190, 100}} {
		if l := lum(img, p[0], p[1]); l < 215 {
			t.Errorf("outside pixel %v has luminance %d", p, l)
		}
	}
}

func TestGivenBytesThatAreNotAJPEGWhenMaskingThenItFails(t *testing.T) {
	if _, err := (Masker{}).Mask([]byte("not an image"), nil); err == nil {
		t.Fatal("no error")
	}
}
