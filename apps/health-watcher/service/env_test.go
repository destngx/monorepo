package service

import (
	"context"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"apps/health-watcher/adapter/filestore"
	"apps/health-watcher/adapter/imaging"
	"apps/health-watcher/adapter/password"
	"apps/health-watcher/adapter/sqlite"
	"apps/health-watcher/domain"
)

// env is a full service stack on a temporary SQLite file and file store.
type env struct {
	t        *testing.T
	ctx      context.Context
	now      time.Time
	store    *sqlite.Store
	files    *filestore.Store
	accounts *Accounts
	tenants  *Tenants
	records  *Records
	inbox    *Inbox
	clinical *Clinical
	engine   *Engine
	pipeline *Pipeline
	ocr      *fakeOCR
	reader   *fakeReader
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlite.Open(ctx, filepath.Join(dir, "health.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	files, err := filestore.New(filepath.Join(dir, "files"))
	if err != nil {
		t.Fatal(err)
	}
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, ctx: ctx, now: time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC), store: store, files: files}
	deps := Deps{Store: store, Files: files, Hasher: password.New(password.Params{MemoryKiB: 64, Passes: 1, Lanes: 1}),
		Clock: func() time.Time { return e.now }, Location: loc, FileLock: &sync.Mutex{}}
	if e.accounts, err = NewAccounts(deps); err != nil {
		t.Fatal(err)
	}
	e.tenants = &Tenants{Deps: deps, Accounts: e.accounts}
	e.records = &Records{Deps: deps}
	e.ocr, e.reader = &fakeOCR{}, &fakeReader{}
	e.pipeline = &Pipeline{Deps: deps, OCR: e.ocr, Masker: imaging.Masker{Quality: 90}, Reader: e.reader,
		Catalog: domain.DefaultCatalog, WorkDir: filepath.Join(dir, "work"), Log: slog.New(slog.DiscardHandler)}
	if err := os.MkdirAll(e.pipeline.WorkDir, 0o700); err != nil {
		t.Fatal(err)
	}
	e.inbox = &Inbox{Deps: deps, Records: e.records, Limits: DefaultLimits, Pipeline: e.pipeline}
	e.clinical = &Clinical{Deps: deps, Catalog: domain.DefaultCatalog}
	e.engine = &Engine{Deps: deps, Log: slog.New(slog.DiscardHandler), DefaultMacOCRBin: filepath.Join(dir, "no-mac-ocr"),
		LocalAddrs: func() ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1"), netip.MustParseAddr("100.64.0.7")}, nil
		}}
	if _, _, err := e.engine.Load(ctx, nil); err != nil {
		t.Fatal(err)
	}
	return e
}

const testPassword = "a long test password"

// setup runs first-run setup and returns the owner's scope in their tenant.
func (e *env) setup() domain.Scope {
	e.t.Helper()
	code, err := e.accounts.PrepareSetup(e.ctx)
	if err != nil {
		e.t.Fatal(err)
	}
	in, err := e.accounts.Setup(e.ctx, SetupInput{Email: "owner@example.test", DisplayName: "Owner Test",
		Password: testPassword, TenantName: "Test Family", SetupCode: code})
	if err != nil {
		e.t.Fatalf("setup: %v", err)
	}
	return e.scope(in.Me.User.ID, in.Me.Memberships[0].Tenant.ID)
}

func (e *env) scope(userID, tenantID string) domain.Scope {
	e.t.Helper()
	sc, err := e.accounts.ScopeFor(e.ctx, userID, tenantID)
	if err != nil {
		e.t.Fatalf("scope: %v", err)
	}
	return sc
}

// join invites a new user with a role into the scope's tenant.
func (e *env) join(owner domain.Scope, email, role string) domain.Scope {
	e.t.Helper()
	inv, err := e.tenants.CreateInvite(e.ctx, owner, role)
	if err != nil {
		e.t.Fatalf("invite: %v", err)
	}
	in, err := e.tenants.AcceptInviteAsNewUser(e.ctx, inv.Token, SignUpInput{Email: email, DisplayName: "Member", Password: testPassword})
	if err != nil {
		e.t.Fatalf("accept: %v", err)
	}
	return e.scope(in.Me.User.ID, owner.TenantID)
}

func (e *env) person(sc domain.Scope, name string) domain.Person {
	e.t.Helper()
	p, err := e.records.CreatePerson(e.ctx, sc, domain.PersonInput{DisplayName: name})
	if err != nil {
		e.t.Fatalf("create person: %v", err)
	}
	return p
}

// pdf returns distinct synthetic PDF bytes.
func pdf(label string) domain.Upload {
	return domain.Upload{Name: label + ".pdf", Data: []byte("%PDF-1.7\n% synthetic " + label + "\n")}
}

func (e *env) upload(sc domain.Scope, oneItem bool, files ...domain.Upload) UploadResponse {
	e.t.Helper()
	resp, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourceWeb, Files: files, OneItem: oneItem})
	if err != nil {
		e.t.Fatalf("upload: %v", err)
	}
	return resp
}
