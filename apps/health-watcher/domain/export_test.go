package domain

import (
	"slices"
	"testing"
	"time"
)

func TestGivenAnInpatientStayWithoutItsLastDayWhenCheckedForExportThenTheEndDateIsMissing(t *testing.T) {
	cases := []struct {
		visit Visit
		want  []ExportGap
	}{
		{Visit{Type: "inpatient"}, []ExportGap{GapEndDate}},
		{Visit{Type: "inpatient", EndDate: "2025-03-18"}, nil},
		{Visit{Type: "checkup"}, nil},
	}
	for _, tc := range cases {
		if got := VisitExportGaps(tc.visit); !slices.Equal(got, tc.want) {
			t.Errorf("VisitExportGaps(%+v) = %v; want %v", tc.visit, got, tc.want)
		}
	}
}

func TestGivenAnApprovedRecordWithoutAnApproverWhenCheckedForExportThenTheApproverIsMissing(t *testing.T) {
	now := time.Now()
	cases := []struct {
		status RecordStatus
		p      Provenance
		want   []ExportGap
	}{
		{StatusApproved, Provenance{}, []ExportGap{GapApprover}},
		{StatusApproved, Provenance{ApprovedBy: "u1", ApprovedAt: &now}, nil},
		{StatusDraft, Provenance{}, nil},
		{StatusRejected, Provenance{}, nil},
	}
	for _, tc := range cases {
		if got := RecordExportGaps(tc.status, tc.p); !slices.Equal(got, tc.want) {
			t.Errorf("RecordExportGaps(%s, %+v) = %v; want %v", tc.status, tc.p, got, tc.want)
		}
	}
}
