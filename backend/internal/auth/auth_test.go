package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	const pw = "correct-horse-battery"

	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if strings.Contains(hash, pw) {
		t.Fatal("the plaintext password appears in the hash")
	}
	if err := VerifyPassword(hash, pw); err != nil {
		t.Errorf("correct password rejected: %v", err)
	}
	if err := VerifyPassword(hash, pw+"x"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("wrong password accepted or wrong error: %v", err)
	}
}

// bcrypt salts every hash, so the same password must never produce the same
// stored value — otherwise identical passwords are visible in a stolen dump.
func TestHashesAreSalted(t *testing.T) {
	a, _ := HashPassword("correct-horse-battery")
	b, _ := HashPassword("correct-horse-battery")
	if a == b {
		t.Error("two hashes of the same password are identical — not salted")
	}
}

func TestPasswordStrength(t *testing.T) {
	weak := map[string]string{
		"too short":     "short1",
		"empty":         "",
		"common":        "password123",
		"common cased":  "Password123",
		"no variety":    "aaaaaaaaaaaa",
		"barely varied": "abababababab",
	}
	for name, pw := range weak {
		t.Run(name, func(t *testing.T) {
			if err := CheckPasswordStrength(pw); !errors.Is(err, ErrWeakPassword) {
				t.Errorf("%q accepted", pw)
			}
			if _, err := HashPassword(pw); err == nil {
				t.Errorf("HashPassword accepted the weak password %q", pw)
			}
		})
	}

	for _, pw := range []string{
		"correct-horse-battery",
		"a-perfectly-fine-one",
		"Tr0ub4dor&3xyz",
	} {
		if err := CheckPasswordStrength(pw); err != nil {
			t.Errorf("%q rejected: %v", pw, err)
		}
	}
}

func TestSessionTokensAreUniqueAndStoredHashed(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		token, hash, err := NewSessionToken()
		if err != nil {
			t.Fatalf("NewSessionToken: %v", err)
		}
		if seen[token] {
			t.Fatal("duplicate session token generated")
		}
		seen[token] = true

		if hash == token {
			t.Fatal("the stored hash equals the token — a leaked backup would be replayable")
		}
		if HashToken(token) != hash {
			t.Fatal("HashToken does not reproduce the stored hash; lookups would never match")
		}
		// URL-safe: it rides in an Authorization header.
		if strings.ContainsAny(token, "+/=") {
			t.Errorf("token is not URL-safe: %q", token)
		}
	}
}

func TestHashTokenIsDeterministic(t *testing.T) {
	if HashToken("abc") != HashToken("abc") {
		t.Error("HashToken is not deterministic — no session would ever validate")
	}
	if HashToken("abc") == HashToken("abd") {
		t.Error("HashToken collides on trivially different inputs")
	}
}

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"  Kinyua@Example.COM ": "kinyua@example.com",
		"a@b.co":                "a@b.co",
	} {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidEmail(t *testing.T) {
	valid := []string{"kinyuaviktar@gmail.com", "a@b.co", "first.last+tag@sub.example.org"}
	for _, e := range valid {
		if !ValidEmail(e) {
			t.Errorf("%q rejected but is valid", e)
		}
	}
	invalid := []string{"", "no-at-sign", "@example.com", "a@", "a@b", "a@@b.com", "a b@c.com", "a@.com", "a@b."}
	for _, e := range invalid {
		if ValidEmail(e) {
			t.Errorf("%q accepted but is invalid", e)
		}
	}
}

// The same error for an unknown address and a wrong password is what stops the
// login endpoint being used to enumerate which addresses have accounts.
func TestBadCredentialsIsIndistinguishable(t *testing.T) {
	hash, _ := HashPassword("correct-horse-battery")
	wrongPw := VerifyPassword(hash, "wrong-password-here")

	if !errors.Is(wrongPw, ErrBadCredentials) {
		t.Fatalf("wrong password gave %v", wrongPw)
	}
	// A caller with no user row returns the same sentinel; assert the message
	// carries nothing distinguishing.
	if strings.Contains(strings.ToLower(ErrBadCredentials.Error()), "user") ||
		strings.Contains(strings.ToLower(ErrBadCredentials.Error()), "not found") {
		t.Errorf("the error text hints at which half failed: %q", ErrBadCredentials)
	}
}
