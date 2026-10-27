// Package validate holds the input rules shared by every endpoint.
package validate

import (
	"net/mail"
	"regexp"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slug: what goes in an address such as /<council>/<facility>.
func Slug(slug string) bool {
	return len(slug) >= 2 && len(slug) <= 64 && slugPattern.MatchString(slug)
}

// Email: a bare address, no display name.
func Email(email string) bool {
	var address, err = mail.ParseAddress(email)
	return err == nil && address.Address == email && len(email) <= 255
}
