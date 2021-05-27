package validator

import (
	"fmt"
	"regexp"

	"github.com/anBertoli/snap-vault/pkg/store"
)

var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

func ValidateUser(v Validator, user store.User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
	v.Check(user.Email != "", "email", "must be provided")
	v.Check(Matches(user.Email, EmailRX), "email", "must be a valid email address")
	v.Check(user.Password != "", "password", "must be provided")
	v.Check(len(user.Password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(user.Password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidatePermissions(v Validator, permissions store.Permissions) {
	invalids := permissions.Invalids()
	v.Check(len(permissions) != 0, "permissions", "no permissions provided")
	v.Check(len(invalids) == 0, "permissions", fmt.Sprintf("invalid permissions: %v", invalids))
}

func ValidateEmail(v Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(Matches(email, EmailRX), "email", "must be a valid email address")
}

func ValidatePassword(v Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}
