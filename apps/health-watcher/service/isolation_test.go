package service

import (
	"errors"
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// TestGivenTwoTenantsWhenOneReachesForTheOthersDataThenEverythingIsNotFound
// is the isolation suite (HW-TEN-6): a member of tenant B, using tenant B's
// scope, asks for each kind of tenant A resource by id.
func TestGivenTwoTenantsWhenOneReachesForTheOthersDataThenEverythingIsNotFound(t *testing.T) {
	e := newEnv(t)
	a := e.setup()
	// The owner of A invites someone, who then creates tenant B of their own.
	outsider := e.join(a, "outsider@example.test", "viewer")
	if err := e.tenants.RemoveMember(e.ctx, outsider, outsider.UserID); err != nil {
		t.Fatal(err)
	}
	tb, err := e.tenants.Create(e.ctx, outsider.UserID, "Tenant B")
	if err != nil {
		t.Fatal(err)
	}
	b := e.scope(outsider.UserID, tb.ID)

	person := e.person(a, "Private Person")
	items := e.upload(a, false, pdf("a1"), pdf("a2")).Items
	filed, err := e.inbox.File(e.ctx, a, items[0].ID, domain.FileInput{Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-01-01"}})
	if err != nil {
		t.Fatal(err)
	}
	fileID := filed.Document.Files[0].ID
	inv, err := e.tenants.CreateInvite(e.ctx, a, "viewer")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.accounts.ScopeFor(e.ctx, outsider.UserID, a.TenantID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("scope for non-member = %v", err)
	}
	checks := map[string]error{
		"person": func() error { _, err := e.records.Person(e.ctx, b, person.ID); return err }(),
		"update person": func() error {
			_, err := e.records.UpdatePerson(e.ctx, b, person.ID, domain.PersonInput{DisplayName: "X"})
			return err
		}(),
		"delete person": e.records.DeletePerson(e.ctx, b, person.ID),
		"visits":        func() error { _, err := e.records.Visits(e.ctx, b, person.ID, port.VisitFilter{}); return err }(),
		"visit":         func() error { _, err := e.records.Visit(e.ctx, b, filed.Visit.ID); return err }(),
		"update visit": func() error {
			_, err := e.records.UpdateVisit(e.ctx, b, filed.Visit.ID, domain.VisitInput{PersonID: person.ID, Date: "2025-01-01"})
			return err
		}(),
		"delete visit":    e.records.DeleteVisit(e.ctx, b, filed.Visit.ID),
		"delete document": e.records.DeleteDocument(e.ctx, b, filed.Document.ID),
		"file":            func() error { _, _, err := e.records.OpenFile(e.ctx, b, fileID); return err }(),
		"create visit for A's person": func() error {
			_, err := e.records.CreateVisit(e.ctx, b, domain.VisitInput{PersonID: person.ID, Date: "2025-01-01"})
			return err
		}(),
		"file A's item": func() error {
			_, err := e.inbox.File(e.ctx, b, items[1].ID, domain.FileInput{VisitID: filed.Visit.ID})
			return err
		}(),
		"discard A's item":  e.inbox.Discard(e.ctx, b, items[1].ID),
		"revoke A's invite": e.tenants.RevokeInvite(e.ctx, b, inv.Invite.ID),
		"set role in A":     e.tenants.SetRole(e.ctx, b, a.UserID, "viewer"),
		"remove A's owner":  e.tenants.RemoveMember(e.ctx, b, a.UserID),
	}
	for name, err := range checks {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: err = %v, want not found", name, err)
		}
	}

	people, _ := e.records.People(e.ctx, b)
	inbox, _ := e.inbox.Items(e.ctx, b)
	providers, _ := e.records.Providers(e.ctx, b)
	members, _ := e.tenants.Members(e.ctx, b)
	if len(people) != 0 || len(inbox) != 0 || len(providers) != 0 || len(members) != 1 {
		t.Fatalf("B sees people=%d inbox=%d providers=%d members=%d", len(people), len(inbox), len(providers), len(members))
	}
	// A's data is untouched by every attempt above.
	if v, err := e.records.Visit(e.ctx, a, filed.Visit.ID); err != nil || v.DocumentCount != 1 {
		t.Fatalf("A's visit = %+v, %v", v, err)
	}
}
