package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"apps/health-watcher/domain"
)

func settingsInput(s domain.Settings) SettingsInput {
	return SettingsInput{Host: s.Host, Port: s.Port, MaxFileMB: s.MaxFileMB, MaxBatchMB: s.MaxBatchMB, MacOCRBin: s.MacOCRBin,
		Workers: s.Workers, AIBaseURL: s.AIBaseURL, AITimeoutSeconds: s.AITimeoutSeconds}
}

func ptr[T any](v T) *T { return &v }

func TestGivenTheSetupAccountWhenReadingEngineSettingsThenItIsAdminAndMembersAreNot(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	member := e.join(owner, "member@example.test", "owner")

	v, err := e.engine.View(e.ctx, owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.Settings, domain.DefaultSettings()) || v.AIModel != domain.AIModel || v.AIEffort != domain.AIEffort ||
		v.Listening != "127.0.0.1:9191" || v.MacOCRFound || len(v.RestartFields) != 0 || v.Stamp.UpdatedAt != nil {
		t.Fatalf("view = %+v", v)
	}
	if _, err := e.engine.View(e.ctx, member.UserID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a family owner who is not admin sees engine settings: %v", err)
	}
	if _, err := e.engine.Update(e.ctx, member.UserID, settingsInput(domain.DefaultSettings())); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a non-admin changes engine settings: %v", err)
	}
	if _, err := e.engine.Users(e.ctx, member.UserID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a non-admin lists accounts: %v", err)
	}
}

func TestGivenAnAdminWhenSavingSettingsThenTheyAreStoredAppliedAndTheKeyStaysHidden(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	var applied []domain.Settings
	e.engine.Apply = func(s domain.Settings) { applied = append(applied, s) }

	in := settingsInput(domain.DefaultSettings())
	in.Workers, in.AIBaseURL, in.AIAPIKey, in.MaxFileMB = 4, "http://localhost:8080/v1/", ptr("sk-test-secret"), 10
	v, err := e.engine.Update(e.ctx, owner.UserID, in)
	if err != nil {
		t.Fatal(err)
	}
	if v.Settings.Workers != 4 || v.Settings.AIBaseURL != "http://localhost:8080/v1" || !v.AIAPIKeySet || len(v.RestartFields) != 0 ||
		v.Stamp.UpdatedAt == nil || v.Stamp.UpdatedBy != owner.UserID || v.Stamp.UpdatedByName != "Owner Test" {
		t.Fatalf("view = %+v", v)
	}
	if len(applied) != 1 || applied[0].AIAPIKey != "sk-test-secret" || applied[0].Workers != 4 {
		t.Fatalf("applied = %+v", applied)
	}
	raw, _ := json.Marshal(v)
	if strings.Contains(string(raw), "sk-test-secret") {
		t.Fatalf("the API key reached the view: %s", raw)
	}

	// A later save without a key keeps it; "" removes it.
	in.AIAPIKey, in.Workers = nil, 3
	if _, err := e.engine.Update(e.ctx, owner.UserID, in); err != nil {
		t.Fatal(err)
	}
	if e.engine.Current().AIAPIKey != "sk-test-secret" {
		t.Fatal("saving without a key dropped it")
	}

	// Another engine on the same database reads what was saved.
	fresh := &Engine{Deps: e.engine.Deps, Log: slog.New(slog.DiscardHandler)}
	s, _, err := fresh.Load(e.ctx, nil)
	if err != nil || s.Workers != 3 || s.AIAPIKey != "sk-test-secret" || s.MaxFileMB != 10 {
		t.Fatalf("loaded = %+v, %v", s, err)
	}

	in.AIAPIKey = ptr("")
	if v, err = e.engine.Update(e.ctx, owner.UserID, in); err != nil || v.AIAPIKeySet || e.engine.Current().AIAPIKey != "" {
		t.Fatalf("key not removed: %+v, %v", v, err)
	}

	// Saving the same values changes nothing.
	n := len(applied)
	if _, err := e.engine.Update(e.ctx, owner.UserID, in); err != nil || len(applied) != n {
		t.Fatalf("an unchanged save was applied: %d, %v", len(applied), err)
	}
}

func TestGivenANewHostWhenSavingThenItMustBeAnAddressOfThisMacAndWaitsForRestart(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	in := settingsInput(domain.DefaultSettings())

	in.Host = "192.0.2.10"
	if _, err := e.engine.Update(e.ctx, owner.UserID, in); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a foreign host was saved: %v", err)
	}
	in.Host, in.Port, in.MaxBatchMB = "100.64.0.7", 9292, domain.DefaultMaxBatchMB+100
	v, err := e.engine.Update(e.ctx, owner.UserID, in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.RestartFields, []string{"host", "port", "maxBatchMb"}) || v.Listening != "127.0.0.1:9191" {
		t.Fatalf("view = %+v", v)
	}

	// With Tailscale down the saved host is not local any more; other
	// settings still save.
	e.engine.LocalAddrs = func() ([]netip.Addr, error) { return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil }
	in.Workers = 5
	if _, err := e.engine.Update(e.ctx, owner.UserID, in); err != nil {
		t.Fatalf("an unchanged host blocked the save: %v", err)
	}

	e.engine.StartedOn("100.64.0.7", 9292)
	if v, _ := e.engine.View(e.ctx, owner.UserID); !reflect.DeepEqual(v.RestartFields, []string{"maxBatchMb"}) {
		t.Fatalf("restart fields after start = %v", v.RestartFields)
	}
}

func TestGivenAMacOCRPathWhenSavingThenItMustBeAProgram(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	dir := t.TempDir()
	plain, prog := filepath.Join(dir, "notes.txt"), filepath.Join(dir, "mac-ocr")
	if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prog, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	in := settingsInput(domain.DefaultSettings())
	for _, bad := range []string{filepath.Join(dir, "missing"), plain, dir} {
		in.MacOCRBin = bad
		if _, err := e.engine.Update(e.ctx, owner.UserID, in); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%s: err = %v", bad, err)
		}
	}
	in.MacOCRBin = prog
	v, err := e.engine.Update(e.ctx, owner.UserID, in)
	if err != nil || !v.MacOCRFound || v.Settings.MacOCRBin != prog {
		t.Fatalf("view = %+v, %v", v, err)
	}
}

func TestGivenABadStoredValueWhenLoadingThenOnlyItUsesTheDefaultAndTheNextSaveRewritesIt(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	stored := domain.DefaultSettings()
	stored.AIBaseURL, stored.AIAPIKey, stored.Workers = "http://localhost:8080/v1", "sk-test", 4
	values := stored.Values()
	values["port"] = "not a number"
	if err := e.store.SaveSettings(e.ctx, values, e.now, owner.UserID); err != nil {
		t.Fatal(err)
	}
	s, _, err := e.engine.Load(e.ctx, nil)
	if err != nil || s.AIAPIKey != "sk-test" || s.Workers != 4 || s.Port != 9191 {
		t.Fatalf("loaded = %+v, %v", s, err)
	}
	v, _ := e.engine.View(e.ctx, owner.UserID)
	if !reflect.DeepEqual(v.Rejected, []string{"port"}) {
		t.Fatalf("rejected = %v", v.Rejected)
	}

	// Saving the same values still rewrites the bad one, and keeps the key.
	if v, err = e.engine.Update(e.ctx, owner.UserID, settingsInput(s)); err != nil || len(v.Rejected) != 0 {
		t.Fatalf("save = %+v, %v", v, err)
	}
	if s, _, _ = e.engine.Load(e.ctx, nil); s.AIAPIKey != "sk-test" || s.Port != 9191 {
		t.Fatalf("reloaded = %+v", s)
	}
	if v, _ = e.engine.View(e.ctx, owner.UserID); len(v.Rejected) != 0 {
		t.Fatalf("still rejected: %v", v.Rejected)
	}
}

func TestGivenRetiredEnvironmentValuesWhenNothingIsStoredThenTheyAreImportedOnce(t *testing.T) {
	e := newEnv(t)
	legacy := map[string]string{"ai_base_url": "http://localhost:8080/v1", "ai_api_key": "sk-test", "workers": "3", "port": "80"}
	s, imported, err := e.engine.Load(e.ctx, legacy)
	if err != nil || !imported || s.AIBaseURL != "http://localhost:8080/v1" || s.AIAPIKey != "sk-test" || s.Workers != 3 || s.Port != 9191 {
		t.Fatalf("loaded = %+v, %v, %v", s, imported, err)
	}
	// Once something is stored, the environment is ignored.
	s, imported, err = e.engine.Load(e.ctx, map[string]string{"workers": "5"})
	if err != nil || imported || s.Workers != 3 {
		t.Fatalf("second load = %+v, %v, %v", s, imported, err)
	}
}

func TestGivenAdminsWhenGrantingAndRemovingThenTheLastAdminStays(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	member := e.join(owner, "member@example.test", "viewer")

	if _, err := e.engine.SetAdmin(e.ctx, member.UserID, member.UserID, true); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a non-admin made themselves admin: %v", err)
	}
	if _, err := e.engine.SetAdmin(e.ctx, owner.UserID, owner.UserID, false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("the last admin was removed: %v", err)
	}
	u, err := e.engine.SetAdmin(e.ctx, owner.UserID, member.UserID, true)
	if err != nil || !u.Admin {
		t.Fatalf("grant = %+v, %v", u, err)
	}
	users, err := e.engine.Users(e.ctx, member.UserID)
	if err != nil || len(users) != 2 || !users[0].Admin || !users[1].Admin {
		t.Fatalf("users = %+v, %v", users, err)
	}
	if _, err := e.engine.SetAdmin(e.ctx, member.UserID, owner.UserID, false); err != nil {
		t.Fatalf("an admin could not remove another: %v", err)
	}
	if _, err := e.engine.View(e.ctx, owner.UserID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a removed admin still sees settings: %v", err)
	}
	if _, err := e.engine.SetAdmin(e.ctx, member.UserID, "missing", true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown account: %v", err)
	}
}

func TestGivenAIProviderSettingsWhenCheckingThenTheAnswerIsReported(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	var got domain.Settings
	var pingErr error
	e.engine.PingAI = func(_ context.Context, s domain.Settings) error { got = s; return pingErr }

	if _, err := e.engine.CheckAI(e.ctx, owner.UserID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("checked without an address: %v", err)
	}
	in := settingsInput(domain.DefaultSettings())
	in.AIBaseURL, in.AIAPIKey = "http://localhost:8080/v1", ptr("sk-test")
	if _, err := e.engine.Update(e.ctx, owner.UserID, in); err != nil {
		t.Fatal(err)
	}
	res, err := e.engine.CheckAI(e.ctx, owner.UserID)
	if err != nil || !res.OK || got.AIAPIKey != "sk-test" || !strings.Contains(res.Message, domain.AIModel) {
		t.Fatalf("check = %+v, %v", res, err)
	}
	pingErr = errors.New("AI provider returned 401: bad key")
	if res, err = e.engine.CheckAI(e.ctx, owner.UserID); err != nil || res.OK || res.Message != pingErr.Error() {
		t.Fatalf("failed check = %+v, %v", res, err)
	}
}

func TestGivenARunningPipelineWhenConfiguredThenNewJobsUseTheNewTools(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.pipeline.Configure(e.ocr, nil, 1)
	if e.pipeline.AIEnabled() {
		t.Fatal("AI still on without a reader")
	}
	next := &fakeOCR{}
	e.pipeline.Configure(next, e.reader, 3)
	if !e.pipeline.AIEnabled() {
		t.Fatal("AI off with a reader")
	}
	e.upload(sc, false, pdf("lab"))
	if err := e.pipeline.Drain(e.ctx); err != nil {
		t.Fatal(err)
	}
	if next.calls != 1 || e.ocr.calls != 0 || len(e.reader.calls) != 1 {
		t.Fatalf("new OCR calls = %d, old = %d, reads = %d", next.calls, e.ocr.calls, len(e.reader.calls))
	}
	if n, _ := e.pipeline.workers(); n != 3 {
		t.Fatalf("workers = %d", n)
	}
	e.pipeline.Configure(next, e.reader, 99)
	if n, _ := e.pipeline.workers(); n != domain.MaxWorkers {
		t.Fatalf("workers above the limit = %d", n)
	}
}

func TestGivenSmallerLimitsWhenUploadingThenTheNextUploadUsesThem(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.inbox.SetLimits(Limits{MaxFileBytes: 8, MaxBatchBytes: 8})
	if _, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourceWeb, Files: []domain.Upload{pdf("lab")}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a file over the new limit was accepted: %v", err)
	}
}

func TestGivenQueuedReadingsWhenAIIsTurnedOffThenTheyShowAsOff(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.upload(sc, false, pdf("lab"))
	if ran, err := e.pipeline.RunOnce(e.ctx); err != nil || !ran {
		t.Fatalf("OCR job: %v, %v", ran, err)
	}
	items, err := e.inbox.Items(e.ctx, sc)
	if err != nil || items[0].Reading == nil || items[0].Reading.Status != domain.ReadingQueued {
		t.Fatalf("before = %+v, %v", items[0].Reading, err)
	}
	e.pipeline.Configure(e.ocr, nil, 1)
	if err := e.pipeline.Drain(e.ctx); err != nil {
		t.Fatal(err)
	}
	items, _ = e.inbox.Items(e.ctx, sc)
	if items[0].Reading.Status != domain.ReadingOff || len(e.reader.calls) != 0 {
		t.Fatalf("after = %+v, reads = %d", items[0].Reading, len(e.reader.calls))
	}
}
