package validate

import (
	"regexp"
)

var regx_char *regexp.Regexp

func init() {
	regx_char = regexp.MustCompile(`[a-zA-Z]+`)
}

func IsChar(s string) bool {
	return regx_char.MatchString(s)
}