package auth

import "golang.org/x/crypto/bcrypt"

// Password is a thin wrapper around bcrypt so callers don't import the crypto
// package directly. Cost is fixed at the bcrypt default (10) which balances
// security and latency; raising it is a config change here, not across the app.
type Password struct{}

func NewPassword() *Password { return &Password{} }

// Hash returns the bcrypt hash of the plaintext password.
func (Password) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare reports whether the plaintext matches the stored hash. It returns a
// boolean so callers can map failure to an Unauthorized error uniformly.
func (Password) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
