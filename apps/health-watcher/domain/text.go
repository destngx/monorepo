package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// FoldKey returns the matching key of a name: lower case, Vietnamese accents
// removed (đ becomes d), and runs of spaces and punctuation collapsed to one
// space. `BV Chợ Rẫy` and `bv  CHO-RAY` have the same key (HW-REC-5).
func FoldKey(s string) string {
	var b strings.Builder
	space := false
	for _, r := range norm.NFD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r == 'đ' || r == 'Đ':
			r = 'd'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		space = true
	}
	return b.String()
}

// CleanText trims s and collapses inner whitespace to single spaces.
func CleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// CleanLines cleans each line of s like CleanText and drops blank lines,
// so a report's description keeps one line per printed line.
func CleanLines(s string) string {
	var out []string
	for l := range strings.Lines(s) {
		if l = CleanText(l); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
