package password

import (
	"strings"
	"testing"
)

// Cheap params keep tests fast; the format and checks are the same.
var testParams = Params{MemoryKiB: 64, Passes: 1, Lanes: 1}

func TestGivenAPasswordWhenHashingThenOnlyTheSamePasswordVerifies(t *testing.T) {
	h := New(testParams)
	encoded, err := h.Hash("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("encoded = %s", encoded)
	}
	if ok, err := h.Verify("correct horse battery", encoded); err != nil || !ok {
		t.Fatalf("Verify(right) = %v, %v", ok, err)
	}
	if ok, err := h.Verify("wrong horse battery", encoded); err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v", ok, err)
	}
}

func TestGivenAHashMadeWithOtherParamsWhenVerifyingThenItStillWorks(t *testing.T) {
	old, _ := New(testParams).Hash("mật khẩu cũ dài")
	if ok, err := New(Params{MemoryKiB: 128, Passes: 2, Lanes: 2}).Verify("mật khẩu cũ dài", old); err != nil || !ok {
		t.Fatalf("Verify = %v, %v", ok, err)
	}
}

func TestGivenAMalformedHashWhenVerifyingThenItIsAnError(t *testing.T) {
	for _, bad := range []string{"", "plain", "$bcrypt$x$y$z$w", "$argon2id$v=19$m=x$a$b"} {
		if _, err := New(testParams).Verify("pw", bad); err == nil {
			t.Errorf("Verify(%q) accepted", bad)
		}
	}
}
