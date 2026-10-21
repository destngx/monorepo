package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestGivenEmailsWhenValidatingThenTheyAreLowerCasedOrRejected(t *testing.T) {
	if got, err := Email("  Someone@Example.COM "); err != nil || got != "someone@example.com" {
		t.Fatalf("Email = %q, %v", got, err)
	}
	for _, bad := range []string{"", "no-at-sign", "Name <a@b.c>", "a@"} {
		if _, err := Email(bad); !errors.Is(err, ErrValidation) {
			t.Errorf("Email(%q) = %v, want validation error", bad, err)
		}
	}
}

func TestGivenPasswordsWhenCheckingThenLengthRulesApply(t *testing.T) {
	if err := CheckPassword("short"); !errors.Is(err, ErrValidation) {
		t.Fatalf("short password accepted: %v", err)
	}
	if err := CheckPassword(strings.Repeat("x", PasswordMaxLength+1)); !errors.Is(err, ErrValidation) {
		t.Fatalf("long password accepted: %v", err)
	}
	if err := CheckPassword("mật khẩu đủ dài"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}

func TestGivenNamesWhenValidatingThenBlankAndLongAreRejected(t *testing.T) {
	if got, err := DisplayName("  Bé   An "); err != nil || got != "Bé An" {
		t.Fatalf("DisplayName = %q, %v", got, err)
	}
	if _, err := DisplayName("   "); !errors.Is(err, ErrValidation) {
		t.Fatalf("blank name accepted: %v", err)
	}
	if _, err := DisplayName(strings.Repeat("a", 81)); !errors.Is(err, ErrValidation) {
		t.Fatalf("long name accepted: %v", err)
	}
}
