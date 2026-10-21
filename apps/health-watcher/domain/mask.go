package domain

import (
	"math"
	"regexp"
	"strings"
)

// MaskPlaceholder replaces a personal line in text sent to the AI provider.
const MaskPlaceholder = "[ẩn]"

// personalLabel is a label printed before personal data on Vietnamese and
// English medical papers. Labels that also occur in ordinary sentences
// count only when a colon follows them (`Bệnh nhân: ...`); BS and Dr also
// count before a dot (`BS. Trần`).
type personalLabel struct {
	words     []string
	needColon bool
	allowDot  bool
	// name marks a label for the patient's name; finding one means the
	// header was read (HW-AI-7).
	name bool
}

var personalLabels = func() []personalLabel {
	type l = personalLabel
	raw := []struct {
		text string
		l    l
	}{
		{"ho ten", l{name: true}}, {"ho va ten", l{name: true}}, {"ten benh nhan", l{name: true}},
		{"ten nguoi benh", l{name: true}}, {"ho ten benh nhan", l{name: true}}, {"patient name", l{name: true}},
		{"full name", l{name: true}},
		{"benh nhan", l{needColon: true, name: true}}, {"nguoi benh", l{needColon: true, name: true}},
		{"khach hang", l{needColon: true, name: true}}, {"ten", l{needColon: true, name: true}},
		{"name", l{needColon: true, name: true}}, {"patient", l{needColon: true, name: true}},
		{"nam sinh", l{}}, {"ngay sinh", l{}}, {"sinh ngay", l{}}, {"date of birth", l{}}, {"dob", l{}},
		{"ma bn", l{}}, {"ma benh nhan", l{}}, {"ma so benh nhan", l{}}, {"ma nguoi benh", l{}}, {"ma y te", l{}},
		{"ma ho so", l{}}, {"so ho so", l{}}, {"so luu tru", l{}}, {"so vao vien", l{}}, {"ma vao vien", l{}},
		{"patient id", l{}}, {"pid", l{needColon: true}}, {"bhyt", l{}}, {"so the", l{}}, {"the bhyt", l{}},
		{"cccd", l{}}, {"cmnd", l{}}, {"can cuoc", l{}}, {"ho chieu", l{}}, {"passport", l{}},
		{"dia chi", l{}}, {"address", l{}}, {"dien thoai", l{}}, {"so dien thoai", l{}}, {"sdt", l{}},
		{"dt", l{needColon: true}}, {"phone", l{}}, {"mobile", l{}}, {"email", l{}},
		{"bac si", l{needColon: true}}, {"bs", l{needColon: true, allowDot: true}},
		{"ths bs", l{needColon: true, allowDot: true}}, {"dr", l{needColon: true, allowDot: true}},
		{"nguoi lay mau", l{needColon: true}}, {"nguoi thuc hien", l{needColon: true}},
		{"ky thuat vien", l{needColon: true}}, {"ktv", l{needColon: true, allowDot: true}},
		{"nguoi nha", l{needColon: true}}, {"nguoi giam ho", l{needColon: true}},
	}
	out := make([]personalLabel, len(raw))
	for i, r := range raw {
		r.l.words = strings.Fields(r.text)
		out[i] = r.l
	}
	return out
}()

var (
	// Long digit runs, possibly split by single spaces, dots or dashes: ID,
	// insurance card and phone numbers.
	longDigits = regexp.MustCompile(`\d(?:[ .\-]?\d){8,}`)
	email      = regexp.MustCompile(`[^\s@]+@[^\s@]+\.[A-Za-z]{2,}`)
)

// token is a folded word and the span of the original text it came from.
type token struct {
	word       string
	start, end int // rune offsets in the NFC original
}

func tokenize(s string) ([]token, []rune) {
	folded, origin, original := foldMap(s)
	var out []token
	for i := 0; i < len(folded); {
		if folded[i] == ' ' {
			i++
			continue
		}
		j := i
		for j < len(folded) && folded[j] != ' ' {
			j++
		}
		out = append(out, token{word: string(folded[i:j]), start: origin[i], end: origin[j-1] + 1})
		i = j
	}
	return out, original
}

// findWords returns the token index where words start in toks, or -1.
func findWords(toks []token, words []string, from int) int {
outer:
	for i := from; i+len(words) <= len(toks); i++ {
		for k, w := range words {
			if toks[i+k].word != w {
				continue outer
			}
		}
		return i
	}
	return -1
}

// labelMatch is a personal label found in a line.
type labelMatch struct {
	label personalLabel
	// empty is true when nothing follows the label on the line, so the
	// value sits on the next line or further right.
	empty bool
}

// findLabel looks for a personal label in a line of text.
func findLabel(line string) (labelMatch, bool) {
	toks, original := tokenize(line)
	var best *labelMatch
	bestLen := 0
	for _, lab := range personalLabels {
		for i := findWords(toks, lab.words, 0); i >= 0; i = findWords(toks, lab.words, i+1) {
			end := toks[i+len(lab.words)-1].end
			rest := strings.TrimLeft(string(original[end:]), " \t")
			punct := strings.HasPrefix(rest, ":") || (lab.allowDot && strings.HasPrefix(rest, "."))
			if lab.needColon && !punct {
				continue
			}
			if punct {
				rest = rest[1:]
			}
			if len(lab.words) > bestLen {
				m := labelMatch{label: lab, empty: strings.TrimSpace(strings.Trim(rest, ":. ")) == ""}
				best, bestLen = &m, len(lab.words)
			}
			break
		}
	}
	if best == nil {
		return labelMatch{}, false
	}
	return *best, true
}

// nameTokens folds the names of the tenant's people. Single-word names are
// left out: a short given name such as `An` also matches ordinary words, so
// those papers rely on the labels instead.
func nameTokens(people []string) [][]string {
	var out [][]string
	for _, p := range people {
		if words := strings.Fields(FoldKey(p)); len(words) >= 2 {
			out = append(out, words)
		}
	}
	return out
}

func hasName(line string, names [][]string) bool {
	toks, _ := tokenize(line)
	for _, n := range names {
		if findWords(toks, n, 0) >= 0 {
			return true
		}
	}
	return false
}

// personalLine reports whether a line holds personal data by itself.
func personalLine(line string, names [][]string) (labelMatch, bool, bool) {
	m, labeled := findLabel(line)
	if labeled {
		return m, true, true
	}
	if hasName(line, names) {
		return labelMatch{label: personalLabel{name: true}}, false, true
	}
	return labelMatch{}, false, longDigits.MatchString(line) || email.MatchString(line)
}

// MaskInput is one page as the OCR tool read it, plus what the engine knows
// about the tenant.
type MaskInput struct {
	Lines           []Line
	Text            string
	Barcodes        []Quad
	BarcodesChecked bool
	HasImage        bool
	// FirstPage pages must show a patient name the engine can find.
	FirstPage bool
	Method    PageMethod
	People    []string
	Manual    []Quad
	// Checked is true once a person checked the masked preview.
	Checked       bool
	Width, Height int
}

// MaskResult is the masked page.
type MaskResult struct {
	Masked  []bool // per input line
	Boxes   []Quad // black boxes on the image, manual ones included
	Text    string
	Status  MaskStatus
	Reasons []string
}

// Reasons a page waits for a check (HW-AI-7).
const (
	ReasonNoImage    = "This page has no rendered image. Rebuild mac-ocr so it can render pages."
	ReasonNoBarcodes = "mac-ocr could not look for barcodes or QR codes on this page."
	ReasonNoName     = "The engine could not find the patient's name on this page."
	ReasonBlurryTop  = "Some text near the top of the page is hard to read, so a name could be missed."
	ReasonNoLines    = "mac-ocr gave no line positions for this page, so nothing could be blacked out."
)

// MaskPage finds the personal lines of a page and builds what the AI
// provider may see: the boxes to black out on the image and the text with
// those lines replaced. When the engine can't be sure it found the
// patient's name, the page waits for a person to check it.
func MaskPage(in MaskInput) MaskResult {
	names := nameTokens(in.People)
	res := MaskResult{Masked: make([]bool, len(in.Lines)), Reasons: []string{}}
	nameFound := false
	for i, l := range in.Lines {
		m, labeled, personal := personalLine(l.Text, names)
		if !personal {
			continue
		}
		res.Masked[i] = true
		nameFound = nameFound || m.label.name
		if labeled && m.empty {
			if j := valueLine(in.Lines, i); j >= 0 {
				res.Masked[j] = true
			}
		}
	}

	// Vision boxes hug the glyphs, so they grow by 30% of the line height on
	// each side; PDF text-layer boxes already hold ascenders and descenders,
	// and growing them as much covers the neighbouring lines.
	pad := 0.3
	if in.Method == MethodText {
		pad = 0.05
	}
	for i, l := range in.Lines {
		if res.Masked[i] {
			res.Boxes = append(res.Boxes, l.Quad.Expand(pad, in.Width, in.Height))
		}
	}
	res.Boxes = append(res.Boxes, in.Barcodes...)
	res.Boxes = append(res.Boxes, in.Manual...)
	res.Text = maskText(in.Text, in.Lines, res.Masked, names, &nameFound)

	if !in.HasImage {
		res.Reasons = append(res.Reasons, ReasonNoImage)
	}
	if !in.BarcodesChecked {
		res.Reasons = append(res.Reasons, ReasonNoBarcodes)
	}
	if len(in.Lines) == 0 && strings.TrimSpace(in.Text) != "" {
		res.Reasons = append(res.Reasons, ReasonNoLines)
	}
	if in.FirstPage && !nameFound {
		res.Reasons = append(res.Reasons, ReasonNoName)
	}
	if in.Method == MethodOCR && blurryTop(in.Lines) {
		res.Reasons = append(res.Reasons, ReasonBlurryTop)
	}
	switch {
	case in.Checked:
		res.Status = MaskChecked
	case len(res.Reasons) > 0:
		res.Status = MaskNeedsCheck
	default:
		res.Status = MaskReady
	}
	return res
}

// valueLine finds the line holding the value of a label that has nothing
// after it: the nearest line to its right on the same row, or else the
// line just below it.
func valueLine(lines []Line, i int) int {
	lx0, ly0, lx1, ly1 := lines[i].Quad.Bounds()
	lh := ly1 - ly0
	right, below := -1, -1
	bestRight, bestBelow := math.Inf(1), math.Inf(1)
	for j, l := range lines {
		if j == i {
			continue
		}
		x0, y0, x1, y1 := l.Quad.Bounds()
		overlap := math.Min(ly1, y1) - math.Max(ly0, y0)
		if overlap >= 0.5*math.Min(lh, y1-y0) && x0 >= lx1-0.01 && x0-lx1 < bestRight {
			right, bestRight = j, x0-lx1
		}
		if y0 >= ly1-0.2*lh && y0-ly1 <= 1.5*lh && x1 > lx0 && x0 < lx1+0.3 && y0-ly1 < bestBelow {
			below, bestBelow = j, y0-ly1
		}
	}
	if right >= 0 {
		return right
	}
	return below
}

// blurryTop reports a line Vision was unsure of in the top third of the page,
// where the patient's details usually are.
func blurryTop(lines []Line) bool {
	for _, l := range lines {
		_, _, _, y1 := l.Quad.Bounds()
		if l.Confidence < 0.5 && y1 <= 0.35 {
			return true
		}
	}
	return false
}

// maskText replaces personal lines in the page text. It applies the line
// rules to each line of the text itself (a label line with no value masks
// the next line too), replaces the masked OCR lines wherever they appear,
// then removes long digit runs and e-mail addresses anywhere. If a masked
// line still shows after that, it falls back to the OCR lines in order.
func maskText(text string, lines []Line, masked []bool, names [][]string, nameFound *bool) string {
	if strings.TrimSpace(text) == "" {
		return linesText(lines, masked)
	}
	rows := strings.Split(text, "\n")
	hide := make([]bool, len(rows))
	for i, row := range rows {
		m, labeled, personal := personalLine(row, names)
		if !personal {
			continue
		}
		hide[i] = true
		*nameFound = *nameFound || m.label.name
		if labeled && m.empty {
			for j := i + 1; j < len(rows); j++ {
				if strings.TrimSpace(rows[j]) != "" {
					hide[j] = true
					break
				}
			}
		}
	}
	for i := range rows {
		if hide[i] {
			rows[i] = MaskPlaceholder
		}
	}
	out := collapsePlaceholders(rows)
	for i, l := range lines {
		if masked[i] && strings.TrimSpace(l.Text) != "" {
			out = strings.ReplaceAll(out, l.Text, MaskPlaceholder)
		}
	}
	out = longDigits.ReplaceAllString(out, MaskPlaceholder)
	out = email.ReplaceAllString(out, MaskPlaceholder)
	if leaks(out, lines, masked, names) {
		return linesText(lines, masked)
	}
	return out
}

// leaks reports whether a masked line or a person's name still shows in
// the masked text, compared without accents.
func leaks(out string, lines []Line, masked []bool, names [][]string) bool {
	folded := " " + strings.Join(strings.Fields(FoldKey(out)), " ") + " "
	for i, l := range lines {
		if !masked[i] {
			continue
		}
		key := FoldKey(l.Text)
		if len(key) >= 4 && strings.Contains(folded, " "+key+" ") {
			return true
		}
	}
	for _, n := range names {
		if strings.Contains(folded, " "+strings.Join(n, " ")+" ") {
			return true
		}
	}
	return false
}

func linesText(lines []Line, masked []bool) string {
	rows := make([]string, len(lines))
	for i, l := range lines {
		rows[i] = l.Text
		if masked[i] {
			rows[i] = MaskPlaceholder
		}
	}
	out := collapsePlaceholders(rows)
	out = longDigits.ReplaceAllString(out, MaskPlaceholder)
	return email.ReplaceAllString(out, MaskPlaceholder)
}

// collapsePlaceholders joins rows, writing one placeholder for a run of
// masked rows so the text doesn't show how many lines were hidden.
func collapsePlaceholders(rows []string) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r == MaskPlaceholder && len(out) > 0 && out[len(out)-1] == MaskPlaceholder {
			continue
		}
		out = append(out, r)
	}
	return strings.Join(out, "\n")
}

// MatchPerson finds the tenant's person whose full name appears in the
// page texts, compared without accents, before any masking. It runs on
// the Mac only; the AI provider never sees the name (HW-READ-3). No match,
// or more than one person (a parent named on a child's paper), gives "".
func MatchPerson(texts []string, people []Person) string {
	found := ""
	for _, p := range people {
		names := nameTokens([]string{p.DisplayName})
		if len(names) == 0 {
			continue
		}
		for _, t := range texts {
			if hasName(t, names) {
				if found != "" && found != p.ID {
					return ""
				}
				found = p.ID
				break
			}
		}
	}
	return found
}
