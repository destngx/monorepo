// Package qrcode draws QR codes with rsc.io/qr (HW-IMP-9).
package qrcode

import (
	"fmt"

	"rsc.io/qr"

	"apps/health-watcher/port"
)

// Coder draws QR codes at medium error correction.
type Coder struct{}

var _ port.QRCoder = Coder{}

// PNG returns text as a QR code image.
func (Coder) PNG(text string) ([]byte, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, fmt.Errorf("draw QR code: %w", err)
	}
	code.Scale = 8
	return code.PNG(), nil
}
