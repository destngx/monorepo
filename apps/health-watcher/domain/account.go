package domain

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// Account rules (HW-AUTH-1, HW-AUTH-2, HW-AUTH-5, HW-TEN-4).
const (
	PasswordMinLength = 10
	PasswordMaxLength = 256
	SessionLifetime   = 30 * 24 * time.Hour
	InviteLifetime    = 7 * 24 * time.Hour
	LoginMaxFailures  = 5
	LoginLockWindow   = 15 * time.Minute
)

// User is a login account. A user may be linked to a Person.
type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	// Admin may change the engine settings (HW-OPS-9).
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"createdAt"`
}

// Email validates an address and returns it in lower case.
func Email(s string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", Invalid("email address is not valid")
	}
	return email, nil
}

// DisplayName validates and cleans a person's or user's display name.
func DisplayName(s string) (string, error) {
	name := CleanText(s)
	if name == "" {
		return "", Invalid("name is required")
	}
	if utf8.RuneCountInString(name) > 80 {
		return "", Invalid("name is longer than 80 characters")
	}
	return name, nil
}

// CheckPassword enforces the length rules on a new password.
func CheckPassword(s string) error {
	n := utf8.RuneCountInString(s)
	if n < PasswordMinLength {
		return Invalid("password must be at least %d characters", PasswordMinLength)
	}
	if n > PasswordMaxLength {
		return Invalid("password must be at most %d characters", PasswordMaxLength)
	}
	return nil
}

// Session is a signed-in browser. The cookie holds the raw id; the store
// keeps only its hash.
type Session struct {
	UserID    string
	CSRFToken string
	ExpiresAt time.Time
}

// Invite is a one-time link that adds a member with a role (HW-TEN-4).
type Invite struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Role      Role      `json:"role"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// UploadToken is a named token that can only post to the inbox (HW-AUTH-3).
type UploadToken struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	TenantID   string     `json:"tenantId"`
	UserID     string     `json:"userId"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	// ExpiresAt is set on the short-lived tokens of a phone capture link;
	// a Shortcut token has none.
	ExpiresAt *time.Time `json:"expiresAt"`
}

// Expired reports whether the token is past its expiry at now.
func (t UploadToken) Expired(now time.Time) bool {
	return t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

// Me is the signed-in user with their tenants.
type Me struct {
	User        User         `json:"user"`
	Memberships []Membership `json:"memberships"`
	CSRFToken   string       `json:"csrfToken"`
}
