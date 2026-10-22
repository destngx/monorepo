package domain

import (
	"errors"
	"testing"
)

func TestGivenEachRoleWhenCheckingPermissionsThenOnlyAllowedActionsPass(t *testing.T) {
	cases := []struct {
		name                       string
		scope                      Scope
		read, edit, manage, upload bool
	}{
		{"owner", Scope{UserID: "u", TenantID: "t", Role: RoleOwner, Credential: CredentialSession}, true, true, true, true},
		{"editor", Scope{UserID: "u", TenantID: "t", Role: RoleEditor, Credential: CredentialSession}, true, true, false, true},
		{"viewer", Scope{UserID: "u", TenantID: "t", Role: RoleViewer, Credential: CredentialSession}, true, false, false, false},
		{"upload token of an owner", Scope{UserID: "u", TenantID: "t", Role: RoleOwner, Credential: CredentialUploadToken}, false, false, false, true},
		{"upload token of a viewer", Scope{UserID: "u", TenantID: "t", Role: RoleViewer, Credential: CredentialUploadToken}, false, false, false, false},
		{"no tenant", Scope{UserID: "u", TenantID: "", Role: RoleOwner, Credential: CredentialSession}, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := [4]bool{tc.scope.CanRead(), tc.scope.CanEdit(), tc.scope.CanManage(), tc.scope.CanUpload()}
			want := [4]bool{tc.read, tc.edit, tc.manage, tc.upload}
			if got != want {
				t.Fatalf("read/edit/manage/upload = %v, want %v", got, want)
			}
			if err := tc.scope.RequireEdit(); (err == nil) != tc.edit || (err != nil && !errors.Is(err, ErrForbidden)) {
				t.Fatalf("RequireEdit = %v", err)
			}
		})
	}
}

func TestGivenAnUnknownRoleWhenParsingThenItIsAValidationError(t *testing.T) {
	if _, err := ParseRole("admin"); !errors.Is(err, ErrValidation) {
		t.Fatalf("ParseRole(admin) = %v", err)
	}
	if r, err := ParseRole("editor"); err != nil || r != RoleEditor {
		t.Fatalf("ParseRole(editor) = %v, %v", r, err)
	}
}

func TestGivenACaptureLinkMadeOnAnItemWhenAddingFilesThenOnlyThatItemAccepts(t *testing.T) {
	link := Scope{UserID: "u", TenantID: "t", Role: RoleEditor, Credential: CredentialCaptureLink, ItemID: "i1"}
	if !link.CanAddTo("i1") || link.CanAddTo("i2") || link.CanAddTo("") {
		t.Fatal("an item link adds to its own item only")
	}
	inboxLink := Scope{UserID: "u", TenantID: "t", Role: RoleEditor, Credential: CredentialCaptureLink}
	if inboxLink.CanAddTo("i1") {
		t.Fatal("an inbox link added to an item")
	}
	viewerLink := Scope{UserID: "u", TenantID: "t", Role: RoleViewer, Credential: CredentialCaptureLink, ItemID: "i1"}
	if err := viewerLink.RequireAddTo("i1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("a viewer's link = %v", err)
	}
	editor := Scope{UserID: "u", TenantID: "t", Role: RoleEditor, Credential: CredentialSession}
	if !editor.CanAddTo("i2") {
		t.Fatal("an editor cannot add to an item")
	}
}
