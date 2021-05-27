package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// Define a new Validator type which contains a map of validation errors.
type Validator map[string]string

// New is a helper which creates a new Validator instance with an empty errors map.
func New() Validator {
	return make(Validator)
}

// Ok returns true if the errors map doesn't contain any entries.
func (v Validator) Error() string {
	var messages []string
	for key, val := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", key, val))
	}
	return strings.Join(messages, ", ")
}

// Ok returns true if the errors map doesn't contain any entries.
func (v Validator) Ok() bool {
	return len(v) == 0
}

// AddError adds an error message to the map (so long as no entry already exists for
// the given key).
func (v Validator) AddError(key, message string) {
	if _, exists := v[key]; !exists {
		v[key] = message
	}
}

// Check adds an error message to the map only if a validation check is not 'ok'.
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// In returns true if a specific value is in a list of strings.
func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

// Matches returns true if a string value matches a specific regexp pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// Unique returns true if all string values in a slice are unique.
func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)
	for _, value := range values {
		uniqueValues[value] = true
	}
	return len(values) == len(uniqueValues)
}
