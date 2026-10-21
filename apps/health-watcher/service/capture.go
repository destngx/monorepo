package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// CaptureLifetime is how long a phone capture link works.
const CaptureLifetime = 15 * time.Minute

// Capture opens phone capture links (HW-IMP-9): the Mac shows a QR code,
// the phone's camera opens it, and the page there uploads photos to the
// inbox with a short-lived upload token, without signing in.
type Capture struct {
	Deps
	QR port.QRCoder
	// Listening is the engine's address now. It is the link's address when
	// the browser on the Mac uses a loopback address, which a phone cannot
	// open.
	Listening func() (host string, port int)
}

// CaptureLink is a new link, shown once. QR is a PNG data URL of URL.
type CaptureLink struct {
	URL       string    `json:"url"`
	QR        string    `json:"qr"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Start makes a capture link for the user's inbox in this tenant. origin is
// the address the browser opened the app at.
func (c *Capture) Start(ctx context.Context, s domain.Scope, origin string) (CaptureLink, error) {
	if err := s.RequireRead(); err != nil {
		return CaptureLink{}, err
	}
	if err := s.RequireUpload(); err != nil {
		return CaptureLink{}, err
	}
	base, err := c.phoneBase(origin)
	if err != nil {
		return CaptureLink{}, err
	}
	now := c.Clock()
	if err := c.Store.DeleteExpiredUploadTokens(ctx, now); err != nil {
		return CaptureLink{}, err
	}
	raw, err := newToken(prefixUpload)
	if err != nil {
		return CaptureLink{}, err
	}
	expires := now.Add(CaptureLifetime)
	tok := domain.UploadToken{ID: NewID(), Name: "Phone camera", TenantID: s.TenantID, UserID: s.UserID, CreatedAt: now, ExpiresAt: &expires}
	if err := c.Store.CreateUploadToken(ctx, tok, HashToken(raw)); err != nil {
		return CaptureLink{}, fmt.Errorf("create capture link: %w", err)
	}
	// The token rides in the fragment, which browsers never send to a server.
	link := base + "/ui/capture#" + raw
	png, err := c.QR.PNG(link)
	if err != nil {
		return CaptureLink{}, err
	}
	return CaptureLink{URL: link, QR: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), ExpiresAt: expires}, nil
}

// phoneBase is the scheme and host a phone can open: the browser's own when
// it is not a loopback address, otherwise the engine's.
func (c *Capture) phoneBase(origin string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.Trim(u.Path, "/") != "" {
		return "", domain.Invalid("origin must be the app's address, such as http://100.64.0.7:9191")
	}
	if !loopback(u.Hostname()) {
		return u.Scheme + "://" + u.Host, nil
	}
	host, port := c.Listening()
	if !loopback(host) {
		return "http://" + netip.AddrPortFrom(netip.MustParseAddr(host), uint16(port)).String(), nil
	}
	return "", domain.Conflict("the phone cannot reach this Mac at %s; an engine admin sets Host in Engine settings to the Mac's Tailscale address and restarts the engine", host)
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	a, err := netip.ParseAddr(host)
	return err == nil && a.IsLoopback()
}
