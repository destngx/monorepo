package service

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Tenants manages tenants, members, invites and upload tokens
// (HW-TEN-1..5, HW-AUTH-3).
type Tenants struct {
	Deps
	Accounts *Accounts
}

// Create makes a new tenant with the user as its owner.
func (t *Tenants) Create(ctx context.Context, userID, nameIn string) (domain.Tenant, error) {
	name, err := domain.TenantName(nameIn)
	if err != nil {
		return domain.Tenant{}, err
	}
	now := t.Clock()
	tenant := domain.Tenant{ID: NewID(), Name: name, CreatedAt: now}
	err = t.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.CreateTenant(ctx, tenant); err != nil {
			return err
		}
		return r.AddMember(ctx, tenant.ID, userID, domain.RoleOwner, now)
	})
	return tenant, err
}

// Rename changes the tenant's name.
func (t *Tenants) Rename(ctx context.Context, s domain.Scope, nameIn string) error {
	if err := s.RequireManage(); err != nil {
		return err
	}
	name, err := domain.TenantName(nameIn)
	if err != nil {
		return err
	}
	return t.Store.RenameTenant(ctx, s.TenantID, name)
}

// Members lists the tenant's members.
func (t *Tenants) Members(ctx context.Context, s domain.Scope) ([]domain.Member, error) {
	if err := s.RequireRead(); err != nil {
		return nil, err
	}
	return t.Store.Members(ctx, s.TenantID)
}

// SetRole changes a member's role. The tenant keeps at least one owner.
func (t *Tenants) SetRole(ctx context.Context, s domain.Scope, userID, roleIn string) error {
	if err := s.RequireManage(); err != nil {
		return err
	}
	role, err := domain.ParseRole(roleIn)
	if err != nil {
		return err
	}
	return t.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.SetRole(ctx, s.TenantID, userID, role); err != nil {
			return err
		}
		return keepAnOwner(ctx, r, s.TenantID)
	})
}

// RemoveMember removes a member. Owners remove anyone; any member may
// remove themselves. The tenant keeps at least one owner, and the member's
// link to a person is cleared.
func (t *Tenants) RemoveMember(ctx context.Context, s domain.Scope, userID string) error {
	if userID != s.UserID {
		if err := s.RequireManage(); err != nil {
			return err
		}
	} else if err := s.RequireRead(); err != nil {
		return err
	}
	return t.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.RemoveMember(ctx, s.TenantID, userID); err != nil {
			return err
		}
		if err := keepAnOwner(ctx, r, s.TenantID); err != nil {
			return err
		}
		// Old tokens must not come back to life if the user is invited again.
		if err := r.DeleteUserUploadTokens(ctx, s.TenantID, userID); err != nil {
			return err
		}
		return r.UnlinkUser(ctx, s.TenantID, userID)
	})
}

func keepAnOwner(ctx context.Context, r port.Repos, tenantID string) error {
	n, err := r.CountOwners(ctx, tenantID)
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.Conflict("a tenant needs at least one owner; make someone else owner first")
	}
	return nil
}

// CreatedInvite is a new invite and its one-time token.
type CreatedInvite struct {
	Invite domain.Invite `json:"invite"`
	Token  string        `json:"token"`
}

// CreateInvite makes a one-time invite with a role, valid InviteLifetime.
func (t *Tenants) CreateInvite(ctx context.Context, s domain.Scope, roleIn string) (CreatedInvite, error) {
	if err := s.RequireManage(); err != nil {
		return CreatedInvite{}, err
	}
	role, err := domain.ParseRole(roleIn)
	if err != nil {
		return CreatedInvite{}, err
	}
	raw, err := newToken(prefixInvite)
	if err != nil {
		return CreatedInvite{}, err
	}
	now := t.Clock()
	inv := domain.Invite{ID: NewID(), TenantID: s.TenantID, Role: role, CreatedBy: s.UserID, CreatedAt: now,
		ExpiresAt: now.Add(domain.InviteLifetime)}
	if err := t.Store.CreateInvite(ctx, inv, HashToken(raw)); err != nil {
		return CreatedInvite{}, err
	}
	return CreatedInvite{Invite: inv, Token: raw}, nil
}

// PendingInvites lists unused, unexpired invites.
func (t *Tenants) PendingInvites(ctx context.Context, s domain.Scope) ([]domain.Invite, error) {
	if err := s.RequireManage(); err != nil {
		return nil, err
	}
	return t.Store.PendingInvites(ctx, s.TenantID, t.Clock())
}

// RevokeInvite deletes a pending invite.
func (t *Tenants) RevokeInvite(ctx context.Context, s domain.Scope, id string) error {
	if err := s.RequireManage(); err != nil {
		return err
	}
	return t.Store.DeleteInvite(ctx, s.TenantID, id)
}

// InviteInfo is what an invite link shows before it is accepted.
type InviteInfo struct {
	TenantName string      `json:"tenantName"`
	Role       domain.Role `json:"role"`
}

func (t *Tenants) liveInvite(ctx context.Context, r port.Repos, raw string) (domain.Invite, error) {
	inv, err := r.InviteByHash(ctx, HashToken(raw))
	if errors.Is(err, domain.ErrNotFound) {
		return inv, domain.NotFound("invite (it may be used, revoked or mistyped)")
	}
	if err != nil {
		return inv, err
	}
	if !t.Clock().Before(inv.ExpiresAt) {
		return inv, domain.NotFound("invite (it has expired)")
	}
	// An invite carries its creator's authority: it dies when they stop
	// being an owner, so a demoted owner can't rejoin through it.
	role, err := r.RoleOf(ctx, inv.TenantID, inv.CreatedBy)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return inv, err
	}
	if role != domain.RoleOwner {
		return inv, domain.NotFound("invite (whoever created it is no longer an owner)")
	}
	return inv, nil
}

// Invite describes a pending invite by its token.
func (t *Tenants) Invite(ctx context.Context, raw string) (InviteInfo, error) {
	inv, err := t.liveInvite(ctx, t.Store, raw)
	if err != nil {
		return InviteInfo{}, err
	}
	tenant, err := t.Store.Tenant(ctx, inv.TenantID)
	if err != nil {
		return InviteInfo{}, err
	}
	return InviteInfo{TenantName: tenant.Name, Role: inv.Role}, nil
}

func (t *Tenants) accept(ctx context.Context, r port.Repos, raw, userID string) (domain.Invite, error) {
	inv, err := t.liveInvite(ctx, r, raw)
	if err != nil {
		return inv, err
	}
	now := t.Clock()
	if err := r.UseInvite(ctx, inv.ID, userID, now); err != nil {
		return inv, err
	}
	return inv, r.AddMember(ctx, inv.TenantID, userID, inv.Role, now)
}

// AcceptInvite adds the signed-in user to the invite's tenant.
func (t *Tenants) AcceptInvite(ctx context.Context, raw, userID string) (domain.Tenant, error) {
	var tenantID string
	err := t.Store.InTx(ctx, func(r port.Repos) error {
		inv, err := t.accept(ctx, r, raw, userID)
		tenantID = inv.TenantID
		return err
	})
	if err != nil {
		return domain.Tenant{}, err
	}
	return t.Store.Tenant(ctx, tenantID)
}

// AcceptInviteAsNewUser creates an account, adds it to the invite's tenant
// and signs it in.
func (t *Tenants) AcceptInviteAsNewUser(ctx context.Context, raw string, in SignUpInput) (SignedIn, error) {
	// A bad token fails before the costly password hash.
	if _, err := t.liveInvite(ctx, t.Store, raw); err != nil {
		return SignedIn{}, err
	}
	user, hash, err := t.Accounts.newUser(in)
	if err != nil {
		return SignedIn{}, err
	}
	err = t.Store.InTx(ctx, func(r port.Repos) error {
		if _, err := t.liveInvite(ctx, r, raw); err != nil {
			return err
		}
		if err := r.CreateUser(ctx, user, hash); err != nil {
			return err
		}
		_, err := t.accept(ctx, r, raw, user.ID)
		return err
	})
	if err != nil {
		return SignedIn{}, err
	}
	return t.Accounts.startSession(ctx, user)
}

// CreatedUploadToken is a new upload token and its raw value, shown once.
type CreatedUploadToken struct {
	UploadToken domain.UploadToken `json:"uploadToken"`
	Token       string             `json:"token"`
}

// CreateUploadToken makes a named token for the user in this tenant.
func (t *Tenants) CreateUploadToken(ctx context.Context, s domain.Scope, nameIn string) (CreatedUploadToken, error) {
	if err := s.RequireRead(); err != nil {
		return CreatedUploadToken{}, err
	}
	if err := s.RequireUpload(); err != nil {
		return CreatedUploadToken{}, err
	}
	name := domain.CleanText(nameIn)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		return CreatedUploadToken{}, domain.Invalid("token name must be 1 to 60 characters")
	}
	raw, err := newToken(prefixUpload)
	if err != nil {
		return CreatedUploadToken{}, err
	}
	tok := domain.UploadToken{ID: NewID(), Name: name, TenantID: s.TenantID, UserID: s.UserID, CreatedAt: t.Clock()}
	if err := t.Store.CreateUploadToken(ctx, tok, HashToken(raw)); err != nil {
		return CreatedUploadToken{}, fmt.Errorf("create upload token: %w", err)
	}
	return CreatedUploadToken{UploadToken: tok, Token: raw}, nil
}

// UploadTokens lists the user's tokens in this tenant.
func (t *Tenants) UploadTokens(ctx context.Context, s domain.Scope) ([]domain.UploadToken, error) {
	if err := s.RequireRead(); err != nil {
		return nil, err
	}
	return t.Store.UploadTokens(ctx, s.TenantID, s.UserID)
}

// RevokeUploadToken deletes one of the user's tokens.
func (t *Tenants) RevokeUploadToken(ctx context.Context, s domain.Scope, id string) error {
	if err := s.RequireRead(); err != nil {
		return err
	}
	return t.Store.DeleteUploadToken(ctx, s.TenantID, s.UserID, id)
}
