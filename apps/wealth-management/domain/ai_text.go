package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// RenderPrompt substitutes {{name}} placeholders (WM-AI-1). Unknown
// placeholders are left as written so a sheet typo stays visible.
func RenderPrompt(template string, vars map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]
		if value, ok := vars[name]; ok {
			return value
		}
		return match
	})
}

var fencePattern = regexp.MustCompile("(?s)^```[A-Za-z]*\\s*\n(.*?)\n?```$")

// ParseAIJSON decodes a model reply that must be exactly one JSON value,
// optionally wrapped in a single code fence (WM-AI-3). Prose around the
// JSON is rejected rather than sliced out.
func ParseAIJSON(text string, target any) error {
	trimmed := strings.TrimSpace(text)
	if match := fencePattern.FindStringSubmatch(trimmed); match != nil {
		trimmed = strings.TrimSpace(match[1])
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrAIOutput, err)
	}
	if decoder.More() {
		return fmt.Errorf("%w: unexpected text after the JSON value", ErrAIOutput)
	}
	return nil
}

// PIIMasker hides identifiers in free text before it reaches a model
// (WM-AI-13): long digit runs (account, card and phone numbers) keep their
// last four digits, e-mail addresses are replaced, and configured names
// become "[name]". Numeric fields are not passed through it.
type PIIMasker struct {
	names *regexp.Regexp
}

var (
	// Nine or more digits, optionally grouped by spaces, dots or dashes.
	longNumberPattern = regexp.MustCompile(`\+?\d(?:[ .-]?\d){8,}`)
	emailPattern      = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	nonDigit          = regexp.MustCompile(`\D`)
)

func NewPIIMasker(names []string) PIIMasker {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			quoted = append(quoted, regexp.QuoteMeta(name))
		}
	}
	if len(quoted) == 0 {
		return PIIMasker{}
	}
	return PIIMasker{names: regexp.MustCompile(`(?i)(?:` + strings.Join(quoted, "|") + `)`)}
}

func (m PIIMasker) Mask(text string) string {
	if text == "" {
		return text
	}
	text = emailPattern.ReplaceAllString(text, "[email]")
	text = longNumberPattern.ReplaceAllStringFunc(text, func(match string) string {
		digits := nonDigit.ReplaceAllString(match, "")
		return "••••" + digits[len(digits)-4:]
	})
	if m.names != nil {
		text = m.maskNames(text)
	}
	return text
}

// maskNames replaces whole-word name matches. RE2's \b only knows ASCII
// word characters, so Vietnamese names such as "Đức" need a Unicode check.
func (m PIIMasker) maskNames(text string) string {
	var out strings.Builder
	last := 0
	for _, match := range m.names.FindAllStringIndex(text, -1) {
		before, _ := utf8.DecodeLastRuneInString(text[:match[0]])
		after, _ := utf8.DecodeRuneInString(text[match[1]:])
		if isWordRune(before) || isWordRune(after) {
			continue
		}
		out.WriteString(text[last:match[0]])
		out.WriteString("[name]")
		last = match[1]
	}
	out.WriteString(text[last:])
	return out.String()
}

func isWordRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_')
}

// ResolveCategory maps a model answer onto the existing category list
// (WM-AI-8): exact, then case-insensitive, else ok=false.
func ResolveCategory(answer string, categories []string) (string, bool) {
	answer = strings.Trim(strings.TrimSpace(answer), `"'`+"`.")
	for _, category := range categories {
		if category == answer {
			return category, true
		}
	}
	for _, category := range categories {
		if strings.EqualFold(strings.TrimSpace(category), answer) {
			return category, true
		}
	}
	return "", false
}
