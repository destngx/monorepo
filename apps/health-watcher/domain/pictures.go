package domain

import (
	"math"
	"time"
)

// PictureKind is what a picture printed on a page shows (HW-READ-11).
type PictureKind string

// PictureKinds lists every picture kind in display order.
var PictureKinds = []PictureKind{"xray", "ultrasound", "endoscopy", "ecg", "ct", "mri", "photo", "diagram", "other"}

// ParsePictureKind validates a picture kind; an unknown kind is other.
func ParsePictureKind(s string) PictureKind {
	for _, k := range PictureKinds {
		if string(k) == s {
			return k
		}
	}
	return "other"
}

// Rect is an axis-aligned box on a page image, 0-1 from the top-left corner.
type Rect struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

// MinPictureSide is the smallest side of a picture, as a share of the
// page's side: anything smaller is a logo or a stamp.
const MinPictureSide = 0.04

// Clip returns the box inside the page, with its corners in order.
func (r Rect) Clip() Rect {
	clip := func(v float64) float64 {
		if math.IsNaN(v) {
			return 0
		}
		return max(0, min(1, v))
	}
	x0, x1 := clip(r.X0), clip(r.X1)
	y0, y1 := clip(r.Y0), clip(r.Y1)
	return Rect{X0: min(x0, x1), Y0: min(y0, y1), X1: max(x0, x1), Y1: max(y0, y1)}
}

// Large reports whether both sides are at least MinPictureSide.
func (r Rect) Large() bool {
	return r.X1-r.X0 >= MinPictureSide && r.Y1-r.Y0 >= MinPictureSide
}

// Picture is one picture printed on a page, such as an X-ray film or an
// ECG trace, cut out of the page image and kept as its own image
// (HW-READ-11, FHIR Media). It is shown, never read with AI (D5).
type Picture struct {
	ID       string      `json:"id"`
	PageID   string      `json:"pageId"`
	Position int         `json:"position"`
	Kind     PictureKind `json:"kind"`
	// Caption is the text printed with the picture, such as `Tai: Màng nhĩ
	// bình thường` under an endoscopy photo; "" when there is none.
	Caption   string    `json:"caption"`
	Box       Rect      `json:"box"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	ImageSHA  string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// ReadingPicture is a picture the model found on a page: where it is and
// what kind it is, never what it shows.
type ReadingPicture struct {
	Page    int     `json:"page"`
	Kind    string  `json:"kind"`
	Caption string  `json:"caption"`
	X0      float64 `json:"x0"`
	Y0      float64 `json:"y0"`
	X1      float64 `json:"x1"`
	Y1      float64 `json:"y1"`
}

// PictureInput is a cleaned ReadingPicture, before it is cut out.
type PictureInput struct {
	Page    int
	Kind    PictureKind
	Caption string
	Box     Rect
}

// MaxReadPictures bounds the pictures one reading may propose.
const MaxReadPictures = 40

// MaxPictureCaption bounds a picture's caption in characters.
const MaxPictureCaption = 200

// CleanPictures keeps the pictures on a sent page whose box is large enough
// once clipped to the page, in the order given.
func (o ReadingOutput) CleanPictures(pageCount int) []PictureInput {
	var out []PictureInput
	for _, p := range o.Pictures {
		if len(out) == MaxReadPictures {
			break
		}
		box := Rect{X0: p.X0, Y0: p.Y0, X1: p.X1, Y1: p.Y1}.Clip()
		if p.Page < 1 || p.Page > pageCount || !box.Large() {
			continue
		}
		out = append(out, PictureInput{Page: p.Page, Kind: ParsePictureKind(p.Kind), Caption: cutLines(p.Caption, MaxPictureCaption), Box: box})
	}
	return out
}
