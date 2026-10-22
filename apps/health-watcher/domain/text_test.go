package domain

import "testing"

func TestGivenProviderNamesWithAccentsAndSpacingWhenFoldingThenTheyShareAKey(t *testing.T) {
	cases := map[string]string{
		"BV Chợ Rẫy":          "bv cho ray",
		"  bv  CHO-RAY ":      "bv cho ray",
		"Đường huyết":         "duong huyet",
		"Phòng khám Nha khoa": "phong kham nha khoa",
		"HbA1c (%)":           "hba1c",
		"":                    "",
	}
	for in, want := range cases {
		if got := FoldKey(in); got != want {
			t.Errorf("FoldKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGivenTextWithExtraWhitespaceWhenCleaningThenSpacesCollapse(t *testing.T) {
	if got := CleanText("  Nha   khoa\tKim \n"); got != "Nha khoa Kim" {
		t.Fatalf("CleanText = %q", got)
	}
}

func TestGivenMultiLineTextWhenCleaningLinesThenEachLineIsCleanedAndBlankLinesGo(t *testing.T) {
	if got := CleanLines("  MÔ TẢ   HÌNH ẢNH \r\n\n • Phổi  phải: đồng nhất.\n\t\nKẾT LUẬN\n"); got != "MÔ TẢ HÌNH ẢNH\n• Phổi phải: đồng nhất.\nKẾT LUẬN" {
		t.Fatalf("CleanLines = %q", got)
	}
}
