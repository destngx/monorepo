// Package macocr runs the mac-ocr tool (apps/mac-ocr) on stored files: text
// from the PDF text layer or Apple Vision, line boxes, barcode boxes and
// rendered page images (HW-READ-1, HW-READ-2, D4).
package macocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Runner calls the mac-ocr binary.
type Runner struct {
	Bin       string
	Languages string
	DPI       int
	// Timeout bounds one file; mac-ocr also limits each page.
	Timeout time.Duration

	once      sync.Once
	canRender bool
	probeErr  error
}

var _ port.OCR = (*Runner)(nil)

// New returns a runner with the engine's defaults.
func New(bin string) *Runner {
	return &Runner{Bin: bin, Languages: "vi-VT,en-US", DPI: 200, Timeout: 10 * time.Minute}
}

// maxStderr bounds the tool's error output kept in a job error.
const maxStderr = 600

// probe checks once whether the binary exists and supports --render-dir
// (schema 1.2). An older build still reads text; its pages have no images,
// so the masking step holds them for a check.
func (r *Runner) probe(ctx context.Context) error {
	r.once.Do(func() {
		out, err := exec.CommandContext(ctx, r.Bin, "--help").CombinedOutput()
		if err != nil && len(out) == 0 {
			r.probeErr = fmt.Errorf("run %s: %w (build it with `pnpx nx build mac-ocr` or set HW_MAC_OCR_BIN)", r.Bin, err)
			return
		}
		r.canRender = bytes.Contains(out, []byte("--render-dir"))
	})
	return r.probeErr
}

// CanRender reports whether the binary renders page images.
func (r *Runner) CanRender(ctx context.Context) (bool, error) {
	err := r.probe(ctx)
	return r.canRender, err
}

// Read runs mac-ocr on path and writes page images to workDir.
func (r *Runner) Read(ctx context.Context, path, workDir string) ([]port.OCRPage, error) {
	if err := r.probe(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	args := []string{"--format", "json", "--bbox", "--lang", r.Languages, "--dpi", strconv.Itoa(r.DPI)}
	if r.canRender {
		args = append(args, "--render-dir", workDir)
	}
	args = append(args, "--", path)
	cmd := exec.CommandContext(ctx, r.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("mac-ocr took longer than %s", r.Timeout)
		}
		return nil, fmt.Errorf("mac-ocr failed: %w: %s", err, clip(stderr.String()))
	}
	return Parse(stdout.Bytes())
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxStderr {
		return s[:maxStderr] + "…"
	}
	return s
}

// wire types of mac-ocr's JSON (schema 1.1 and 1.2).
type wireFile struct {
	SchemaVersion string     `json:"schema_version"`
	Pages         []wirePage `json:"pages"`
}

type wireImage struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type wireBox struct {
	Text       string       `json:"text"`
	Confidence float64      `json:"confidence"`
	X          float64      `json:"x"`
	Y          float64      `json:"y"`
	W          float64      `json:"w"`
	H          float64      `json:"h"`
	Quad       [][2]float64 `json:"quad"`
}

type wireBarcode struct {
	Quad [][2]float64 `json:"quad"`
}

type wirePage struct {
	Index      int            `json:"index"`
	Method     string         `json:"method"`
	Content    string         `json:"content"`
	Confidence float64        `json:"confidence"`
	BBox       []wireBox      `json:"bbox"`
	Barcodes   *[]wireBarcode `json:"barcodes"`
	Image      *wireImage     `json:"image"`
	Error      string         `json:"error"`
}

// flip turns mac-ocr's bottom-left coordinates into top-left ones.
func flip(p [2]float64) domain.Point { return domain.Point{X: p[0], Y: 1 - p[1]} }

func quadOf(raw [][2]float64) (domain.Quad, bool) {
	if len(raw) != 4 {
		return domain.Quad{}, false
	}
	q := domain.Quad{flip(raw[0]), flip(raw[1]), flip(raw[2]), flip(raw[3])}
	return q, q.Valid()
}

// Parse reads mac-ocr's JSON for one file.
func Parse(raw []byte) ([]port.OCRPage, error) {
	var f wireFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("read mac-ocr output: %w", err)
	}
	if len(f.Pages) == 0 {
		return nil, errors.New("mac-ocr found no pages")
	}
	pages := make([]port.OCRPage, 0, len(f.Pages))
	failed := 0
	for i, wp := range f.Pages {
		p := port.OCRPage{Number: wp.Index, Text: wp.Content, Confidence: wp.Confidence, Method: domain.MethodOCR}
		if p.Number <= 0 {
			p.Number = i + 1
		}
		if wp.Method == "direct" || wp.Method == "text" {
			p.Method = domain.MethodText
		}
		if wp.Error != "" {
			failed++
		}
		for _, b := range wp.BBox {
			q, ok := quadOf(b.Quad)
			if !ok {
				// Schema 1.1: the axis-aligned box, origin at its bottom-left.
				q = domain.RectQuad(b.X, 1-(b.Y+b.H), b.W, b.H)
			}
			p.Lines = append(p.Lines, domain.Line{Text: b.Text, Confidence: b.Confidence, Quad: q})
		}
		if wp.Barcodes != nil {
			p.BarcodesChecked = true
			for _, b := range *wp.Barcodes {
				q, ok := quadOf(b.Quad)
				if !ok {
					// A barcode we can't place can't be blacked out.
					p.BarcodesChecked = false
					continue
				}
				p.Barcodes = append(p.Barcodes, q)
			}
		}
		if wp.Image != nil && wp.Image.Path != "" {
			p.ImagePath, p.Width, p.Height = wp.Image.Path, wp.Image.Width, wp.Image.Height
		}
		pages = append(pages, p)
	}
	if failed == len(pages) {
		return nil, fmt.Errorf("mac-ocr could not read any page: %s", clip(f.Pages[0].Error))
	}
	return pages, nil
}
