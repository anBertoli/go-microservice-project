package validator

import (
	"fmt"
	"regexp"

	"github.com/anBertoli/snap-vault/pkg/store"
)

// RegExp to be matched against email strings, on order to verify their correctness.
var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// Validate the user name, email and password.
func ValidateUser(v Validator, user store.User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
	ValidateEmail(v, user.Email)
	ValidatePassword(v, user.Password)
}

// Validate permissions.
func ValidatePermissions(v Validator, permissions store.Permissions) {
	invalids := permissions.Invalids()
	v.Check(len(permissions) != 0, "permissions", "no permissions provided")
	v.Check(len(invalids) == 0, "permissions", fmt.Sprintf("invalid permissions: %v", invalids))
}

// Validate only the email.
func ValidateEmail(v Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(Matches(email, EmailRX), "email", "must be a valid email address")
}

// Validate only the password.
func ValidatePassword(v Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}
