package domain

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

// MaskPlaceholder replaces a personal line in text sent to the AI provider.
const MaskPlaceholder = "[ẩn]"

// MaskRulesVersion goes up when the masking rules change, so pages masked
// with older rules are masked again.
const MaskRulesVersion = 5

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
	// signature marks a sign-here prompt such as `(Ký và ghi rõ họ tên)`:
	// the name is written below it, past the space left for the signature.
	signature bool
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
		{"ma ho so", l{}}, {"so ho so", l{}}, {"ho so", l{needColon: true}}, {"so luu tru", l{}}, {"so vao vien", l{}}, {"ma vao vien", l{}},
		{"patient id", l{}}, {"pid", l{needColon: true}}, {"bhyt", l{}}, {"so the", l{}}, {"the bhyt", l{}},
		{"cccd", l{}}, {"cmnd", l{}}, {"can cuoc", l{}}, {"ho chieu", l{}}, {"passport", l{}},
		{"dia chi", l{}}, {"address", l{}}, {"dien thoai", l{}}, {"so dien thoai", l{}}, {"sdt", l{}},
		{"dt", l{needColon: true}}, {"phone", l{}}, {"mobile", l{}}, {"email", l{}},
		{"ten dang nhap", l{}}, {"mat khau", l{}}, {"username", l{}}, {"password", l{}},
		{"bac si", l{needColon: true}}, {"bs", l{needColon: true, allowDot: true}},
		{"ths bs", l{needColon: true, allowDot: true}}, {"dr", l{needColon: true, allowDot: true}},
		{"nguoi lay mau", l{needColon: true}}, {"nguoi thuc hien", l{needColon: true}},
		{"ky thuat vien", l{needColon: true}}, {"ktv", l{needColon: true, allowDot: true}},
		{"nguoi nha", l{needColon: true}}, {"nguoi giam ho", l{needColon: true}},
		{"ghi ro ho ten", l{signature: true}}, {"ghi ro ho va ten", l{signature: true}}, {"ky ho ten", l{signature: true}},
		{"ky ten", l{signature: true}}, {"ky va ho ten", l{signature: true}}, {"signature", l{signature: true}},
	}
	out := make([]personalLabel, len(raw))
	for i, r := range raw {
		r.l.words = strings.Fields(r.text)
		out[i] = r.l
	}
	return out
}()

var (
	// digitRun finds runs of digits split by single spaces, dots or dashes;
	// longDigitRuns keeps the ID, insurance card and phone numbers.
	digitRun = regexp.MustCompile(`\d(?:[ .\-]?\d){8,}`)
	email    = regexp.MustCompile(`[^\s@]+@[^\s@]+\.[A-Za-z]{2,}`)
)

// minLongDigits is the digit count of the shortest ID or phone number.
const minLongDigits = 9

// longDigitRuns returns the byte spans of long digit runs: ID, insurance
// card and phone numbers. A dot followed by fewer than three digits is a
// decimal point, not a separator, so results and reference ranges such as
// `62.00-106.00` don't count.
func longDigitRuns(s string) [][2]int {
	var out [][2]int
	for _, m := range digitRun.FindAllStringIndex(s, -1) {
		start := m[0]
		for k := m[0]; k <= m[1]; k++ {
			if k < m[1] && (s[k] != '.' || digitsAt(s[k+1:m[1]]) >= 3) {
				continue
			}
			if countDigits(s[start:k]) >= minLongDigits {
				out = append(out, [2]int{start, k})
			}
			start = k + 1
		}
	}
	return out
}

func countDigits(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n++
		}
	}
	return n
}

// digitsAt counts the digits at the start of s.
func digitsAt(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

func hasLongDigits(s string) bool { return len(longDigitRuns(s)) > 0 }

// hideNumbers replaces long digit runs and e-mail addresses.
func hideNumbers(s string) string {
	runs := longDigitRuns(s)
	for i := len(runs) - 1; i >= 0; i-- {
		s = s[:runs[i][0]] + MaskPlaceholder + s[runs[i][1]:]
	}
	return email.ReplaceAllString(s, MaskPlaceholder)
}

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

// hasName reports whether a line holds one of the names: its words next to
// each other in any order, since a paper may print the family name first
// or last and a film's label prints `NGUYEN, VAN MAU`. With partial, a name
// of four or more words also matches with one word missing, as on a film
// label cut off at the edge; MatchPerson leaves it out, because two
// siblings share all but one word of their names.
func hasName(line string, names [][]string, partial bool) bool {
	toks, _ := tokenize(line)
	words := make([]string, len(toks))
	for i, t := range toks {
		words[i] = t.word
	}
	for _, n := range names {
		if hasWordsInAnyOrder(words, n, len(n)) || (partial && len(n) >= 4 && hasWordsInAnyOrder(words, n, len(n)-1)) {
			return true
		}
	}
	return false
}

// hasWordsInAnyOrder reports whether size words in a row of words are all
// words of name, each used at most once.
func hasWordsInAnyOrder(words, name []string, size int) bool {
outer:
	for i := 0; i+size <= len(words); i++ {
		left := slices.Clone(name)
		for _, w := range words[i : i+size] {
			k := slices.Index(left, w)
			if k < 0 {
				continue outer
			}
			left = slices.Delete(left, k, k+1)
		}
		return true
	}
	return false
}

// personalLine reports whether a line holds personal data by itself.
func personalLine(line string, names [][]string) (labelMatch, bool, bool) {
	m, labeled := findLabel(line)
	if labeled {
		return m, true, true
	}
	if hasName(line, names, true) {
		return labelMatch{label: personalLabel{name: true}}, false, true
	}
	return labelMatch{}, false, hasLongDigits(line) || email.MatchString(line)
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
	// Off are the keys of rule boxes a person turned off.
	Off []string
	// Checked is true once a person checked the masked preview.
	Checked       bool
	Width, Height int
}

// MaskResult is the masked page.
type MaskResult struct {
	Masked  []bool // per input line, without the lines turned off
	Boxes   []Quad // black boxes on the image, manual ones included
	Auto    []MaskBox
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
		if labeled && m.label.signature {
			for _, j := range signedName(in.Lines, i) {
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
	res.Auto = []MaskBox{}
	off := make([]bool, len(in.Lines))
	for i, l := range in.Lines {
		if !res.Masked[i] {
			continue
		}
		b := MaskBox{Key: fmt.Sprintf("line:%d", i), Kind: MaskBoxLine, Quad: l.Quad.Expand(pad, in.Width, in.Height), Text: l.Text}
		b.Off = slices.Contains(in.Off, b.Key)
		res.Auto = append(res.Auto, b)
		if b.Off {
			res.Masked[i], off[i] = false, true
		} else {
			res.Boxes = append(res.Boxes, b.Quad)
		}
	}
	for i, q := range in.Barcodes {
		b := MaskBox{Key: fmt.Sprintf("barcode:%d", i), Kind: MaskBoxBarcode, Quad: q}
		b.Off = slices.Contains(in.Off, b.Key)
		res.Auto = append(res.Auto, b)
		if !b.Off {
			res.Boxes = append(res.Boxes, q)
		}
	}
	res.Boxes = append(res.Boxes, in.Manual...)
	// A line under a box a person drew is hidden in the text too, even one
	// whose rule box they turned off: the text must not send what the image
	// hides.
	for i, l := range in.Lines {
		if slices.ContainsFunc(in.Manual, func(q Quad) bool { return covers(q, l.Quad, in.Width, in.Height) }) {
			res.Masked[i], off[i] = true, false
		}
	}
	res.Text = maskText(in.Text, in.Lines, res.Masked, off, names, &nameFound)

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

// signedName finds the lines under a sign-here prompt that may hold the
// signer's name: every line in its column that starts within eight line
// heights below it. The signature in between often reads as stray letters,
// so the first line below is not enough.
func signedName(lines []Line, i int) []int {
	lx0, ly0, lx1, ly1 := lines[i].Quad.Bounds()
	lh := ly1 - ly0
	var out []int
	for j, l := range lines {
		if j == i {
			continue
		}
		x0, y0, x1, _ := l.Quad.Bounds()
		if y0 >= ly1-0.2*lh && y0-ly1 <= 8*lh && x1 > lx0 && x0 < lx1 {
			out = append(out, j)
		}
	}
	return out
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
// then removes long digit runs and e-mail addresses anywhere. The lines a
// person turned off (off) stay as they are, and the rules run on the rest
// of their row. If a masked line still shows after that, it falls back to
// the OCR lines in order.
func maskText(text string, lines []Line, masked, off []bool, names [][]string, nameFound *bool) string {
	if strings.TrimSpace(text) == "" {
		return linesText(lines, masked, off)
	}
	var kept []string
	for i, l := range lines {
		if off[i] && strings.TrimSpace(l.Text) != "" {
			kept = append(kept, strings.TrimSpace(l.Text))
		}
	}
	rows := strings.Split(text, "\n")
	parts := make([][]part, len(rows))
	for i, row := range rows {
		parts[i] = splitKept(row, kept)
	}
	hideRow := func(j int) {
		for k := range parts[j] {
			parts[j][k].hidden = !parts[j][k].kept
		}
	}
	for i := range rows {
		for k, p := range parts[i] {
			if p.kept {
				continue
			}
			m, labeled, personal := personalLine(p.text, names)
			if !personal {
				continue
			}
			parts[i][k].hidden = true
			*nameFound = *nameFound || m.label.name
			if labeled && m.empty && k == len(parts[i])-1 {
				for j := i + 1; j < len(rows); j++ {
					if strings.TrimSpace(rows[j]) != "" {
						hideRow(j)
						break
					}
				}
			}
		}
	}
	var checked []string
	for i := range rows {
		var b strings.Builder
		for _, p := range parts[i] {
			switch {
			case p.kept:
				b.WriteString(p.text)
			case p.hidden:
				lead := p.text[:len(p.text)-len(strings.TrimLeft(p.text, " \t"))]
				trail := p.text[len(strings.TrimRight(p.text, " \t")):]
				if strings.TrimSpace(p.text) == "" {
					lead, trail = p.text, ""
				}
				b.WriteString(lead + MaskPlaceholder + trail)
			default:
				t := scrubPart(p.text, lines, masked)
				b.WriteString(t)
				checked = append(checked, t)
			}
		}
		rows[i] = b.String()
	}
	if leaks(strings.Join(checked, "\n"), lines, masked, names) {
		return linesText(lines, masked, off)
	}
	return collapsePlaceholders(rows)
}

// part is a piece of a row of text. Kept parts are lines a person turned
// off; hidden parts become the placeholder.
type part struct {
	text   string
	kept   bool
	hidden bool
}

// splitKept splits a row at the lines a person turned off. A row that is
// one of those lines, ignoring spacing, is kept whole.
func splitKept(row string, kept []string) []part {
	for _, k := range kept {
		if rowKey(row) == rowKey(k) {
			return []part{{text: row, kept: true}}
		}
	}
	out := []part{}
	for row != "" {
		at, n := -1, 0
		for _, k := range kept {
			if i := strings.Index(row, k); i >= 0 && (at < 0 || i < at || (i == at && len(k) > n)) {
				at, n = i, len(k)
			}
		}
		if at < 0 {
			out = append(out, part{text: row})
			break
		}
		if at > 0 {
			out = append(out, part{text: row[:at]})
		}
		out = append(out, part{text: row[at : at+n], kept: true})
		row = row[at+n:]
	}
	return out
}

// scrubPart replaces the masked OCR lines, long digit runs and e-mail
// addresses in a piece of text.
func scrubPart(text string, lines []Line, masked []bool) string {
	for i, l := range lines {
		if masked[i] && strings.TrimSpace(l.Text) != "" {
			text = strings.ReplaceAll(text, l.Text, MaskPlaceholder)
		}
	}
	return hideNumbers(text)
}

// rowKey compares a row of text with an OCR line, ignoring spacing.
func rowKey(s string) string {
	return strings.Join(strings.Fields(s), " ")
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
	return hasName(out, names, true)
}

func linesText(lines []Line, masked, off []bool) string {
	rows := make([]string, len(lines))
	for i, l := range lines {
		switch {
		case masked[i]:
			rows[i] = MaskPlaceholder
		case off[i]:
			rows[i] = l.Text
		default:
			rows[i] = hideNumbers(l.Text)
		}
	}
	return collapsePlaceholders(rows)
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
			if hasName(t, names, false) {
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

// covers reports whether box b hides some of line l: it spans at least half
// the line's height and at least one character of it, taken as the line's
// height in pixels. A box over the line above that grazes this one does not
// count. Both use their axis-aligned bounds.
func covers(b, l Quad, width, height int) bool {
	bx0, by0, bx1, by1 := b.Bounds()
	lx0, ly0, lx1, ly1 := l.Bounds()
	w, h := float64(width), float64(height)
	if width <= 0 || height <= 0 {
		w, h = 1, 1
	}
	lineH := (ly1 - ly0) * h
	overlapY := (math.Min(by1, ly1) - math.Max(by0, ly0)) * h
	overlapX := (math.Min(bx1, lx1) - math.Max(bx0, lx0)) * w
	return lineH > 0 && overlapY >= lineH/2 && overlapX >= math.Min(lineH, (lx1-lx0)*w)
}
