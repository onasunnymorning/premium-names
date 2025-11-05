package normalize

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
)

var (
	// ErrInvalidLabel indicates the provided string is not a valid DNS label.
	ErrInvalidLabel = errors.New("invalid DNS label")
)

var (
	ldhRe = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)
)

// ExtractFirstLabel returns the first label of a domain-like string.
// It trims whitespace, cuts off any path/query (# or /) if present,
// removes any trailing dot, and returns the substring before the first dot.
// Examples:
//
//	"Example.com" -> "Example"
//	" café.example" -> " café" (normalization happens later)
func ExtractFirstLabel(s string) string {
	s = strings.TrimSpace(s)
	// remove any path/fragment/query portions
	if i := strings.IndexAny(s, "/#?"); i >= 0 {
		s = s[:i]
	}
	// remove a possible trailing dot (FQDN)
	s = strings.TrimSuffix(s, ".")
	// take only the first label before dot
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	return s
}

// ToASCII converts a single label to its ASCII (Punycode) form using IDNA Lookup profile.
func ToASCII(label string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", ErrInvalidLabel
	}
	ascii, err := idna.Lookup.ToASCII(label)
	if err != nil {
		return "", ErrInvalidLabel
	}
	// DNS is case-insensitive; store lowercase canonical form
	ascii = strings.ToLower(ascii)
	return ascii, nil
}

// ToUnicode converts an ASCII label (possibly punycoded) to Unicode.
func ToUnicode(ascii string) (string, error) {
	unicode, err := idna.Lookup.ToUnicode(ascii)
	if err != nil {
		return "", ErrInvalidLabel
	}
	return unicode, nil
}

// ValidateLDH asserts the ASCII label is LDH and within length constraints with no leading/trailing hyphen.
func ValidateLDH(ascii string) error {
	if !ldhRe.MatchString(ascii) {
		return ErrInvalidLabel
	}
	if strings.HasPrefix(ascii, "-") || strings.HasSuffix(ascii, "-") {
		return ErrInvalidLabel
	}
	return nil
}

// NormalizeInput accepts a raw input that may be a label or a full domain.
// It extracts the first label, converts to ASCII and Unicode, validates LDH, and returns both forms.
func NormalizeInput(input string) (ascii string, unicode string, err error) {
	s := strings.TrimSpace(input)
	// If the input looks like a full domain (has a dot), extract first label.
	// If it contains a slash but no dot, treat as invalid (labels can't contain '/').
	if strings.Contains(s, "/") && !strings.Contains(s, ".") {
		return "", "", ErrInvalidLabel
	}
	label := ExtractFirstLabel(s)
	if strings.ContainsAny(label, "/") {
		return "", "", ErrInvalidLabel
	}
	ascii, err = ToASCII(label)
	if err != nil {
		return "", "", err
	}
	if err = ValidateLDH(ascii); err != nil {
		return "", "", err
	}
	unicode, err = ToUnicode(ascii)
	if err != nil {
		return "", "", err
	}
	return ascii, unicode, nil
}
