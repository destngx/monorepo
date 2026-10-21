package qrcode

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGivenALinkWhenDrawnThenItIsAPNGLargeEnoughToScan(t *testing.T) {
	raw, err := Coder{}.PNG("http://100.64.0.7:9191/ui/capture#hwu_abc")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() < 200 || b.Dx() != b.Dy() {
		t.Fatalf("size = %v", b)
	}
}
