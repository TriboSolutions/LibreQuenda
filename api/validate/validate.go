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

var dniPattern = regexp.MustCompile(`^[0-9XYZ][0-9]{7}[A-Z]$`)

const dniLetters = "TRWAGMYFPDXBNJZSQVHLCKE"

// DNI: a Spanish DNI or NIE (X/Y/Z for its first digit), upper case without
// spaces or dashes, whose control letter matches.
func DNI(dni string) bool {
	if !dniPattern.MatchString(dni) {
		return false
	}
	var number = 0
	for i, c := range dni[:8] {
		var digit = int(c - '0')
		if i == 0 && c >= 'X' {
			digit = int(c - 'X') // X=0, Y=1, Z=2
		}
		number = number*10 + digit
	}
	return dni[8] == dniLetters[number%23]
}

var postalCodePattern = regexp.MustCompile(`^(0[1-9]|[1-4][0-9]|5[0-2])[0-9]{3}$`)

// PostalCode: a Spanish one, five digits from a real province (01 to 52).
func PostalCode(code string) bool {
	return postalCodePattern.MatchString(code)
}

var phonePattern = regexp.MustCompile(`^\+?[0-9 ()-]+$`)

// Phone: a phone number as people write it ("+34 986 000 000", "(986) 00-00-00"):
// digits, spaces, brackets and dashes, an optional leading +, from 6 characters
// up to max (its column's size), so it always fits.
func Phone(phone string, max int) bool {
	return len(phone) >= 6 && len(phone) <= max && phonePattern.MatchString(phone) // ASCII only: bytes are characters
}
