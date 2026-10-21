package macocr

import (
	"math"
	"testing"
)

const v11 = `{"schema_version":"1.1","pages":[{"index":1,"method":"ocr","content":"Họ tên: A\nGlucose 6.4","confidence":1,
 "bbox":[{"text":"Họ tên: A","confidence":1,"x":0.1,"y":0.8,"w":0.5,"h":0.02}]}],"summary":{}}`

const v12 = `{"schema_version":"1.2","pages":[{"index":1,"method":"direct","content":"x","confidence":1,
 "bbox":[{"text":"x","confidence":0.5,"x":0.1,"y":0.8,"w":0.5,"h":0.02,"quad":[[0.1,0.82],[0.6,0.82],[0.6,0.8],[0.1,0.8]]}],
 "barcodes":[{"symbology":"QR","quad":[[0.8,0.2],[0.9,0.2],[0.9,0.1],[0.8,0.1]]}],
 "image":{"path":"/w/page-1.jpg","width":1654,"height":2339}}]}`

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestGivenSchema11OutputWhenParsingThenBoxesFlipToTopLeftAndNothingElseIsClaimed(t *testing.T) {
	pages, err := Parse([]byte(v11))
	if err != nil {
		t.Fatal(err)
	}
	p := pages[0]
	if p.BarcodesChecked || p.ImagePath != "" || len(p.Lines) != 1 {
		t.Fatalf("page = %+v", p)
	}
	q := p.Lines[0].Quad
	// Bottom-left box at y=0.8 with h=0.02 has its top at 0.82, i.e. 0.18 from the top.
	if !near(q[0].X, 0.1) || !near(q[0].Y, 0.18) || !near(q[2].X, 0.6) || !near(q[2].Y, 0.2) {
		t.Fatalf("quad = %+v", q)
	}
}

func TestGivenSchema12OutputWhenParsingThenQuadsBarcodesAndImagesAreRead(t *testing.T) {
	pages, err := Parse([]byte(v12))
	if err != nil {
		t.Fatal(err)
	}
	p := pages[0]
	if p.Method != "text" || !p.BarcodesChecked || len(p.Barcodes) != 1 || p.ImagePath != "/w/page-1.jpg" || p.Width != 1654 {
		t.Fatalf("page = %+v", p)
	}
	if q := p.Lines[0].Quad; !near(q[0].Y, 0.18) || !near(q[3].Y, 0.2) {
		t.Fatalf("quad = %+v", q)
	}
	if b := p.Barcodes[0]; !near(b[0].Y, 0.8) || !near(b[2].Y, 0.9) {
		t.Fatalf("barcode = %+v", b)
	}
}

func TestGivenEveryPageFailedWhenParsingThenItIsAnError(t *testing.T) {
	if _, err := Parse([]byte(`{"pages":[{"index":1,"content":"","error":"timeout"}]}`)); err == nil {
		t.Fatal("no error")
	}
	if _, err := Parse([]byte(`{"pages":[]}`)); err == nil {
		t.Fatal("no error for no pages")
	}
}
