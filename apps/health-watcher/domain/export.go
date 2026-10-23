package domain

// ExportGap names something a record lacks that a complete FHIR export
// needs (D9). The field stays optional while a person works; the export
// (HW-INT-2) lists every gap and waits until they are filled.
type ExportGap string

const (
	// GapEndDate: an inpatient stay without its last day
	// (Encounter.period.end).
	GapEndDate ExportGap = "end_date"
	// GapApprover: an approved record with nobody recorded as approving
	// it, which is every record approved before D9 (Provenance verifier).
	GapApprover ExportGap = "approver"
)

// VisitExportGaps lists what a visit lacks for export. A document's date is
// never a gap: the visit date stands in for it.
func VisitExportGaps(v Visit) []ExportGap {
	if v.Type == "inpatient" && v.EndDate == "" {
		return []ExportGap{GapEndDate}
	}
	return nil
}

// RecordExportGaps lists what an observation or finding lacks for export.
// Drafts and rejected records are never exported, so they have no gaps.
func RecordExportGaps(status RecordStatus, p Provenance) []ExportGap {
	if status == StatusApproved && p.ApprovedBy == "" {
		return []ExportGap{GapApprover}
	}
	return nil
}
