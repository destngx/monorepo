package service

import (
	"errors"
	"strings"
	"testing"

	"apps/health-watcher/domain"
)

type fakeQR struct{}

func (fakeQR) PNG(text string) ([]byte, error) { return []byte("png:" + text), nil }

func token(t *testing.T, link string) string {
	t.Helper()
	_, raw, ok := strings.Cut(link, "#")
	if !ok || !strings.HasPrefix(raw, "hwu_") {
		t.Fatalf("link = %q", link)
	}
	return raw
}

func TestGivenTheAppOpenedOnTheTailnetWhenStartingACaptureThenTheLinkUsesThatAddressAndAFragment(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	link, err := e.capture.Start(e.ctx, sc, "http://100.64.0.7:5174", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link.URL, "http://100.64.0.7:5174/ui/capture#hwu_") || !strings.HasPrefix(link.QR, "data:image/png;base64,") ||
		!link.ExpiresAt.Equal(e.now.Add(CaptureLifetime)) {
		t.Fatalf("link = %+v", link)
	}

	// The link uploads to this tenant's inbox, tagged as from the phone.
	up, err := e.accounts.UploadScope(e.ctx, token(t, link.URL))
	if err != nil || up.TenantID != sc.TenantID || up.Credential != domain.CredentialCaptureLink {
		t.Fatalf("scope = %+v, %v", up, err)
	}
	// Capture links stay out of the Shortcut token list.
	if toks, err := e.tenants.UploadTokens(e.ctx, sc); err != nil || len(toks) != 0 {
		t.Fatalf("tokens = %+v, %v", toks, err)
	}
}

func TestGivenACaptureLinkWhenItExpiresThenItNoLongerUploadsAndIsCleanedUp(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	link, err := e.capture.Start(e.ctx, sc, "http://100.64.0.7:9191", "")
	if err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(CaptureLifetime)
	if _, err := e.accounts.UploadScope(e.ctx, token(t, link.URL)); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("an expired link uploads: %v", err)
	}
	if _, err := e.capture.Start(e.ctx, sc, "http://100.64.0.7:9191", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.UploadTokenByHash(e.ctx, HashToken(token(t, link.URL))); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expired link kept: %v", err)
	}
}

func TestGivenTheAppOpenedOnTheMacItselfWhenStartingACaptureThenTheEngineAddressIsUsedOrItExplains(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	if _, err := e.capture.Start(e.ctx, sc, "http://127.0.0.1:9191", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a loopback-only engine made a link: %v", err)
	}
	e.engine.StartedOn("100.64.0.7", 9191)
	// The phone opens the same server the Mac's browser uses: the engine,
	// or the UI dev server, which listens on every address.
	for origin, want := range map[string]string{
		"http://127.0.0.1:9191": "http://100.64.0.7:9191/ui/capture#",
		"http://localhost:5174": "http://100.64.0.7:5174/ui/capture#",
		"http://localhost":      "http://100.64.0.7:9191/ui/capture#",
	} {
		link, err := e.capture.Start(e.ctx, sc, origin, "")
		if err != nil || !strings.HasPrefix(link.URL, want) {
			t.Errorf("%s: link = %+v, %v", origin, link, err)
		}
	}
	e.engine.StartedOn("0.0.0.0", 9191)
	if _, err := e.capture.Start(e.ctx, sc, "http://127.0.0.1:9191", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a link to 0.0.0.0: %v", err)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "http://100.64.0.7/ui/inbox", "ftp://100.64.0.7"} {
		if _, err := e.capture.Start(e.ctx, sc, bad, ""); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
}

func TestGivenAViewerWhenStartingACaptureThenItIsForbidden(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	viewer := e.join(owner, "viewer@example.test", "viewer")
	if _, err := e.capture.Start(e.ctx, viewer, "http://100.64.0.7:9191", ""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenACaptureLinkMadeOnAnItemWhenThePhoneSendsThenThePhotosJoinThatItemOnly(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")
	other := e.upload(sc, false, pdf("other")).Items[0]

	link, err := e.capture.Start(e.ctx, sc, "http://100.64.0.7:9191", d.Item.ID)
	if err != nil {
		t.Fatal(err)
	}
	phone, err := e.accounts.UploadScope(e.ctx, token(t, link.URL))
	if err != nil || phone.ItemID != d.Item.ID {
		t.Fatalf("scope = %+v, %v", phone, err)
	}

	resp, err := e.inbox.AddFiles(e.ctx, phone, d.Item.ID, []domain.Upload{pdf("xray")})
	if err != nil || len(resp.Items[0].Files) != 2 {
		t.Fatalf("add = %+v, %v", resp, err)
	}
	if _, err := e.inbox.AddFiles(e.ctx, phone, other.ID, []domain.Upload{pdf("x2")}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("added to another item: %v", err)
	}
	if _, err := e.inbox.Upload(e.ctx, phone, UploadRequest{Source: domain.SourcePhone, Files: []domain.Upload{pdf("x3")}}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("started a new item: %v", err)
	}

	if err := e.inbox.Discard(e.ctx, sc, other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.capture.Start(e.ctx, sc, "http://100.64.0.7:9191", other.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("link for a discarded item: %v", err)
	}
}
