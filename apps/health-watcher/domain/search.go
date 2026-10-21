package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// SearchIndexVersion names how the search index is built. Bump it when
// FoldSearchText or the indexed text changes; the engine then rebuilds the
// index at start.
const SearchIndexVersion = "2026-10-21.1"

// SearchKind is what a search hit points at (HW-FIND-2).
type SearchKind string

const (
	SearchPage    SearchKind = "page"
	SearchFinding SearchKind = "finding"
	SearchVisit   SearchKind = "visit"
)

// SnippetPart is a run of snippet text; Match marks a run that matched the query.
type SnippetPart struct {
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

// SearchHit is one result: the matching text and where it lives. A page of
// an unfiled inbox item has an ItemID and no visit.
type SearchHit struct {
	Kind          SearchKind    `json:"kind"`
	Snippet       []SnippetPart `json:"snippet"`
	PersonID      string        `json:"personId,omitempty"`
	PersonName    string        `json:"personName,omitempty"`
	Date          string        `json:"date,omitempty"`
	VisitID       string        `json:"visitId,omitempty"`
	VisitTitle    string        `json:"visitTitle,omitempty"`
	DocumentID    string        `json:"documentId,omitempty"`
	DocumentKind  DocumentKind  `json:"documentKind,omitempty"`
	DocumentTitle string        `json:"documentTitle,omitempty"`
	ItemID        string        `json:"itemId,omitempty"`
	PageID        string        `json:"pageId,omitempty"`
	PageNumber    int           `json:"pageNumber,omitempty"`
	ProviderName  string        `json:"providerName,omitempty"`
}

// MaxSearchTerms bounds the words of one query.
const MaxSearchTerms = 8

// foldMap folds text for matching: lower case, Vietnamese accents removed,
// đ to d, and every other character that is not a letter or digit to a
// space. origin[i] is the rune index in the NFC form of s that folded rune i
// came from, so a match in the folded text maps back to the original.
func foldMap(s string) (folded []rune, origin []int, original []rune) {
	original = []rune(norm.NFC.String(s))
	folded = make([]rune, 0, len(original))
	origin = make([]int, 0, len(original))
	var buf [utf8.UTFMax * 4]byte
	for i, r := range original {
		f := ' '
		switch {
		case r == 'đ' || r == 'Đ':
			f = 'd'
		case unicode.Is(unicode.Mn, r):
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			n := utf8.EncodeRune(buf[:], r)
			base, _ := utf8.DecodeRune(norm.NFD.Bytes(buf[:n]))
			f = unicode.ToLower(base)
		}
		folded = append(folded, f)
		origin = append(origin, i)
	}
	return folded, origin, original
}

// FoldSearchText is the indexed form of a text: FoldKey's rules, keeping
// one character per character so the index and the snippet agree.
func FoldSearchText(s string) string {
	folded, _, _ := foldMap(s)
	return string(folded)
}

// SearchQuery turns what the user typed into an FTS5 expression: every word
// must appear, as a word or the start of one, ignoring case and accents
// (`cat lop vi tinh` finds `Cắt lớp vi tính`). It also returns the folded
// words for the snippet.
func SearchQuery(q string) (string, []string, error) {
	terms := strings.Fields(FoldSearchText(q))
	if len(terms) == 0 {
		return "", nil, Invalid("type a word to search for")
	}
	if len(terms) > MaxSearchTerms {
		terms = terms[:MaxSearchTerms]
	}
	parts := make([]string, len(terms))
	for i, t := range terms {
		// Folded terms hold only letters and digits, so quoting is safe.
		parts[i] = `"` + t + `"*`
	}
	return strings.Join(parts, " AND "), terms, nil
}

func isWordRune(r rune) bool { return r != ' ' }

// Snippet returns the part of text around the first match of any term,
// about radius characters on each side, with every match in it marked.
// Text without a match gives its start.
func Snippet(text string, terms []string, radius int) []SnippetPart {
	folded, origin, original := foldMap(text)
	type span struct{ start, end int } // in original runes
	var spans []span
	for i := 0; i < len(folded); i++ {
		if !isWordRune(folded[i]) || (i > 0 && isWordRune(folded[i-1])) {
			continue
		}
		best := 0
		for _, t := range terms {
			tr := []rune(t)
			if len(tr) > best && i+len(tr) <= len(folded) && string(folded[i:i+len(tr)]) == t {
				best = len(tr)
			}
		}
		if best > 0 {
			spans = append(spans, span{origin[i], origin[i+best-1] + 1})
			i += best - 1
		}
	}
	from, to := 0, min(len(original), 2*radius)
	if len(spans) > 0 {
		from = max(0, spans[0].start-radius)
		to = min(len(original), spans[0].end+radius)
		// Shrink the window to whole words.
		for from > 0 && from < spans[0].start && !unicode.IsSpace(original[from-1]) {
			from++
		}
		for to < len(original) && to > spans[0].end && !unicode.IsSpace(original[to]) {
			to--
		}
	}
	var parts []SnippetPart
	add := func(s string, match bool) {
		if match {
			s = strings.Join(strings.Fields(s), " ")
		} else {
			s = keepEdgeSpaces(s)
		}
		if s != "" {
			parts = append(parts, SnippetPart{Text: s, Match: match})
		}
	}
	if from > 0 {
		parts = append(parts, SnippetPart{Text: "… "})
	}
	cursor := from
	for _, sp := range spans {
		if sp.start < from || sp.end > to {
			continue
		}
		add(string(original[cursor:sp.start]), false)
		add(string(original[sp.start:sp.end]), true)
		cursor = sp.end
	}
	add(string(original[cursor:to]), false)
	if to < len(original) {
		parts = append(parts, SnippetPart{Text: " …"})
	}
	if parts == nil {
		parts = []SnippetPart{}
	}
	return parts
}

// keepEdgeSpaces collapses inner whitespace but keeps one leading and one
// trailing space, so runs join with the spacing of the original.
func keepEdgeSpaces(s string) string {
	if strings.TrimSpace(s) == "" {
		if s == "" {
			return ""
		}
		return " "
	}
	lead, trail := "", ""
	if r, _ := utf8.DecodeRuneInString(s); unicode.IsSpace(r) {
		lead = " "
	}
	if r, _ := utf8.DecodeLastRuneInString(s); unicode.IsSpace(r) {
		trail = " "
	}
	return lead + strings.Join(strings.Fields(s), " ") + trail
}
