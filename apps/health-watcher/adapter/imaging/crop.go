package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"slices"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Cropper cuts pictures out of page JPEGs (HW-READ-11). The model's box is
// only roughly right, so each side is moved to the picture's edge first.
type Cropper struct {
	// Quality is the JPEG quality of the crop; 0 means 90.
	Quality int
}

var _ port.PictureCropper = Cropper{}

const (
	// cropMargin is how far past the model's box the picture may reach, as
	// a share of the page's longer side.
	cropMargin = 0.05
	// minKept is the least share of the model's box the fitted box keeps;
	// below that the fit went wrong and the model's box is used.
	minKept = 0.4
	// cropSamples bounds the samples taken along each side of the box.
	cropSamples = 800
)

// Crop fits box to the picture under it and returns the picture as a JPEG.
func (c Cropper) Crop(data []byte, box domain.Rect) (port.Crop, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return port.Crop{}, fmt.Errorf("decode page image: %w", err)
	}
	b := img.Bounds()
	box = box.Clip()
	fit := fitPicture(img, box)
	px := image.Rect(b.Min.X+int(math.Floor(fit.X0*float64(b.Dx()))), b.Min.Y+int(math.Floor(fit.Y0*float64(b.Dy()))),
		b.Min.X+int(math.Ceil(fit.X1*float64(b.Dx()))), b.Min.Y+int(math.Ceil(fit.Y1*float64(b.Dy())))).Intersect(b)
	if px.Empty() {
		return port.Crop{}, fmt.Errorf("picture box %+v is outside the page", box)
	}
	out := image.NewRGBA(image.Rect(0, 0, px.Dx(), px.Dy()))
	draw.Draw(out, out.Bounds(), img, px.Min, draw.Src)
	q := c.Quality
	if q == 0 {
		q = 90
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: q}); err != nil {
		return port.Crop{}, fmt.Errorf("encode picture: %w", err)
	}
	return port.Crop{JPEG: buf.Bytes(), Box: fit, Width: px.Dx(), Height: px.Dy()}, nil
}

// fitPicture moves each side of the model's box to the picture's edge: the
// line within cropMargin of that side where the band outside looks most
// like paper against the band inside, along the side's whole length.
// Films, scans, photos and trace grids are all darker or more colourful
// than the paper around them, while an edge inside a picture leans one way
// here and the other way there. The median
// along the line keeps a text row, whose ink is sparse, from looking like
// an edge; bands as wide as edgeBand let the edge of a tilted photo count
// along its whole length; comparing neighbours keeps the shade across a
// phone photo from mattering. A side with no clear edge stays where the
// model put it.
func fitPicture(img image.Image, box domain.Rect) domain.Rect {
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	step := max(1, int(max(w, h))/(2*cropSamples))
	cols, rows := b.Dx()/step, b.Dy()/step
	if cols < 4 || rows < 4 {
		return box
	}
	// sumX[j][i] sums row j's paperness left of column i; sumY[i][j]
	// sums column i's paperness above row j.
	sumX := make([][]float64, rows)
	sumY := make([][]float64, cols)
	for i := range cols {
		sumY[i] = make([]float64, rows+1)
	}
	for j := range rows {
		sumX[j] = make([]float64, cols+1)
		for i := range cols {
			l := paperness(img.At(b.Min.X+i*step, b.Min.Y+j*step))
			sumX[j][i+1] = sumX[j][i] + l
			sumY[i][j+1] = sumY[i][j] + l
		}
	}
	band := max(1, int(edgeBand*float64(max(cols, rows))))
	margin := int(cropMargin * float64(max(cols, rows)))
	// edge finds the strongest line near at (a share of n samples), across
	// from..to (shares of m samples along the line); sums holds the
	// prefix sums across the line. With ends, the two ends of the line,
	// next to the picture's corners, also count on their own.
	edge := func(at float64, n int, from, to float64, m int, sums func(t int) []float64, outsideFirst, ends bool) float64 {
		center := int(at * float64(n))
		a, z := max(0, int(from*float64(m))), min(m, int(to*float64(m)))
		if z-a < 10 {
			return at
		}
		inset := (z - a) / 10
		spans := [][2]int{{a + inset, z - inset}}
		if ends {
			spans = append(spans, [2]int{a, a + inset}, [2]int{z - inset, z})
		}
		lo, hi := max(band, center-margin), min(n-band, center+margin)
		if hi < lo {
			return at
		}
		scores := make([]float64, hi-lo+1)
		diffs := make([]float64, z-a)
		top := 0.0
		for pos := lo; pos <= hi; pos++ {
			for t := a; t < z; t++ {
				s := sums(t)
				before := (s[pos] - s[pos-band]) / float64(band)
				after := (s[pos+band] - s[pos]) / float64(band)
				if outsideFirst {
					diffs[t-a] = before - after
				} else {
					diffs[t-a] = after - before
				}
			}
			score := 0.0
			for _, sp := range spans {
				score = max(score, percentile(diffs[sp[0]-a:sp[1]-a], 0.5))
			}
			scores[pos-lo] = score
			top = max(top, score)
		}
		if top < minEdge {
			return at
		}
		// A big caption or a text column beside a picture can make an edge
		// as strong as the picture's own, so of the strong peaks the one
		// nearest the model's line wins.
		best, bestDist := center, math.MaxInt
		for k, sc := range scores {
			if sc < max(minEdge, top/2) {
				continue
			}
			peak := true
			for d := max(0, k-band/2); d <= min(len(scores)-1, k+band/2); d++ {
				if scores[d] > sc {
					peak = false
					break
				}
			}
			if dist := abs(lo + k - center); peak && dist < bestDist {
				best, bestDist = lo+k, dist
			}
		}
		return float64(best) / float64(n)
	}
	across := func(t int) []float64 { return sumX[t] }
	down := func(t int) []float64 { return sumY[t] }
	// The first pass fits each side along the model's box. A film can be
	// as light as paper along most of an edge, such as a skull at the top
	// of a neck film, with only its dark corners showing the edge, so the
	// second pass also scores the ends of each side next to the corners
	// the first pass found.
	fitSides := func(along domain.Rect, ends bool) domain.Rect {
		return domain.Rect{
			X0: edge(box.X0, cols, along.Y0, along.Y1, rows, across, true, ends),
			X1: edge(box.X1, cols, along.Y0, along.Y1, rows, across, false, ends),
			Y0: edge(box.Y0, rows, along.X0, along.X1, cols, down, true, ends),
			Y1: edge(box.Y1, rows, along.X0, along.X1, cols, down, false, ends),
		}.Clip()
	}
	fit := fitSides(fitSides(box, false), true)
	boxArea := (box.X1 - box.X0) * (box.Y1 - box.Y0)
	if !fit.Large() || (fit.X1-fit.X0)*(fit.Y1-fit.Y0) < minKept*boxArea {
		return box
	}
	return fit
}

// edgeBand is the width of the bands compared on either side of a
// candidate edge, as a share of the page's longer side.
const edgeBand = 0.012

// minEdge is the least median change in brightness, out of 255, across a
// line for it to count as a picture's edge.
const minEdge = 8

// paperness is high for light grey paper and lower for anything darker or
// more colourful: brightness less colourfulness, both out of 255.
func paperness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r>>8), float64(g>>8), float64(b>>8)
	return 0.299*rf + 0.587*gf + 0.114*bf - (max(rf, gf, bf) - min(rf, gf, bf))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func percentile(values []float64, p float64) float64 {
	s := slices.Clone(values)
	slices.Sort(s)
	return s[min(len(s)-1, int(p*float64(len(s))))]
}
