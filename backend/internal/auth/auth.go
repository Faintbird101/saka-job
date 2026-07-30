// Package auth handles password hashing and session tokens.
//
// Kept separate from the HTTP and database layers so the security-critical
// parts — how a password is hashed, how a token is generated and compared —
// are in one small auditable place rather than spread through handlers.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// SessionTokenBytes is the entropy in a session token. 32 bytes is far beyond
// guessable and costs nothing.
const SessionTokenBytes = 32

// BcryptCost is deliberately above bcrypt's default of 10. This runs on a
// laptop for one user logging in occasionally, so ~100ms per hash is invisible
// to you and expensive for anyone brute-forcing a stolen database.
const BcryptCost = 12

// MinPasswordLength is a floor, not a policy. Length beats composition rules,
// which mostly produce "Password1!" and a sticky note.
const MinPasswordLength = 10

var (
	// ErrWeakPassword → 400.
	ErrWeakPassword = errors.New("password is too weak")
	// ErrBadCredentials is returned for BOTH an unknown email and a wrong
	// password, deliberately: distinguishing them tells an attacker which
	// addresses have accounts.
	ErrBadCredentials = errors.New("invalid email or password")
)

// HashPassword returns a bcrypt hash.
func HashPassword(password string) (string, error) {
	if err := CheckPasswordStrength(password); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// VerifyPassword compares a candidate against a stored hash.
//
// bcrypt's comparison is constant-time with respect to the hash, so this does
// not leak how much of a guess was right.
func VerifyPassword(hash, candidate string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(candidate)); err != nil {
		return ErrBadCredentials
	}
	return nil
}

// CheckPasswordStrength enforces a length floor and rejects the handful of
// passwords that are always the first guesses.
func CheckPasswordStrength(password string) error {
	if len([]rune(password)) < MinPasswordLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrWeakPassword, MinPasswordLength)
	}
	if isCommon(password) {
		return fmt.Errorf("%w: that is one of the most commonly used passwords", ErrWeakPassword)
	}
	// A single repeated character reaches the length floor without any entropy.
	if uniqueRunes(password) < 4 {
		return fmt.Errorf("%w: too few distinct characters", ErrWeakPassword)
	}
	return nil
}

var commonPasswords = map[string]bool{
	"password":    true,
	"password1":   true,
	"password123": true,
	"1234567890":  true,
	"qwertyuiop":  true,
	"letmein123":  true,
	"iloveyou123": true,
	"admin12345":  true,
	"welcome123":  true,
	"changeme123": true,
}

func isCommon(p string) bool {
	return commonPasswords[strings.ToLower(strings.TrimSpace(p))]
}

func uniqueRunes(s string) int {
	seen := map[rune]bool{}
	for _, r := range s {
		seen[unicode.ToLower(r)] = true
	}
	return len(seen)
}

// NewSessionToken mints a token and returns it alongside the hash to store.
//
// The plaintext is shown to the client exactly once and never persisted; the
// database keeps only the hash, so a leaked backup cannot be replayed as a
// login. Same reasoning as passwords.
func NewSessionToken() (token, hash string, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	// URL-safe and unpadded: it travels in an Authorization header and gets
	// pasted into places that mangle '+' and '='.
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

// HashToken is the one-way transform applied before storing or looking up a
// session token.
//
// SHA-256 rather than bcrypt here, and that is intentional: a session token is
// 256 bits of randomness, not a human-chosen password, so there is no
// dictionary to slow down — and this runs on every authenticated request,
// where bcrypt's deliberate slowness would be a self-inflicted DoS.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NormalizeEmail lowercases and trims, so a login is not defeated by an
// autocapitalising phone keyboard.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidEmail is a deliberately loose check. Anything stricter rejects valid
// addresses; the only real proof an address works is sending to it.
func ValidEmail(email string) bool {
	email = NormalizeEmail(email)
	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	if strings.Count(email, "@") != 1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".") &&
		!strings.HasPrefix(domain, ".") &&
		!strings.HasSuffix(domain, ".") &&
		!strings.ContainsAny(email, " \t\n")
}
