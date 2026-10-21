package domain

import (
	"errors"
	"strings"
	"testing"
)

func joinSnippet(parts []SnippetPart) (text string, matches []string) {
	var b strings.Builder
	for _, p := range parts {
		if p.Match {
			matches = append(matches, p.Text)
			b.WriteString("[" + p.Text + "]")
			continue
		}
		b.WriteString(p.Text)
	}
	return b.String(), matches
}

func TestGivenAnUnaccentedQueryWhenBuildingTheExpressionThenEveryWordIsAFoldedPrefix(t *testing.T) {
	expr, terms, err := SearchQuery("  Cắt LỚP vi-tính  ")
	if err != nil {
		t.Fatal(err)
	}
	if expr != `"cat"* AND "lop"* AND "vi"* AND "tinh"*` {
		t.Fatalf("expr = %s", expr)
	}
	if strings.Join(terms, ",") != "cat,lop,vi,tinh" {
		t.Fatalf("terms = %v", terms)
	}
}

func TestGivenAnEmptyOrPunctuationQueryWhenBuildingThenItIsRejected(t *testing.T) {
	for _, q := range []string{"", "   ", `"*()-`} {
		if _, _, err := SearchQuery(q); !errors.Is(err, ErrValidation) {
			t.Errorf("SearchQuery(%q) err = %v", q, err)
		}
	}
}

func TestGivenTextWithAccentsWhenFoldingForTheIndexThenLengthAndLettersMatchTheQueryFolding(t *testing.T) {
	got := FoldSearchText("Đường huyết: 6,4")
	if got != "duong huyet  6 4" {
		t.Fatalf("FoldSearchText = %q", got)
	}
}

func TestGivenAMatchInsideLongTextWhenSnippetingThenTheOriginalAccentsAreMarked(t *testing.T) {
	text := strings.Repeat("Khám định kỳ hằng năm. ", 10) + "Chụp CẮT LỚP vi tính ngực không tiêm thuốc cản quang. " + strings.Repeat("Kết luận bình thường. ", 10)
	_, terms, _ := SearchQuery("cat lop")
	got, matches := joinSnippet(Snippet(text, terms, 30))
	if strings.Join(matches, "|") != "CẮT|LỚP" {
		t.Fatalf("matches = %v in %q", matches, got)
	}
	if !strings.HasPrefix(got, "… ") || !strings.HasSuffix(got, " …") {
		t.Fatalf("snippet is not cut on both sides: %q", got)
	}
	if !strings.Contains(got, "Chụp [CẮT] [LỚP] vi tính") {
		t.Fatalf("snippet = %q", got)
	}
}

func TestGivenATermInsideAWordWhenSnippetingThenOnlyWordStartsMatch(t *testing.T) {
	_, terms, _ := SearchQuery("ure")
	_, matches := joinSnippet(Snippet("Measure Ure and Urea", terms, 50))
	if strings.Join(matches, "|") != "Ure|Ure" {
		t.Fatalf("matches = %v", matches)
	}
}

func TestGivenNoMatchWhenSnippetingThenTheStartIsShown(t *testing.T) {
	got, matches := joinSnippet(Snippet("Phiếu kết quả xét nghiệm", []string{"ct"}, 50))
	if matches != nil || got != "Phiếu kết quả xét nghiệm" {
		t.Fatalf("snippet = %q, %v", got, matches)
	}
}
