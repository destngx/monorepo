package domain

import (
	"math"
	"time"
)

// Point is a position on a page image, 0-1 from the top-left corner.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Quad is a possibly rotated box: top-left, top-right, bottom-right,
// bottom-left, in page coordinates.
type Quad [4]Point

// RectQuad is the quad of an axis-aligned box.
func RectQuad(x, y, w, h float64) Quad {
	return Quad{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}}
}

// Valid reports whether every corner is a finite point within the page,
// allowing a little overhang.
func (q Quad) Valid() bool {
	for _, p := range q {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) || p.X < -0.05 || p.X > 1.05 || p.Y < -0.05 || p.Y > 1.05 {
			return false
		}
	}
	return true
}

// Bounds returns the axis-aligned box around the quad.
func (q Quad) Bounds() (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = q[0].X, q[0].Y, q[0].X, q[0].Y
	for _, p := range q[1:] {
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
		maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
	}
	return
}

// Expand grows the quad outward along its own axes by frac of its height on
// every side. width and height are the page size in pixels, so the growth
// is even on a page that is not square.
func (q Quad) Expand(frac float64, width, height int) Quad {
	if width <= 0 || height <= 0 {
		return q
	}
	w, h := float64(width), float64(height)
	px := func(p Point) [2]float64 { return [2]float64{p.X * w, p.Y * h} }
	tl, tr, bl := px(q[0]), px(q[1]), px(q[3])
	ux, uy := tr[0]-tl[0], tr[1]-tl[1]
	vx, vy := bl[0]-tl[0], bl[1]-tl[1]
	ul, vl := math.Hypot(ux, uy), math.Hypot(vx, vy)
	if ul == 0 || vl == 0 {
		return q
	}
	pad := frac * vl
	ux, uy, vx, vy = ux/ul*pad, uy/ul*pad, vx/vl*pad, vy/vl*pad
	signs := [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}
	var out Quad
	for i, p := range q {
		c := px(p)
		out[i] = Point{
			X: (c[0] + signs[i][0]*ux + signs[i][1]*vx) / w,
			Y: (c[1] + signs[i][0]*uy + signs[i][1]*vy) / h,
		}
	}
	return out
}

// Line is one line of text on a page with its box.
type Line struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Quad       Quad    `json:"quad"`
}

// PageMethod says how a page's text was read.
type PageMethod string

const (
	MethodText PageMethod = "text" // the PDF's own text layer
	MethodOCR  PageMethod = "ocr"
)

// Page is one page of a stored file: its text, its rendered image and the
// local mask made before any AI call (HW-READ-1, HW-READ-2, HW-AI-7).
type Page struct {
	ID         string     `json:"id"`
	FileID     string     `json:"fileId"`
	Number     int        `json:"number"`
	Method     PageMethod `json:"method"`
	Text       string     `json:"text"`
	Confidence float64    `json:"confidence"`
	Width      int        `json:"width"`
	Height     int        `json:"height"`
	// ImageSHA is the rendered JPEG in the file store, "" when the OCR tool
	// could not render the page.
	ImageSHA string `json:"-"`
	HasImage bool   `json:"hasImage"`
	Lines    []Line `json:"-"`
	Barcodes []Quad `json:"-"`
	// BarcodesChecked is false when the OCR tool could not look for barcodes.
	BarcodesChecked bool      `json:"-"`
	Mask            *PageMask `json:"mask,omitempty"`
	CreatedAt       time.Time `json:"-"`
}

// MaskStatus says whether a masked page may be sent to the AI provider.
type MaskStatus string

const (
	// MaskReady pages are sent without a check.
	MaskReady MaskStatus = "ready"
	// MaskNeedsCheck pages wait until a person checks the masked preview.
	MaskNeedsCheck MaskStatus = "needs_check"
	// MaskChecked pages were checked by a person and may be sent.
	MaskChecked MaskStatus = "checked"
)

// PageMask is what the AI provider gets for a page: the image with black
// boxes and the text with personal lines replaced (HW-AI-7).
type PageMask struct {
	Status      MaskStatus `json:"status"`
	Reasons     []string   `json:"reasons"`
	Boxes       []Quad     `json:"boxes"`
	Manual      []Quad     `json:"manual"`
	MaskedLines int        `json:"maskedLines"`
	// Text is the masked text, shown next to the original on the review
	// page and in the viewer.
	Text     string `json:"text"`
	ImageSHA string `json:"-"`
}
