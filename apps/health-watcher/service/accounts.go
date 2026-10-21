package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Accounts signs users up, in and out, and resolves requests to a scope
// (HW-AUTH-1..5, HW-TEN-5).
type Accounts struct {
	Deps
	// dummyHash is verified when an email is unknown, so a login takes as
	// long for an unknown email as for a wrong password.
	dummyHash string

	setupMu   sync.Mutex
	setupCode string
}

// NewAccounts builds the accounts service.
func NewAccounts(d Deps) (*Accounts, error) {
	dummy, err := d.Hasher.Hash("not a real password, only for timing")
	if err != nil {
		return nil, fmt.Errorf("prepare login: %w", err)
	}
	return &Accounts{Deps: d, dummyHash: dummy}, nil
}

// SetupInput creates the first user and tenant (HW-AUTH-4).
type SetupInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	TenantName  string `json:"tenantName"`
	// SetupCode is the one-time code from PrepareSetup.
	SetupCode string `json:"setupCode"`
}

// PrepareSetup makes the one-time setup code when no user exists yet, and
// returns it; the engine writes it to a private file in the data directory.
// Reading that file proves access to the Mac, whatever address the request
// comes from (HW-AUTH-4). It returns "" once setup is done.
func (a *Accounts) PrepareSetup(ctx context.Context) (string, error) {
	need, err := a.NeedsSetup(ctx)
	if err != nil || !need {
		return "", err
	}
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("setup code: %w", err)
	}
	code := strings.ToUpper(hex.EncodeToString(b))
	code = code[:4] + "-" + code[4:8] + "-" + code[8:]
	a.setupMu.Lock()
	a.setupCode = code
	a.setupMu.Unlock()
	return code, nil
}

func (a *Accounts) checkSetupCode(got string) error {
	a.setupMu.Lock()
	want := a.setupCode
	a.setupMu.Unlock()
	got = strings.ToUpper(strings.TrimSpace(got))
	if want == "" || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return domain.Forbidden("first-run setup: wrong setup code; it is in the setup-code file in the data directory")
	}
	return nil
}

// SignUpInput creates an account when accepting an invite.
type SignUpInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

// SignedIn is a new session: the raw token for the cookie and who it is.
type SignedIn struct {
	Token string
	Me    domain.Me
}

// NeedsSetup reports whether no user exists yet.
func (a *Accounts) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := a.Store.CountUsers(ctx)
	return n == 0, err
}

func (a *Accounts) newUser(in SignUpInput) (domain.User, string, error) {
	email, err := domain.Email(in.Email)
	if err != nil {
		return domain.User{}, "", err
	}
	name, err := domain.DisplayName(in.DisplayName)
	if err != nil {
		return domain.User{}, "", err
	}
	if err := domain.CheckPassword(in.Password); err != nil {
		return domain.User{}, "", err
	}
	hash, err := a.Hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}
	return domain.User{ID: NewID(), Email: email, DisplayName: name, CreatedAt: a.Clock()}, hash, nil
}

// Setup creates the first user, their tenant, an owner membership and a
// person for the user. It needs the setup code and runs only once.
func (a *Accounts) Setup(ctx context.Context, in SetupInput) (SignedIn, error) {
	if err := a.checkSetupCode(in.SetupCode); err != nil {
		return SignedIn{}, err
	}
	tenantName, err := domain.TenantName(in.TenantName)
	if err != nil {
		return SignedIn{}, err
	}
	user, hash, err := a.newUser(SignUpInput{Email: in.Email, DisplayName: in.DisplayName, Password: in.Password})
	if err != nil {
		return SignedIn{}, err
	}
	// The account that sets the engine up is its first admin (HW-OPS-9).
	user.Admin = true
	now := a.Clock()
	err = a.Store.InTx(ctx, func(r port.Repos) error {
		n, err := r.CountUsers(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return domain.Conflict("setup is already done")
		}
		if err := r.CreateUser(ctx, user, hash); err != nil {
			return err
		}
		tenant := domain.Tenant{ID: NewID(), Name: tenantName, CreatedAt: now}
		if err := r.CreateTenant(ctx, tenant); err != nil {
			return err
		}
		if err := r.AddMember(ctx, tenant.ID, user.ID, domain.RoleOwner, now); err != nil {
			return err
		}
		return r.CreatePerson(ctx, tenant.ID, domain.Person{ID: NewID(), DisplayName: user.DisplayName, UserID: user.ID, CreatedAt: now})
	})
	if err != nil {
		return SignedIn{}, err
	}
	a.setupMu.Lock()
	a.setupCode = ""
	a.setupMu.Unlock()
	return a.startSession(ctx, user)
}

// Login checks the password and starts a session. After LoginMaxFailures
// failures in LoginLockWindow the email is locked for the window (HW-AUTH-5).
func (a *Accounts) Login(ctx context.Context, emailIn, password string) (SignedIn, error) {
	email, err := domain.Email(emailIn)
	if err != nil {
		return SignedIn{}, domain.ErrUnauthorized
	}
	// The attempt is recorded before the password is checked, so parallel
	// guesses all count; a successful login clears the record.
	now := a.Clock()
	if err := a.Store.RecordLoginFailure(ctx, email, now); err != nil {
		return SignedIn{}, err
	}
	attempts, err := a.Store.CountLoginFailures(ctx, email, now.Add(-domain.LoginLockWindow))
	if err != nil {
		return SignedIn{}, err
	}
	if attempts > domain.LoginMaxFailures {
		return SignedIn{}, fmt.Errorf("%w: try again in %d minutes", domain.ErrRateLimited, int(domain.LoginLockWindow.Minutes()))
	}
	user, hash, err := a.Store.UserByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		hash, user = a.dummyHash, domain.User{}
	} else if err != nil {
		return SignedIn{}, err
	}
	ok, err := a.Hasher.Verify(password, hash)
	if err != nil {
		return SignedIn{}, fmt.Errorf("check password: %w", err)
	}
	if !ok || user.ID == "" {
		return SignedIn{}, fmt.Errorf("%w: wrong email or password", domain.ErrUnauthorized)
	}
	if err := a.Store.ClearLoginFailures(ctx, email); err != nil {
		return SignedIn{}, err
	}
	return a.startSession(ctx, user)
}

func (a *Accounts) startSession(ctx context.Context, user domain.User) (SignedIn, error) {
	raw, err := newToken(prefixSession)
	if err != nil {
		return SignedIn{}, err
	}
	csrf, err := newToken("")
	if err != nil {
		return SignedIn{}, err
	}
	now := a.Clock()
	s := domain.Session{UserID: user.ID, CSRFToken: csrf, ExpiresAt: now.Add(domain.SessionLifetime)}
	if err := a.Store.CreateSession(ctx, HashToken(raw), s, now); err != nil {
		return SignedIn{}, err
	}
	me, err := a.me(ctx, user, csrf)
	if err != nil {
		return SignedIn{}, err
	}
	return SignedIn{Token: raw, Me: me}, nil
}

// Authenticate resolves a session cookie. A session within a day of being
// fresh is left alone; older ones are extended to a full lifetime (HW-AUTH-2).
func (a *Accounts) Authenticate(ctx context.Context, raw string) (domain.Session, error) {
	if raw == "" {
		return domain.Session{}, domain.ErrUnauthorized
	}
	hash := HashToken(raw)
	s, err := a.Store.SessionByHash(ctx, hash)
	if errors.Is(err, domain.ErrNotFound) {
		return s, domain.ErrUnauthorized
	}
	if err != nil {
		return s, err
	}
	now := a.Clock()
	if !now.Before(s.ExpiresAt) {
		if err := a.Store.DeleteSession(ctx, hash); err != nil {
			return s, err
		}
		return s, domain.ErrUnauthorized
	}
	if s.ExpiresAt.Sub(now) < domain.SessionLifetime-dayDuration {
		s.ExpiresAt = now.Add(domain.SessionLifetime)
		if err := a.Store.ExtendSession(ctx, hash, s.ExpiresAt); err != nil {
			return s, err
		}
	}
	return s, nil
}

// Logout ends the session.
func (a *Accounts) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return a.Store.DeleteSession(ctx, HashToken(raw))
}

// Me returns the signed-in user with their tenants.
func (a *Accounts) Me(ctx context.Context, s domain.Session) (domain.Me, error) {
	user, _, err := a.Store.UserByID(ctx, s.UserID)
	if err != nil {
		return domain.Me{}, err
	}
	return a.me(ctx, user, s.CSRFToken)
}

func (a *Accounts) me(ctx context.Context, user domain.User, csrf string) (domain.Me, error) {
	memberships, err := a.Store.MembershipsOf(ctx, user.ID)
	if err != nil {
		return domain.Me{}, err
	}
	return domain.Me{User: user, Memberships: memberships, CSRFToken: csrf}, nil
}

// ChangePassword checks the current password, sets the new one and ends
// every session of the user, then starts a new one for this browser.
func (a *Accounts) ChangePassword(ctx context.Context, userID, current, next string) (SignedIn, error) {
	user, hash, err := a.Store.UserByID(ctx, userID)
	if err != nil {
		return SignedIn{}, err
	}
	ok, err := a.Hasher.Verify(current, hash)
	if err != nil {
		return SignedIn{}, fmt.Errorf("check password: %w", err)
	}
	if !ok {
		return SignedIn{}, domain.Invalid("current password is wrong")
	}
	if err := a.setPassword(ctx, userID, next); err != nil {
		return SignedIn{}, err
	}
	return a.startSession(ctx, user)
}

// ResetPassword sets a new password by email and ends the user's sessions.
// Only the reset-password command calls it (HW-AUTH-1).
func (a *Accounts) ResetPassword(ctx context.Context, emailIn, next string) error {
	email, err := domain.Email(emailIn)
	if err != nil {
		return err
	}
	user, _, err := a.Store.UserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if err := a.setPassword(ctx, user.ID, next); err != nil {
		return err
	}
	return a.Store.ClearLoginFailures(ctx, email)
}

func (a *Accounts) setPassword(ctx context.Context, userID, next string) error {
	if err := domain.CheckPassword(next); err != nil {
		return err
	}
	hash, err := a.Hasher.Hash(next)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return a.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.SetPasswordHash(ctx, userID, hash); err != nil {
			return err
		}
		return r.DeleteUserSessions(ctx, userID)
	})
}

// ScopeFor checks that the user is a member of the tenant. A non-member
// gets domain.ErrNotFound, so tenant ids reveal nothing (HW-TEN-5).
func (a *Accounts) ScopeFor(ctx context.Context, userID, tenantID string) (domain.Scope, error) {
	role, err := a.Store.RoleOf(ctx, tenantID, userID)
	if err != nil {
		return domain.Scope{}, err
	}
	return domain.Scope{UserID: userID, TenantID: tenantID, Role: role, Credential: domain.CredentialSession}, nil
}

// UploadScope resolves a personal upload token (HW-AUTH-3). The token
// carries its owner's current role, so a demoted owner loses upload rights.
func (a *Accounts) UploadScope(ctx context.Context, raw string) (domain.Scope, error) {
	if raw == "" {
		return domain.Scope{}, domain.ErrUnauthorized
	}
	tok, err := a.Store.UploadTokenByHash(ctx, HashToken(raw))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Scope{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.Scope{}, err
	}
	if tok.Expired(a.Clock()) {
		return domain.Scope{}, domain.ErrUnauthorized
	}
	role, err := a.Store.RoleOf(ctx, tok.TenantID, tok.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Scope{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.Scope{}, err
	}
	if err := a.Store.TouchUploadToken(ctx, tok.ID, a.Clock()); err != nil {
		return domain.Scope{}, err
	}
	cred := domain.CredentialUploadToken
	if tok.ExpiresAt != nil {
		cred = domain.CredentialCaptureLink
	}
	return domain.Scope{UserID: tok.UserID, TenantID: tok.TenantID, Role: role, Credential: cred}, nil
}
