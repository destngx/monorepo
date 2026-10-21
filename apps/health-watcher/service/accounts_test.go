package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"apps/health-watcher/domain"
)

func TestGivenAnEmptyDatabaseWhenSettingUpThenOwnerTenantAndPersonExist(t *testing.T) {
	e := newEnv(t)
	if need, err := e.accounts.NeedsSetup(e.ctx); err != nil || !need {
		t.Fatalf("NeedsSetup = %v, %v", need, err)
	}
	owner := e.setup()
	if owner.Role != domain.RoleOwner {
		t.Fatalf("role = %s", owner.Role)
	}
	people, err := e.records.People(e.ctx, owner)
	if err != nil || len(people) != 1 || people[0].UserID != owner.UserID || people[0].DisplayName != "Owner Test" {
		t.Fatalf("people = %+v, %v", people, err)
	}
	if code, err := e.accounts.PrepareSetup(e.ctx); err != nil || code != "" {
		t.Fatalf("PrepareSetup after setup = %q, %v", code, err)
	}
	_, err = e.accounts.Setup(e.ctx, SetupInput{Email: "x@example.test", DisplayName: "X", Password: testPassword, TenantName: "T"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("second setup = %v, want forbidden", err)
	}
}

func TestGivenAWrongOrMissingSetupCodeWhenSettingUpThenItIsForbidden(t *testing.T) {
	e := newEnv(t)
	in := SetupInput{Email: "o@example.test", DisplayName: "O", Password: testPassword, TenantName: "T"}
	if _, err := e.accounts.Setup(e.ctx, in); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("setup before a code exists = %v", err)
	}
	code, err := e.accounts.PrepareSetup(e.ctx)
	if err != nil || len(code) != 14 {
		t.Fatalf("PrepareSetup = %q, %v", code, err)
	}
	in.SetupCode = "0000-0000-0000"
	if _, err := e.accounts.Setup(e.ctx, in); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("wrong code = %v", err)
	}
	in.SetupCode = " " + strings.ToLower(code) + " "
	if _, err := e.accounts.Setup(e.ctx, in); err != nil {
		t.Fatalf("right code typed loosely = %v", err)
	}
}

func TestGivenWrongPasswordsWhenLoggingInThenTheEmailLocksAfterFiveFailures(t *testing.T) {
	e := newEnv(t)
	e.setup()
	for i := 0; i < domain.LoginMaxFailures; i++ {
		if _, err := e.accounts.Login(e.ctx, "owner@example.test", "wrong password!"); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("attempt %d = %v", i, err)
		}
	}
	if _, err := e.accounts.Login(e.ctx, "OWNER@example.test", testPassword); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("locked login = %v", err)
	}
	e.now = e.now.Add(domain.LoginLockWindow + time.Second)
	if _, err := e.accounts.Login(e.ctx, "owner@example.test", testPassword); err != nil {
		t.Fatalf("login after window = %v", err)
	}
}

func TestGivenAnUnknownEmailWhenLoggingInThenItLooksLikeAWrongPassword(t *testing.T) {
	e := newEnv(t)
	e.setup()
	if _, err := e.accounts.Login(e.ctx, "nobody@example.test", testPassword); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("unknown email = %v", err)
	}
}

func TestGivenASessionWhenTimePassesThenItExtendsOnUseAndExpiresWhenIdle(t *testing.T) {
	e := newEnv(t)
	e.setup()
	in, err := e.accounts.Login(e.ctx, "owner@example.test", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(10 * 24 * time.Hour)
	s, err := e.accounts.Authenticate(e.ctx, in.Token)
	if err != nil || !s.ExpiresAt.Equal(e.now.Add(domain.SessionLifetime)) {
		t.Fatalf("Authenticate = %+v, %v", s, err)
	}
	e.now = e.now.Add(domain.SessionLifetime)
	if _, err := e.accounts.Authenticate(e.ctx, in.Token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expired session = %v", err)
	}
	if _, err := e.accounts.Authenticate(e.ctx, "hws_made_up"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("unknown session = %v", err)
	}
}

func TestGivenTwoSessionsWhenChangingPasswordThenOnlyTheNewSessionWorks(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	other, err := e.accounts.Login(e.ctx, "owner@example.test", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.ChangePassword(e.ctx, owner.UserID, "not the password", "a new long password"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("wrong current = %v", err)
	}
	fresh, err := e.accounts.ChangePassword(e.ctx, owner.UserID, testPassword, "a new long password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.Authenticate(e.ctx, other.Token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("old session = %v", err)
	}
	if _, err := e.accounts.Authenticate(e.ctx, fresh.Token); err != nil {
		t.Fatalf("new session = %v", err)
	}
	if _, err := e.accounts.Login(e.ctx, "owner@example.test", "a new long password"); err != nil {
		t.Fatalf("login with new password = %v", err)
	}
}

func TestGivenAForgottenPasswordWhenResettingFromTheCommandThenLoginWorksAgain(t *testing.T) {
	e := newEnv(t)
	e.setup()
	if err := e.accounts.ResetPassword(e.ctx, "owner@example.test", "reset long password"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.Login(e.ctx, "owner@example.test", "reset long password"); err != nil {
		t.Fatalf("login = %v", err)
	}
}

func TestGivenAnUploadTokenWhenItsOwnerIsDemotedOrItIsRevokedThenUploadsStop(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	editor := e.join(owner, "editor@example.test", "editor")
	tok, err := e.tenants.CreateUploadToken(e.ctx, editor, "iPhone Shortcut")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := e.accounts.UploadScope(e.ctx, tok.Token)
	if err != nil || sc.TenantID != owner.TenantID || sc.CanRead() || !sc.CanUpload() {
		t.Fatalf("UploadScope = %+v, %v", sc, err)
	}
	if _, err := e.records.People(e.ctx, sc); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("token read = %v", err)
	}
	if err := e.tenants.SetRole(e.ctx, owner, editor.UserID, "viewer"); err != nil {
		t.Fatal(err)
	}
	sc, _ = e.accounts.UploadScope(e.ctx, tok.Token)
	if _, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Files: []domain.Upload{pdf("a")}}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer token upload = %v", err)
	}
	if err := e.tenants.RevokeUploadToken(e.ctx, e.scope(editor.UserID, owner.TenantID), tok.UploadToken.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.UploadScope(e.ctx, tok.Token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("revoked token = %v", err)
	}
}

func TestGivenParallelWrongPasswordsWhenLoggingInThenTheLimitStillHolds(t *testing.T) {
	e := newEnv(t)
	e.setup()
	const attempts = 20
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			_, err := e.accounts.Login(e.ctx, "owner@example.test", "wrong password!")
			errs <- err
		}()
	}
	limited := 0
	for i := 0; i < attempts; i++ {
		if errors.Is(<-errs, domain.ErrRateLimited) {
			limited++
		}
	}
	if limited < attempts-domain.LoginMaxFailures {
		t.Fatalf("only %d of %d parallel attempts were rate limited", limited, attempts)
	}
}
