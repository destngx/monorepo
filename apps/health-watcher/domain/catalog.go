package domain

import (
	"math"
	"strings"
	"unicode"
)

// MetricCategory groups metrics on screen (HW-MET-1).
type MetricCategory string

// Conversion turns a value in From into the metric's canonical unit:
// canonical = value*Factor + Offset.
type Conversion struct {
	From   string  `json:"from"`
	Factor float64 `json:"factor"`
	Offset float64 `json:"offset,omitempty"`
}

// Metric is one entry of the catalog: a test or a vital sign.
type Metric struct {
	Code     string         `json:"code"`
	LOINC    string         `json:"loinc"`
	Name     string         `json:"name"`
	NameVI   string         `json:"nameVi"`
	Category MetricCategory `json:"category"`
	// Unit is the canonical unit; "" for results read as text (Âm tính).
	Unit        string       `json:"unit"`
	Conversions []Conversion `json:"conversions,omitempty"`
	Aliases     []string     `json:"aliases"`
	// Urine marks a urine test whose names are shared with a blood test
	// (glucose, protein); it matches a blood name only in a urine context.
	Urine bool `json:"urine,omitempty"`
	// WholeName marks a test whose names begin the names of other tests
	// (`Số lượng bạch cầu đoạn` is not the white cell count), so a longer
	// printed name never matches it by its start.
	WholeName bool `json:"-"`
}

// Catalog is the shipped metric list with its matching index.
type Catalog struct {
	Version string
	metrics []Metric
	byCode  map[string]int
	byAlias map[string][]int
}

// NewCatalog indexes metrics by code and by folded alias.
func NewCatalog(version string, metrics []Metric) *Catalog {
	c := &Catalog{Version: version, metrics: metrics, byCode: map[string]int{}, byAlias: map[string][]int{}}
	for i, m := range metrics {
		c.byCode[m.Code] = i
		seen := map[string]bool{}
		for _, a := range append([]string{m.Code, m.Name, m.NameVI}, m.Aliases...) {
			key := analyteKey(a)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			c.byAlias[key] = append(c.byAlias[key], i)
		}
	}
	return c
}

// Metrics returns every entry in catalog order.
func (c *Catalog) Metrics() []Metric { return c.metrics }

// Metric returns an entry by code.
func (c *Catalog) Metric(code string) (Metric, bool) {
	i, ok := c.byCode[code]
	if !ok {
		return Metric{}, false
	}
	return c.metrics[i], true
}

// analyteKey folds an analyte name for matching. `%` and `#` mark the
// percentage and absolute count of a white cell type, so they become words;
// `Tỷ lệ %` already says percentage, so its `%` is dropped.
func analyteKey(s string) string {
	s = strings.NewReplacer("%", " pct ", "#", " abs ").Replace(s)
	key := FoldKey(s)
	if rest, ok := strings.CutPrefix(key, "ty le pct "); ok {
		return "ty le " + rest
	}
	return key
}

// urineWords mark an analyte name as a urine test.
var urineWords = [][]string{{"nieu"}, {"nuoc", "tieu"}, {"urine"}, {"urinalysis"}, {"tong", "phan", "tich", "nuoc", "tieu"}}

func isUrine(key string) bool {
	toks := strings.Fields(key)
	for _, w := range urineWords {
	outer:
		for i := 0; i+len(w) <= len(toks); i++ {
			for k := range w {
				if toks[i+k] != w[k] {
					continue outer
				}
			}
			return true
		}
	}
	return false
}

// splitParens returns the name outside parentheses and each part inside:
// `ALT (GPT)` gives `ALT` and [`GPT`].
func splitParens(s string) (string, []string) {
	var outside strings.Builder
	var inside []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(', '[':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
				if depth == 0 {
					inside = append(inside, s[start:i])
				}
			}
		default:
			if depth == 0 {
				outside.WriteRune(r)
			}
		}
	}
	return outside.String(), inside
}

// Match finds the catalog entry for an analyte name as printed, using the
// unit and value to choose between entries that share a name (HW-READ-5).
// It tries the whole name, the name outside parentheses, each part inside
// them, and finally the longest known name the printed one starts with.
// Context words such as `niệu` pick urine tests, and a text result such as
// `Âm tính` picks a test read as text. An unknown or ambiguous name gives
// false.
func (c *Catalog) Match(analyte, unit, value string) (Metric, bool) {
	full := analyteKey(analyte)
	if full == "" {
		return Metric{}, false
	}
	urine := isUrine(full)
	outside, inside := splitParens(analyte)
	stripped := analyteKey(stripUrineWords(outside))
	levels := [][]string{{full}, {stripped}}
	var in []string
	for _, part := range inside {
		in = append(in, analyteKey(part))
	}
	levels = append(levels, in)
	text := strings.TrimSpace(value) != "" && ParseNumber(value) == nil
	for _, keys := range levels {
		if m, ok := c.pick(c.exact(keys), unit, urine, text); ok {
			return m, true
		}
	}
	return c.pick(c.prefix(stripped), unit, urine, text)
}

func stripUrineWords(s string) string {
	key := FoldKey(s)
	if !isUrine(key) {
		return s
	}
	toks := strings.Fields(key)
	var keep []string
	for i := 0; i < len(toks); i++ {
		switch {
		case toks[i] == "nieu" || toks[i] == "urine":
			continue
		case toks[i] == "nuoc" && i+1 < len(toks) && toks[i+1] == "tieu":
			i++
			continue
		}
		keep = append(keep, toks[i])
	}
	return strings.Join(keep, " ")
}

func (c *Catalog) exact(keys []string) []int {
	var out []int
	for _, k := range keys {
		if k != "" {
			out = append(out, c.byAlias[k]...)
		}
	}
	return out
}

// prefix returns the entries whose longest alias starts key at a word
// boundary. A WholeName entry never matches this way, and still blocks
// shorter aliases: `Số lượng bạch cầu đoạn` must not fall back to `bạch cầu`.
func (c *Catalog) prefix(key string) []int {
	if key == "" {
		return nil
	}
	best := 0
	var out []int
	for alias, idx := range c.byAlias {
		if len(alias) < 3 || len(alias) < best || !strings.HasPrefix(key, alias) || (len(key) > len(alias) && key[len(alias)] != ' ') {
			continue
		}
		if len(alias) > best {
			best, out = len(alias), nil
		}
		out = append(out, idx...)
	}
	for _, i := range out {
		if c.metrics[i].WholeName && len(key) > best {
			return nil
		}
	}
	return out
}

// pick chooses one entry among candidates: by urine context, then by the
// kind of result, then by unit.
func (c *Catalog) pick(cands []int, unit string, urine, text bool) (Metric, bool) {
	cands = dedupe(cands)
	if len(cands) == 0 {
		return Metric{}, false
	}
	if text {
		// A text result fits only a test read as text: `Bạch cầu` with
		// `Âm tính` is the urine strip, not the white cell count.
		var fit []int
		for _, i := range cands {
			if c.metrics[i].Unit == "" {
				fit = append(fit, i)
			}
		}
		if len(fit) > 0 {
			cands = fit
		}
	}
	var urineC, bloodC []int
	for _, i := range cands {
		if c.metrics[i].Urine {
			urineC = append(urineC, i)
		} else {
			bloodC = append(bloodC, i)
		}
	}
	switch {
	case urine && len(urineC) > 0:
		cands = urineC
	case !urine && len(bloodC) > 0:
		cands = bloodC
	case !urine && !text && strings.TrimSpace(unit) != "":
		// Only urine tests share this name and nothing says urine: a
		// printed unit must fit them (`Protein` in g/L is a blood test).
		var fit []int
		for _, i := range cands {
			if _, ok := c.metrics[i].ToCanonical(1, unit); ok {
				fit = append(fit, i)
			}
		}
		cands = fit
	}
	if len(cands) > 1 && strings.TrimSpace(unit) != "" {
		var fit []int
		for _, i := range cands {
			if _, ok := c.metrics[i].ToCanonical(1, unit); ok {
				fit = append(fit, i)
			}
		}
		if len(fit) > 0 {
			cands = fit
		}
	}
	if len(cands) != 1 {
		return Metric{}, false
	}
	return c.metrics[cands[0]], true
}

func dedupe(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, i := range in {
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	return out
}

// ToCanonical converts a value in unit to the metric's canonical unit.
func (m Metric) ToCanonical(value float64, unit string) (float64, bool) {
	u := NormalizeUnit(unit)
	if u == m.Unit {
		return value, true
	}
	for _, conv := range m.Conversions {
		if conv.From == u {
			v := value*conv.Factor + conv.Offset
			return math.Round(v*1e6) / 1e6, true
		}
	}
	return 0, false
}

// unitSpellings maps folded spellings to the catalog's unit names. Count
// units depend on case (G/L is 10^9 cells per litre, g/L grams), so
// NormalizeUnit handles those first.
var unitSpellings = map[string]string{
	"mmol/l": "mmol/L", "umol/l": "µmol/L", "nmol/l": "nmol/L", "pmol/l": "pmol/L",
	"mg/dl": "mg/dL", "g/dl": "g/dL", "g/l": "g/L", "mg/l": "mg/L", "ug/dl": "µg/dL", "ug/l": "µg/L",
	"ng/ml": "ng/mL", "pg/ml": "pg/mL", "ng/dl": "ng/dL", "ug/ml": "µg/mL",
	"miu/l": "mIU/L", "uiu/ml": "µIU/mL", "miu/ml": "mIU/mL", "iu/ml": "IU/mL",
	"u/l": "U/L", "iu/l": "U/L", "ui/l": "U/L",
	"meq/l": "mEq/L", "%": "%", "mmol/mol": "mmol/mol", "l/l": "L/L",
	"fl": "fL", "pg": "pg", "mm/h": "mm/h", "mm/gio": "mm/h", "mm/1h": "mm/h", "mm/hr": "mm/h",
	"s": "s", "sec": "s", "giay": "s",
	"ml/min/1.73m2": "mL/min/1.73m²", "ml/ph/1.73m2": "mL/min/1.73m²", "ml/phut/1.73m2": "mL/min/1.73m²",
	"ml/min": "mL/min/1.73m²",
	"mg/g":   "mg/g", "mg/mmol": "mg/mmol",
	"kg": "kg", "lb": "lb", "lbs": "lb", "cm": "cm", "m": "m", "in": "in",
	"kg/m2": "kg/m²", "mmhg": "mmHg", "/min": "/min", "bpm": "/min", "lan/phut": "/min", "l/p": "/min", "l/ph": "/min",
	"°c": "°C", "oc": "°C", "c": "°C", "°f": "°F", "of": "°F",
	"10^9/l": "G/L", "10^3/ul": "G/L", "k/ul": "G/L", "x10^9/l": "G/L", "giga/l": "G/L",
	"10^12/l": "T/L", "10^6/ul": "T/L", "m/ul": "T/L", "x10^12/l": "T/L", "tera/l": "T/L",
	"ratio": "", "-": "",
}

// NormalizeUnit returns the catalog spelling of a printed unit: `umol/L`
// and `μmol/l` become `µmol/L`, `10^9/L` becomes `G/L`. An unknown unit is
// returned trimmed.
func NormalizeUnit(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if len(u) >= 3 && strings.EqualFold(u[1:], "/l") {
		switch u[0] {
		case 'G':
			return "G/L"
		case 'T':
			return "T/L"
		}
	}
	key := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return -1
		case r == 'μ' || r == 'µ':
			return 'u'
		case r == '²':
			return '2'
		case r == '×' || r == '*':
			return '^'
		}
		return unicode.ToLower(r)
	}, u)
	key = strings.NewReplacer("x10^", "10^", "10^^", "10^", "10e", "10^", "giây", "giay", "phút", "phut", "giờ", "gio", "lần", "lan").Replace(key)
	if v, ok := unitSpellings[key]; ok {
		return v
	}
	return u
}
