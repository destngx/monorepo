package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"testing"

	"apps/health-watcher/domain"
)

// pageWith draws a paper page with a dark picture at pic and a line of
// dark "text" at text, both in pixels.
func pageWith(t *testing.T, w, h int, pic, text image.Rectangle, picColor color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{235, 235, 230, 255}
			p := image.Pt(x, y)
			switch {
			case p.In(pic):
				c = picColor
			case p.In(text) && x%8 == 0:
				c = color.RGBA{30, 30, 30, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.012 }

func TestGivenALooseBoxAroundAFilmWhenCroppingThenItTightensToTheFilmAndLeavesTheTextOut(t *testing.T) {
	// A film at 20-80% across and 30-60% down, with a text line just below it.
	page := pageWith(t, 400, 600, image.Rect(80, 180, 320, 360), image.Rect(80, 370, 320, 382), color.RGBA{40, 40, 45, 255})
	crop, err := Cropper{Quality: 90}.Crop(page, domain.Rect{X0: 0.15, Y0: 0.27, X1: 0.85, Y1: 0.64})
	if err != nil {
		t.Fatal(err)
	}
	if !near(crop.Box.X0, 0.2) || !near(crop.Box.Y0, 0.3) || !near(crop.Box.X1, 0.8) || !near(crop.Box.Y1, 0.6) {
		t.Fatalf("box = %+v", crop.Box)
	}
	img, err := jpeg.Decode(bytes.NewReader(crop.JPEG))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != crop.Width || b.Dy() != crop.Height || math.Abs(float64(crop.Width-240)) > 6 || math.Abs(float64(crop.Height-180)) > 6 {
		t.Fatalf("size = %dx%d, image %v", crop.Width, crop.Height, b)
	}
}

func TestGivenABoxThatCutsOffAFilmEdgeWhenCroppingThenItGrowsToTheEdge(t *testing.T) {
	page := pageWith(t, 400, 600, image.Rect(80, 180, 320, 360), image.Rectangle{}, color.RGBA{40, 40, 45, 255})
	crop, err := Cropper{Quality: 90}.Crop(page, domain.Rect{X0: 0.22, Y0: 0.32, X1: 0.78, Y1: 0.58})
	if err != nil {
		t.Fatal(err)
	}
	if !near(crop.Box.X0, 0.2) || !near(crop.Box.Y0, 0.3) || !near(crop.Box.X1, 0.8) || !near(crop.Box.Y1, 0.6) {
		t.Fatalf("box = %+v", crop.Box)
	}
}

func TestGivenAPalePinkTraceGridWhenCroppingThenTheGridCountsAsThePicture(t *testing.T) {
	page := pageWith(t, 400, 600, image.Rect(40, 100, 360, 300), image.Rect(40, 320, 360, 332), color.RGBA{245, 200, 200, 255})
	crop, err := Cropper{Quality: 90}.Crop(page, domain.Rect{X0: 0.06, Y0: 0.13, X1: 0.94, Y1: 0.54})
	if err != nil {
		t.Fatal(err)
	}
	if !near(crop.Box.X0, 0.1) || !near(crop.Box.Y0, 1.0/6) || !near(crop.Box.X1, 0.9) || !near(crop.Box.Y1, 0.5) {
		t.Fatalf("box = %+v", crop.Box)
	}
}

func TestGivenABoxOverBlankPaperWhenCroppingThenTheModelsBoxIsKept(t *testing.T) {
	page := pageWith(t, 400, 600, image.Rectangle{}, image.Rectangle{}, color.RGBA{})
	box := domain.Rect{X0: 0.2, Y0: 0.2, X1: 0.6, Y1: 0.5}
	crop, err := Cropper{Quality: 90}.Crop(page, box)
	if err != nil {
		t.Fatal(err)
	}
	if crop.Box != box {
		t.Fatalf("box = %+v", crop.Box)
	}
}

func TestGivenBytesThatAreNotAJPEGWhenCroppingThenItFails(t *testing.T) {
	if _, err := (Cropper{}).Crop([]byte("not an image"), domain.Rect{X1: 1, Y1: 1}); err == nil {
		t.Fatal("no error")
	}
}
