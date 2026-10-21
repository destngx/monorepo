package domain

import "time"

// Role is a member's role in a tenant (HW-TEN-2).
type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// ParseRole validates a role name.
func ParseRole(s string) (Role, error) {
	switch r := Role(s); r {
	case RoleOwner, RoleEditor, RoleViewer:
		return r, nil
	}
	return "", Invalid("role must be owner, editor or viewer")
}

// Tenant is a family or group; everything stored belongs to one (HW-TEN-1).
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// TenantName validates and cleans a tenant name.
func TenantName(s string) (string, error) {
	name := CleanText(s)
	if name == "" {
		return "", Invalid("tenant name is required")
	}
	if len([]rune(name)) > 80 {
		return "", Invalid("tenant name is longer than 80 characters")
	}
	return name, nil
}

// Membership is a user's role in one tenant.
type Membership struct {
	Tenant Tenant `json:"tenant"`
	Role   Role   `json:"role"`
}

// Member is a user as seen from a tenant's member list.
type Member struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        Role      `json:"role"`
	JoinedAt    time.Time `json:"joinedAt"`
}

// Credential says how a request proved who it is.
type Credential string

const (
	CredentialSession     Credential = "session"
	CredentialUploadToken Credential = "upload_token"
)

// Scope is who acts, in which tenant, with which role. Services take it
// from the request and pass TenantID to repositories (HW-TEN-6).
type Scope struct {
	UserID     string
	TenantID   string
	Role       Role
	Credential Credential
}

// CanRead reports whether the scope may read tenant data. An upload token
// can only send files to the inbox (HW-AUTH-3).
func (s Scope) CanRead() bool {
	return s.Credential == CredentialSession && s.TenantID != ""
}

// CanEdit reports whether the scope may add, change and approve records.
func (s Scope) CanEdit() bool {
	return s.CanRead() && (s.Role == RoleOwner || s.Role == RoleEditor)
}

// CanManage reports whether the scope may manage members and settings.
func (s Scope) CanManage() bool {
	return s.CanRead() && s.Role == RoleOwner
}

// CanUpload reports whether the scope may send files to the inbox.
func (s Scope) CanUpload() bool {
	return s.TenantID != "" && (s.Role == RoleOwner || s.Role == RoleEditor)
}

// RequireRead returns ErrForbidden unless the scope can read.
func (s Scope) RequireRead() error {
	if !s.CanRead() {
		return Forbidden("read records")
	}
	return nil
}

// RequireEdit returns ErrForbidden unless the scope can edit.
func (s Scope) RequireEdit() error {
	if !s.CanEdit() {
		return Forbidden("change records")
	}
	return nil
}

// RequireManage returns ErrForbidden unless the scope can manage.
func (s Scope) RequireManage() error {
	if !s.CanManage() {
		return Forbidden("manage the tenant")
	}
	return nil
}

// RequireUpload returns ErrForbidden unless the scope can upload.
func (s Scope) RequireUpload() error {
	if !s.CanUpload() {
		return Forbidden("upload files")
	}
	return nil
}
