package service

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"strings"
	"testing"
)

var (
	oldToken = strings.Repeat("o", 32)
	newToken = strings.Repeat("n", 64)
)

func TestGivenAStoredTokenWhenStartingThenItGuardsWrites(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{domain.APITokenSettingKey: oldToken}}

	tokens := NewAPITokenService(store, store.values, "100.66.1.2")

	if tokens.Token() != oldToken {
		t.Fatal("the stored token must be in use")
	}
	if status := tokens.Status(); status != (domain.APITokenStatus{Set: true, Loopback: false}) {
		t.Fatalf("status = %+v", status)
	}
}

func TestGivenNoTokenOnLoopbackWhenAskingThenStatusSaysWritesAreOpenLocally(t *testing.T) {
	tokens := NewAPITokenService(&memorySettingsStore{values: map[string]string{}}, nil, "127.0.0.1")

	if tokens.Token() != "" || tokens.Status() != (domain.APITokenStatus{Set: false, Loopback: true}) {
		t.Fatalf("token %q status %+v", tokens.Token(), tokens.Status())
	}
}

func TestGivenANewTokenWhenSettingThenItIsSavedAndAppliesAtOnce(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{domain.APITokenSettingKey: oldToken, "AI_MODEL": "gpt"}}
	tokens := NewAPITokenService(store, store.values, "0.0.0.0")

	status, err := tokens.Set(context.Background(), domain.APITokenUpdate{Token: " " + newToken + " "})

	if err != nil || !status.Set {
		t.Fatalf("status %+v err %v", status, err)
	}
	if tokens.Token() != newToken || store.values[domain.APITokenSettingKey] != newToken || store.values["AI_MODEL"] != "gpt" {
		t.Fatalf("token %q stored %v", tokens.Token(), store.values)
	}
}

func TestGivenAnInvalidTokenWhenSettingThenNothingChanges(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{domain.APITokenSettingKey: oldToken}}
	tokens := NewAPITokenService(store, store.values, "0.0.0.0")

	for _, value := range []string{"", "short", strings.Repeat("x", 40) + " y"} {
		if _, err := tokens.Set(context.Background(), domain.APITokenUpdate{Token: value}); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%q: err = %v", value, err)
		}
	}
	if tokens.Token() != oldToken || store.saves != 0 {
		t.Fatalf("token %q saves %d", tokens.Token(), store.saves)
	}
}

func TestGivenTheStoreFailsWhenSettingThenTheOldTokenStays(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{domain.APITokenSettingKey: oldToken}, saveErr: errors.New("disk full")}
	tokens := NewAPITokenService(store, store.values, "0.0.0.0")

	if _, err := tokens.Set(context.Background(), domain.APITokenUpdate{Token: newToken}); err == nil {
		t.Fatal("a failed save must be reported")
	}
	if tokens.Token() != oldToken {
		t.Fatal("the running token must only change once it is saved")
	}
}
