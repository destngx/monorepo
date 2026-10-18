package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestGivenATokenWhenNormalizingThenOnlyLongPrintableTokensPass(t *testing.T) {
	long := strings.Repeat("a", minAPITokenLength)
	cases := []struct {
		name, value, want string
		wantErr           bool
	}{
		{name: "hex token", value: long, want: long},
		{name: "surrounding spaces are trimmed", value: "  " + long + "\t", want: long},
		{name: "empty", value: "   ", wantErr: true},
		{name: "too short", value: long[1:], wantErr: true},
		{name: "too long", value: strings.Repeat("a", maxAPITokenLength+1), wantErr: true},
		{name: "inner space", value: long + " b", wantErr: true},
		{name: "newline", value: long + "\nb", wantErr: true},
		{name: "non ascii", value: long + "é", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeAPIToken(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("err = %v, want a validation error", err)
				}
				if strings.Contains(err.Error(), tc.value) && tc.value != "" {
					t.Fatalf("the error must not echo the token: %v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestGivenAHostWhenCheckingLoopbackThenOnlyLocalAddressesAre(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1":     true,
		"localhost":     true,
		"::1":           true,
		"[::1]":         true,
		" 127.0.0.2 ":   true,
		"0.0.0.0":       false,
		"100.66.1.2":    false,
		"mac.ts.net":    false,
		"":              false,
		"192.168.1.10":  false,
		"not a host at": false,
	} {
		if got := IsLoopbackHost(host); got != want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}
