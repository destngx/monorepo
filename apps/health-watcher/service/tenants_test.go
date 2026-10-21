package service

import (
	"errors"
	"testing"
	"time"

	"apps/health-watcher/domain"
)

func TestGivenAnInviteWhenAcceptedThenItAddsTheMemberOnce(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	inv, err := e.tenants.CreateInvite(e.ctx, owner, "editor")
	if err != nil {
		t.Fatal(err)
	}
	info, err := e.tenants.Invite(e.ctx, inv.Token)
	if err != nil || info.TenantName != "Test Family" || info.Role != domain.RoleEditor {
		t.Fatalf("Invite = %+v, %v", info, err)
	}
	in, err := e.tenants.AcceptInviteAsNewUser(e.ctx, inv.Token, SignUpInput{Email: "new@example.test", DisplayName: "New", Password: testPassword})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Me.Memberships) != 1 || in.Me.Memberships[0].Role != domain.RoleEditor {
		t.Fatalf("memberships = %+v", in.Me.Memberships)
	}
	_, err = e.tenants.AcceptInviteAsNewUser(e.ctx, inv.Token, SignUpInput{Email: "again@example.test", DisplayName: "A", Password: testPassword})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("reused invite = %v", err)
	}
	if _, _, err := e.store.UserByEmail(e.ctx, "again@example.test"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("failed accept left a user behind: %v", err)
	}
}

func TestGivenAnExistingUserWhenAcceptingAnInviteThenTheyJoinASecondTenant(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	other, err := e.tenants.Create(e.ctx, owner.UserID, "Parents")
	if err != nil {
		t.Fatal(err)
	}
	member := e.join(owner, "member@example.test", "viewer")
	inv, err := e.tenants.CreateInvite(e.ctx, e.scope(owner.UserID, other.ID), "editor")
	if err != nil {
		t.Fatal(err)
	}
	tenant, err := e.tenants.AcceptInvite(e.ctx, inv.Token, member.UserID)
	if err != nil || tenant.Name != "Parents" {
		t.Fatalf("AcceptInvite = %+v, %v", tenant, err)
	}
	if _, err := e.tenants.AcceptInvite(e.ctx, inv.Token, member.UserID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second accept = %v", err)
	}
}

func TestGivenAnExpiredInviteWhenAcceptingThenItIsNotFound(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	inv, _ := e.tenants.CreateInvite(e.ctx, owner, "viewer")
	e.now = e.now.Add(domain.InviteLifetime + time.Minute)
	if _, err := e.tenants.Invite(e.ctx, inv.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expired invite = %v", err)
	}
	pending, err := e.tenants.PendingInvites(e.ctx, owner)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending = %v, %v", pending, err)
	}
}

func TestGivenNonOwnersWhenManagingTheTenantThenTheyAreForbidden(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	editor := e.join(owner, "editor@example.test", "editor")
	if _, err := e.tenants.CreateInvite(e.ctx, editor, "viewer"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor invite = %v", err)
	}
	if err := e.tenants.SetRole(e.ctx, editor, owner.UserID, "viewer"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor set role = %v", err)
	}
	if err := e.tenants.RemoveMember(e.ctx, editor, owner.UserID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor remove = %v", err)
	}
	if err := e.tenants.Rename(e.ctx, editor, "Mine"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor rename = %v", err)
	}
}

func TestGivenTheLastOwnerWhenDemotedOrLeavingThenItIsRefused(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	member := e.join(owner, "member@example.test", "editor")
	if err := e.tenants.SetRole(e.ctx, owner, owner.UserID, "editor"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("demote last owner = %v", err)
	}
	if err := e.tenants.RemoveMember(e.ctx, owner, owner.UserID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("last owner leaves = %v", err)
	}
	if err := e.tenants.SetRole(e.ctx, owner, member.UserID, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := e.tenants.RemoveMember(e.ctx, owner, owner.UserID); err != nil {
		t.Fatalf("owner leaves with another owner = %v", err)
	}
	if _, err := e.accounts.ScopeFor(e.ctx, owner.UserID, owner.TenantID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("former member scope = %v", err)
	}
	people, _ := e.records.People(e.ctx, e.scope(member.UserID, owner.TenantID))
	if people[0].UserID != "" {
		t.Fatalf("former member still linked to %s", people[0].DisplayName)
	}
}

func TestGivenAViewerWhenLeavingThenTheyCanRemoveThemselves(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	viewer := e.join(owner, "viewer@example.test", "viewer")
	if err := e.tenants.RemoveMember(e.ctx, viewer, viewer.UserID); err != nil {
		t.Fatalf("leave = %v", err)
	}
}

func TestGivenAnInviteWhenItsCreatorIsDemotedThenItStopsWorking(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	second := e.join(owner, "second@example.test", "owner")
	inv, err := e.tenants.CreateInvite(e.ctx, second, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.tenants.SetRole(e.ctx, owner, second.UserID, "viewer"); err != nil {
		t.Fatal(err)
	}
	_, err = e.tenants.AcceptInviteAsNewUser(e.ctx, inv.Token, SignUpInput{Email: "alt@example.test", DisplayName: "Alt", Password: testPassword})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("invite of a demoted owner = %v", err)
	}
}

func TestGivenARemovedMemberWhenInvitedAgainThenTheirOldUploadTokensStayDead(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	editor := e.join(owner, "editor@example.test", "editor")
	tok, err := e.tenants.CreateUploadToken(e.ctx, editor, "Old phone")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.tenants.RemoveMember(e.ctx, owner, editor.UserID); err != nil {
		t.Fatal(err)
	}
	inv, _ := e.tenants.CreateInvite(e.ctx, owner, "editor")
	if _, err := e.tenants.AcceptInvite(e.ctx, inv.Token, editor.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.UploadScope(e.ctx, tok.Token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("old token after re-invite = %v", err)
	}
}
