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
		{"owner", Scope{"u", "t", RoleOwner, CredentialSession}, true, true, true, true},
		{"editor", Scope{"u", "t", RoleEditor, CredentialSession}, true, true, false, true},
		{"viewer", Scope{"u", "t", RoleViewer, CredentialSession}, true, false, false, false},
		{"upload token of an owner", Scope{"u", "t", RoleOwner, CredentialUploadToken}, false, false, false, true},
		{"upload token of a viewer", Scope{"u", "t", RoleViewer, CredentialUploadToken}, false, false, false, false},
		{"no tenant", Scope{"u", "", RoleOwner, CredentialSession}, false, false, false, false},
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
