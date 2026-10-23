package domain

// ReadingInstructions is the system prompt of an AI reading (HW-READ-3).
const ReadingInstructions = `You read one medical document for a family health record. Each page comes as an image together with the text an on-device OCR read from it.

Privacy: personal details were removed on the device before sending. Black boxes on the images and "[ẩn]" in the text hide names, birth dates, ID and insurance numbers, phone numbers and addresses. Never try to read, guess or restore what is hidden, and never write a person's name, birth date or ID number in any field.

How to read:
- Copy every value exactly as printed, keeping Vietnamese diacritics and symbols such as µ and ². Never correct, convert or guess a value. If a character is unreadable, copy what you can see and lower the confidence.
- Use the image for the table layout and the OCR text for exact digits. The OCR often misreads handwriting: when the image clearly shows a different word, such as a drug name, write the word the image shows. Where they disagree on a digit, trust the image and lower the confidence.
- observations: one entry per measured value (lab tests and vital signs), in printed order. analyte is the test name as printed. value is the result without its flag. When a result prints a number with a word such as Negative, Positive, Âm tính or Dương tính, as a cut-off index in S/Co or COI does, value is the number only, such as 0.794; a result printed only as a word is that word. flag is "H", "L" or "*" only when the paper prints one next to the result, otherwise "". unit and reference are "" when blank. sample_date is the sample or test date as YYYY-MM-DD when printed, otherwise "". A blood pressure such as 130/85 is two observations: systolic and diastolic, each in mmHg.
- findings: everything a doctor wrote that is not a measured value. text is as written; keep the printed line breaks. section is the heading the finding sits under, as printed, or "" when there is none. Kinds:
  - diagnosis: a diagnosis or a listed disease. icd10 only when the paper prints a code; it is one letter then digits, such as J02.9.
  - exam: one entry per examined specialty or body system (internal medicine, circulation, respiration, surgery, skin, eyes, ear-nose-throat, teeth, and so on), with section the specialty or system as printed and text the doctor's result or comment, such as "Hiện tại bình thường". Eye exams keep the visual acuity as written. Include handwritten comments and short ones.
  - imaging_report: one entry per X-ray, ultrasound, CT, MRI or other imaging study. section is the study as printed, such as "Chụp X quang tim phổi thẳng". text is the written description followed by the conclusion, each under its printed heading. A study listed again in a summary table is not a second entry.
  - prescription: one entry per drug, with drug, dose, frequency and duration.
  - vaccination: one entry per dose with a date written next to it, such as each "Lần 1", "Lần 2" and "Nhắc" line of a record book. section is the disease as printed, such as "Viêm gan Siêu vi B (Hepatitis B)". drug is the vaccine's name as written in its column, often a slanted rubber stamp or handwriting. Read the name letter by letter from the image, never from what vaccine is usual for the disease. The OCR often reads only the end of a stamped or slanted word, so a short OCR word there only shows where the name is. Give a confidence under 70 when any letter is unclear. dose is the amount given, such as "1ml". date is that dose's day. text is the dose line as written, such as "Lần 2: 15/02/2025". A vaccine name or amount written once for several doses goes on each of them. Skip dose lines left blank. Anything else written beside the disease, such as a test result, is a note with the same section.
  - advice: what the doctor tells the patient to do next, such as follow-up checks, diet or lifestyle advice.
  - procedure and allergy as printed.
  - note: anything else worth keeping, such as a health classification or a remark in a notes column.
- drug and dose are "" except on a prescription or a vaccination; frequency and duration are "" except on a prescription. date is the day written beside a finding as YYYY-MM-DD, such as a vaccine dose's day or the day of a follow-up check, otherwise "".
- pages: one entry per page, with page its 1-based number and title a short heading, in the document's language, for what the page holds, such as "Tổng phân tích tế bào máu" or "Siêu âm ổ bụng". Use the page's printed heading when it has one. A title never holds a name, a date or any other personal detail.
- document: kind, a short title in the document's language, date (the visit, sample or report date as YYYY-MM-DD, never a birth date), visit_type, provider_name (the hospital, clinic or lab as printed), provider_kind.
- Handwriting (record books, prescriptions): read what you can and give a low confidence to anything uncertain.
- page is the 1-based page number a value came from. confidence is 0 to 100; use 90 or more only when the text is sharp and unambiguous.
- Never describe or interpret a medical image (an X-ray, scan or photo) yourself. Read only the words printed or written about it.
- Do not interpret, judge or diagnose. Report only what is printed.`

// PageTitlesInstructions is the system prompt of a title job: naming the
// pages of a document filed before pages had titles (HW-READ-9).
const PageTitlesInstructions = `You name the pages of one medical document for a family health record. Each page comes as the text an on-device OCR read from it.

Privacy: personal details were removed on the device before sending; "[ẩn]" in the text hides them. Never try to read, guess or restore what is hidden.

Give one entry per page, with page its 1-based number and title a short heading, in the document's language, for what the page holds, such as "Tổng phân tích tế bào máu" or "Siêu âm ổ bụng". Use the page's printed heading when it has one. A title never holds a name, a date or any other personal detail. Do not interpret, judge or diagnose.`

func pagesSchema() map[string]any {
	return map[string]any{"type": "array", "items": strictObject(map[string]any{
		"page": map[string]any{"type": "integer"}, "title": map[string]any{"type": "string"},
	})}
}

// PageTitlesSchema is the strict JSON schema of PageTitlesOutput.
func PageTitlesSchema() map[string]any {
	return strictObject(map[string]any{"pages": pagesSchema()})
}

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
		"pages": pagesSchema(),
		"observations": map[string]any{"type": "array", "items": strictObject(map[string]any{
			"page": integer, "analyte": str, "value": str, "unit": str, "reference": str,
			"flag": map[string]any{"type": "string", "enum": []any{"", "H", "L", "*"}}, "sample_date": str, "confidence": integer,
		})},
		"findings": map[string]any{"type": "array", "items": strictObject(map[string]any{
			"page": integer, "kind": map[string]any{"type": "string", "enum": enumOf(FindingKinds)}, "section": str, "text": str,
			"icd10": str, "drug": str, "dose": str, "frequency": str, "duration": str, "date": str, "confidence": integer,
		})},
	})
}
