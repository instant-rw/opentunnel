package auth

import (
	"bytes"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "incorrect password") {
		t.Fatal("incorrect password verified")
	}
}

func TestPasswordHashesUseUniqueSalts(t *testing.T) {
	t.Parallel()
	first, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("password hashes unexpectedly match")
	}
}

func TestSecretsAreHashedAndRandom(t *testing.T) {
	t.Parallel()
	first, firstHash, err := NewSecret(32)
	if err != nil {
		t.Fatal(err)
	}
	second, secondHash, err := NewSecret(32)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || bytes.Equal(firstHash, secondHash) {
		t.Fatal("generated duplicate secrets")
	}
	if !bytes.Equal(firstHash, Digest(first)) {
		t.Fatal("secret digest mismatch")
	}
}

func TestParseBearer(t *testing.T) {
	t.Parallel()
	token, err := ParseBearer("Bearer secret")
	if err != nil {
		t.Fatal(err)
	}
	if token != "secret" {
		t.Fatalf("got token %q", token)
	}
	if _, err := ParseBearer("Basic secret"); err == nil {
		t.Fatal("accepted non-bearer authorization")
	}
}
