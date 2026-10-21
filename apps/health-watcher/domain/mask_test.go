package domain

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// row builds a line at a vertical position; x and w are 0-1.
func row(text string, x, y, w float64) Line {
	return Line{Text: text, Confidence: 1, Quad: RectQuad(x, y, w, 0.02)}
}

func labReport() []Line {
	return []Line{
		row("SỞ Y TẾ TP. HỒ CHÍ MINH", 0.35, 0.08, 0.3),
		row("BỆNH VIỆN ĐA KHOA MẪU", 0.35, 0.10, 0.3),
		row("Khoa Xét nghiệm - ĐT: 028 0000 0000", 0.3, 0.12, 0.4),
		row("PHIẾU KẾT QUẢ XÉT NGHIỆM", 0.2, 0.16, 0.6),
		row("Họ tên: NGUYỄN VĂN MẪU Năm sinh: 1985 Giới: Nam", 0.1, 0.20, 0.7),
		row("Mã BN: 24-000123 Số thẻ BHYT: DN4790000000000", 0.1, 0.22, 0.7),
		row("Chẩn đoán: Khám sức khỏe định kỳ Ngày lấy mẫu: 14/03/2025 07:42", 0.1, 0.24, 0.8),
		row("Glucose (đói) 6.4 H mmol/L 3.9 - 6.1", 0.1, 0.30, 0.8),
		row("Bệnh nhân có tiền sử tăng huyết áp", 0.1, 0.32, 0.6),
		row("BS. TRẦN THỊ MẪU", 0.7, 0.80, 0.2),
	}
}

func maskedTexts(in MaskInput, res MaskResult) []string {
	var out []string
	for i, m := range res.Masked {
		if m {
			out = append(out, in.Lines[i].Text)
		}
	}
	return out
}

func readyInput(lines []Line) MaskInput {
	return MaskInput{Lines: lines, HasImage: true, BarcodesChecked: true, FirstPage: true, Method: MethodOCR, Width: 1654, Height: 2339}
}

func TestGivenALabReportWhenMaskingThenPersonalLinesAreHiddenAndClinicalLinesStay(t *testing.T) {
	in := readyInput(labReport())
	res := MaskPage(in)
	want := []string{
		"Khoa Xét nghiệm - ĐT: 028 0000 0000",
		"Họ tên: NGUYỄN VĂN MẪU Năm sinh: 1985 Giới: Nam",
		"Mã BN: 24-000123 Số thẻ BHYT: DN4790000000000",
		"BS. TRẦN THỊ MẪU",
	}
	if got := maskedTexts(in, res); !slices.Equal(got, want) {
		t.Fatalf("masked = %q", got)
	}
	if res.Status != MaskReady || len(res.Reasons) != 0 {
		t.Fatalf("status = %s %v", res.Status, res.Reasons)
	}
	if len(res.Boxes) != len(want) {
		t.Fatalf("boxes = %d", len(res.Boxes))
	}
}

func TestGivenALabelWithItsValueOnTheRightWhenMaskingThenTheValueIsHiddenToo(t *testing.T) {
	lines := []Line{row("Họ tên:", 0.1, 0.2, 0.1), row("NGUYỄN VĂN A", 0.25, 0.2, 0.3), row("Xét nghiệm máu", 0.1, 0.4, 0.3)}
	in := readyInput(lines)
	if got := maskedTexts(in, MaskPage(in)); !slices.Equal(got, []string{"Họ tên:", "NGUYỄN VĂN A"}) {
		t.Fatalf("masked = %q", got)
	}
}

func TestGivenALabelWithItsValueBelowWhenMaskingThenTheLineBelowIsHidden(t *testing.T) {
	lines := []Line{row("Địa chỉ:", 0.1, 0.20, 0.15), row("12 Đường Mẫu, Quận Mẫu", 0.1, 0.225, 0.4), row("Glucose 5.0", 0.1, 0.5, 0.3)}
	in := readyInput(lines)
	if got := maskedTexts(in, MaskPage(in)); !slices.Equal(got, []string{"Địa chỉ:", "12 Đường Mẫu, Quận Mẫu"}) {
		t.Fatalf("masked = %q", got)
	}
}

func TestGivenAPersonsNameMisreadWithoutAccentsWhenMaskingThenItIsStillHidden(t *testing.T) {
	lines := []Line{row("NGUYÊN VĂN MÀU", 0.1, 0.1, 0.3), row("Glucose 5.0", 0.1, 0.5, 0.3)}
	in := readyInput(lines)
	in.People = []string{"Nguyễn Văn Mẫu", "An"}
	res := MaskPage(in)
	if got := maskedTexts(in, res); !slices.Equal(got, []string{"NGUYÊN VĂN MÀU"}) {
		t.Fatalf("masked = %q", got)
	}
	if res.Status != MaskReady {
		t.Fatalf("a matched person counts as the name: %s %v", res.Status, res.Reasons)
	}
}

func TestGivenAFirstPageWithoutANameWhenMaskingThenItWaitsForACheckUntilChecked(t *testing.T) {
	in := readyInput([]Line{row("Glucose 5.0", 0.1, 0.5, 0.3)})
	res := MaskPage(in)
	if res.Status != MaskNeedsCheck || !slices.Contains(res.Reasons, ReasonNoName) {
		t.Fatalf("status = %s %v", res.Status, res.Reasons)
	}
	in.Checked = true
	if res := MaskPage(in); res.Status != MaskChecked {
		t.Fatalf("checked status = %s", res.Status)
	}
	in.Checked, in.FirstPage = false, false
	if res := MaskPage(in); res.Status != MaskReady {
		t.Fatalf("a later page needs no name: %s %v", res.Status, res.Reasons)
	}
}

func TestGivenAnOCRToolWithoutBarcodesOrImagesWhenMaskingThenThePageWaits(t *testing.T) {
	in := readyInput(labReport())
	in.BarcodesChecked, in.HasImage = false, false
	res := MaskPage(in)
	if res.Status != MaskNeedsCheck || !slices.Contains(res.Reasons, ReasonNoBarcodes) || !slices.Contains(res.Reasons, ReasonNoImage) {
		t.Fatalf("status = %s %v", res.Status, res.Reasons)
	}
}

func TestGivenBlurryTextNearTheTopWhenMaskingThenThePageWaits(t *testing.T) {
	lines := labReport()
	lines[3].Confidence = 0.3
	res := MaskPage(readyInput(lines))
	if !slices.Contains(res.Reasons, ReasonBlurryTop) {
		t.Fatalf("reasons = %v", res.Reasons)
	}
}

func TestGivenRowOrderedTextWhenMaskingThenPersonalRowsAndNumbersAreReplaced(t *testing.T) {
	in := readyInput(labReport())
	in.Text = strings.Join([]string{
		"BỆNH VIỆN ĐA KHOA MẪU",
		"Họ tên: NGUYỄN VĂN MẪU Năm sinh: 1985 Giới: Nam",
		"Mã BN: 24-000123 Số thẻ BHYT: DN4790000000000",
		"Liên hệ 0900 000 000 hoặc mau@example.test",
		"STT Tên xét nghiệm Kết quả",
		"1 Glucose (đói) 6.4 H mmol/L 3.9 - 6.1",
		"BS. TRẦN THỊ MẪU",
	}, "\n")
	got := MaskPage(in).Text
	want := strings.Join([]string{
		"BỆNH VIỆN ĐA KHOA MẪU",
		MaskPlaceholder, // name, record numbers and the contact row, as one run
		"STT Tên xét nghiệm Kết quả",
		"1 Glucose (đói) 6.4 H mmol/L 3.9 - 6.1",
		MaskPlaceholder,
	}, "\n")
	if got != want {
		t.Fatalf("masked text:\n%s\nwant:\n%s", got, want)
	}
}

func TestGivenAMaskedLineTheTextRulesMissWhenMaskingThenTheOCRLinesAreUsed(t *testing.T) {
	// The value line is masked by position only; the page text joins it
	// with clinical text, so the text rules can't see it.
	lines := []Line{row("Họ tên:", 0.1, 0.2, 0.1), row("TRẦN VĂN B", 0.25, 0.2, 0.3), row("Glucose 5.0", 0.1, 0.4, 0.3)}
	in := readyInput(lines)
	in.Text = "Glucose 5.0 TRẦN  VĂN B\nHọ tên:"
	got := MaskPage(in).Text
	if strings.Contains(FoldKey(got), "tran van b") {
		t.Fatalf("name leaked: %q", got)
	}
	if got != MaskPlaceholder+"\nGlucose 5.0" {
		t.Fatalf("fallback text = %q", got)
	}
}

func TestGivenAClinicalSentenceWhenMaskingThenWordsLikeBenhNhanNeedAColon(t *testing.T) {
	if _, _, personal := personalLine("Bệnh nhân có tiền sử tăng huyết áp", nil); personal {
		t.Fatal("clinical sentence masked")
	}
	if _, _, personal := personalLine("Bệnh nhân: Nguyễn Văn C", nil); !personal {
		t.Fatal("labelled patient line kept")
	}
}

func TestGivenARotatedQuadWhenExpandingThenItGrowsAlongItsOwnAxes(t *testing.T) {
	// A 100x20 px box rotated 90 degrees on a 1000x1000 page.
	q := Quad{{0.5, 0.1}, {0.5, 0.2}, {0.48, 0.2}, {0.48, 0.1}}
	got := q.Expand(0.5, 1000, 1000)
	// Height is 20 px, so each side grows by 10 px = 0.01.
	want := Quad{{0.51, 0.09}, {0.51, 0.21}, {0.47, 0.21}, {0.47, 0.09}}
	for i := range got {
		if math.Abs(got[i].X-want[i].X) > 1e-9 || math.Abs(got[i].Y-want[i].Y) > 1e-9 {
			t.Fatalf("corner %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGivenPageTextsWhenMatchingPeopleThenOnlyOneFullNameMatchCounts(t *testing.T) {
	people := []Person{{ID: "a", DisplayName: "Nguyễn Văn Mẫu"}, {ID: "b", DisplayName: "Trần Thị Mẫu"}, {ID: "c", DisplayName: "Bé"}}
	if got := MatchPerson([]string{"Họ tên: NGUYÊN VĂN MÀU Năm sinh: 1985"}, people); got != "a" {
		t.Fatalf("match = %q", got)
	}
	if got := MatchPerson([]string{"Họ tên: NGUYỄN VĂN MẪU", "Người nhà: TRẦN THỊ MẪU"}, people); got != "" {
		t.Fatalf("two people matched as %q", got)
	}
	if got := MatchPerson([]string{"Glucose 6.4"}, people); got != "" {
		t.Fatalf("no name matched as %q", got)
	}
}

func TestGivenATextLayerPageWhenMaskingThenBoxesBarelyGrowBecausePDFLineBoxesAlreadyHoldTheGlyphs(t *testing.T) {
	lines := []Line{row("Họ tên: NGUYỄN VĂN MẪU", 0.1, 0.2, 0.5)}
	ocr := MaskPage(readyInput(lines))
	in := readyInput(lines)
	in.Method = MethodText
	text := MaskPage(in)
	height := func(q Quad) float64 { return q[3].Y - q[0].Y }
	if got, want := height(ocr.Boxes[0]), 0.02*1.6; math.Abs(got-want) > 1e-9 {
		t.Fatalf("OCR box height = %v, want %v", got, want)
	}
	if got, want := height(text.Boxes[0]), 0.02*1.1; math.Abs(got-want) > 1e-9 {
		t.Fatalf("text-layer box height = %v, want %v", got, want)
	}
}
