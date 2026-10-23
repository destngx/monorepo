package domain

// SuggestInstructions is the system prompt of a metric suggestion
// (HW-MET-8). The request holds printed test names and units only: no
// values, people or dates.
const SuggestInstructions = `You help keep a family health record's metric catalog complete. You get test names exactly as Vietnamese and English lab papers printed them, each with its printed unit, and the catalog: each metric's code, English and Vietnamese names and canonical unit. No results, people or dates are sent.

For each printed name, give one answer with name copied exactly as given:
- action "link" when the name is a catalog metric under another name, abbreviation or language, such as "GRA#" for granulocytes. metric_code is that metric's code. Only link when it is the same test on the same specimen and the units fit: a percentage is never a count, and a urine test is never a blood test.
- action "new" when it is a measured test the catalog lacks. new_name is the usual English name, new_name_vi the usual Vietnamese name, unit the canonical unit as usually printed (such as G/L, mmol/L, %, or "" for a result read as text), category one of the given categories, and loinc the LOINC code of that test on that specimen when you are sure of it, else "".
- action "none" when it is not a measured test, such as a heading, a note or a signature line.
reason says in one short English sentence why. Do not interpret, judge or diagnose.`

// SuggestSchema is the strict JSON schema of SuggestOutput.
func SuggestSchema() map[string]any {
	str := map[string]any{"type": "string"}
	return strictObject(map[string]any{
		"suggestions": map[string]any{"type": "array", "items": strictObject(map[string]any{
			"name": str, "action": map[string]any{"type": "string", "enum": []any{"link", "new", "none"}}, "metric_code": str,
			"new_name": str, "new_name_vi": str, "loinc": str, "unit": str,
			"category": map[string]any{"type": "string", "enum": append(enumOf(MetricCategories), "")}, "reason": str,
		})},
	})
}
