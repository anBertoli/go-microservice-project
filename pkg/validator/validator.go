package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// Define a new Validator type which contains a map of keys -> errors.
type Validator map[string]string

func New() Validator {
	return make(Validator)
}

// Implement the Error interface, so the Validator could be used as a regular error.
func (v Validator) Error() string {
	var messages []string
	for key, val := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", key, val))
	}
	return strings.Join(messages, ", ")
}

// Returns true if the errors map doesn't contain any entries.
func (v Validator) Ok() bool {
	return len(v) == 0
}

// Adds an error message to the map (so long as no entry already exists for
// the given key).
func (v Validator) AddError(key, message string) {
	if _, exists := v[key]; !exists {
		v[key] = message
	}
}

// Adds an error message to the map if a validation check is not 'ok'.
// E.g. v.Check(name != "", "name", "name must not be empty")
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// Returns true if a string value matches a specific regexp pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}
