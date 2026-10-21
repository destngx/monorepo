package domain

// ReadingInstructions is the system prompt of an AI reading (HW-READ-3).
const ReadingInstructions = `You read one medical document for a family health record. Each page comes as an image together with the text an on-device OCR read from it.

Privacy: personal details were removed on the device before sending. Black boxes on the images and "[ẩn]" in the text hide names, birth dates, ID and insurance numbers, phone numbers and addresses. Never try to read, guess or restore what is hidden, and never write a person's name, birth date or ID number in any field.

How to read:
- Copy every value exactly as printed, keeping Vietnamese diacritics and symbols such as µ and ². Never correct, convert or guess a value. If a character is unreadable, copy what you can see and lower the confidence.
- Use the image for the table layout and the OCR text for exact digits. Where they disagree, trust the image and lower the confidence.
- observations: one entry per measured value (lab tests and vital signs), in printed order. analyte is the test name as printed. value is the result without its flag. flag is "H", "L" or "*" only when the paper prints one next to the result, otherwise "". unit and reference are "" when blank. sample_date is the sample or test date as YYYY-MM-DD when printed, otherwise "".
- findings: diagnoses (icd10 only when the paper prints a code), prescriptions (one entry per drug with drug, dose, frequency and duration), procedures, imaging reports (the conclusion as written), allergies, vaccinations, and other notes worth keeping. text is as written.
- document: kind, a short title in the document's language, date (the visit, sample or report date as YYYY-MM-DD, never a birth date), visit_type, provider_name (the hospital, clinic or lab as printed), provider_kind.
- Handwriting (record books, prescriptions): read what you can and give a low confidence to anything uncertain.
- page is the 1-based page number a value came from. confidence is 0 to 100; use 90 or more only when the text is sharp and unambiguous.
- Do not interpret, judge or diagnose. Report only what is printed.`

func enumOf[T ~string](values []T) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func strictObject(props map[string]any) map[string]any {
	required := make([]any, 0, len(props))
	for k := range props {
		required = append(required, k)
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": required}
}

// ReadingSchema is the strict JSON schema of ReadingOutput: every field is
// required, and empty strings stand for blanks.
func ReadingSchema() map[string]any {
	str := map[string]any{"type": "string"}
	integer := map[string]any{"type": "integer"}
	providerKinds := []ProviderKind{ProviderHospital, ProviderClinic, ProviderDental, ProviderLab, ProviderPharmacy, ProviderOther}
	return strictObject(map[string]any{
		"document": strictObject(map[string]any{
			"kind": map[string]any{"type": "string", "enum": enumOf(DocumentKinds)}, "title": str, "date": str,
			"visit_type": map[string]any{"type": "string", "enum": enumOf(VisitTypes)}, "provider_name": str,
			"provider_kind": map[string]any{"type": "string", "enum": enumOf(providerKinds)}, "confidence": integer,
		}),
		"observations": map[string]any{"type": "array", "items": strictObject(map[string]any{
			"page": integer, "analyte": str, "value": str, "unit": str, "reference": str,
			"flag": map[string]any{"type": "string", "enum": []any{"", "H", "L", "*"}}, "sample_date": str, "confidence": integer,
		})},
		"findings": map[string]any{"type": "array", "items": strictObject(map[string]any{
			"page": integer, "kind": map[string]any{"type": "string", "enum": enumOf(FindingKinds)}, "text": str,
			"icd10": str, "drug": str, "dose": str, "frequency": str, "duration": str, "confidence": integer,
		})},
	})
}
