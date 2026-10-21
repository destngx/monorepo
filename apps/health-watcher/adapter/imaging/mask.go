// Package imaging blacks out boxes on page images before they leave the
// Mac (HW-AI-7).
package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"sort"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Masker fills quads with black on JPEG images.
type Masker struct {
	// Quality is the JPEG quality of the masked image.
	Quality int
}

var _ port.ImageMasker = Masker{}

// Mask decodes a JPEG, fills every box with black and encodes the result.
func (m Masker) Mask(data []byte, boxes []domain.Quad) ([]byte, error) {
	src, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode page image: %w", err)
	}
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	for _, q := range boxes {
		fillQuad(dst, q)
	}
	quality := m.Quality
	if quality <= 0 {
		quality = 85
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode masked image: %w", err)
	}
	return out.Bytes(), nil
}

// fillQuad paints a quad black with a scanline fill. Edges are rounded
// outward so no partly covered pixel keeps its text.
func fillQuad(img *image.RGBA, q domain.Quad) {
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	var xs, ys [4]float64
	minY, maxY := math.Inf(1), math.Inf(-1)
	for i, p := range q {
		xs[i], ys[i] = p.X*w, p.Y*h
		minY, maxY = math.Min(minY, ys[i]), math.Max(maxY, ys[i])
	}
	black := color.RGBA{A: 255}
	y0 := max(b.Min.Y, int(math.Floor(minY)))
	y1 := min(b.Max.Y-1, int(math.Ceil(maxY)))
	for y := y0; y <= y1; y++ {
		// Sample the top and bottom of the pixel row and fill the union.
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, sy := range []float64{float64(y), float64(y) + 1} {
			cross := crossings(xs, ys, sy)
			if len(cross) >= 2 {
				lo, hi = math.Min(lo, cross[0]), math.Max(hi, cross[len(cross)-1])
			}
		}
		if math.IsInf(lo, 1) {
			continue
		}
		x0 := max(b.Min.X, int(math.Floor(lo)))
		x1 := min(b.Max.X-1, int(math.Ceil(hi)))
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y, black)
		}
	}
}

// crossings returns where the quad's edges cross the horizontal line y,
// sorted. A quad is convex, so the fill spans the first to the last.
func crossings(xs, ys [4]float64, y float64) []float64 {
	var out []float64
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		ya, yb := ys[i], ys[j]
		if ya == yb {
			if y == ya {
				out = append(out, xs[i], xs[j])
			}
			continue
		}
		if (y < math.Min(ya, yb)) || (y > math.Max(ya, yb)) {
			continue
		}
		t := (y - ya) / (yb - ya)
		out = append(out, xs[i]+t*(xs[j]-xs[i]))
	}
	sort.Float64s(out)
	return out
}
