package service

import (
	"errors"
	"testing"

	"apps/health-watcher/domain"
)

func TestGivenAReadingWhenFiledThenEachRecordKeepsTheModelAndWhoApprovedIt(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "lab")
	for _, o := range d.Observations {
		if o.Source != domain.SourceReading || o.SourceDetail != "test-model" || o.Version != 1 || o.ApprovedBy != "" || o.ApprovedAt != nil {
			t.Fatalf("draft observation provenance = %+v", o.Provenance)
		}
	}

	filed, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: e.person(sc, "Nguyễn Văn Mẫu").ID, Date: "2025-03-14"}})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Observations) == 0 || len(doc.Findings) == 0 {
		t.Fatalf("filed document = %+v", doc)
	}
	for _, o := range doc.Observations {
		if o.Source != domain.SourceReading || o.SourceDetail != "test-model" || o.ApprovedBy != sc.UserID || o.ApprovedAt == nil {
			t.Fatalf("filed observation provenance = %+v", o.Provenance)
		}
	}
	for _, f := range doc.Findings {
		if f.Source != domain.SourceReading || f.SourceDetail != "test-model" || f.ApprovedBy != sc.UserID || f.ApprovedAt == nil {
			t.Fatalf("filed finding provenance = %+v", f.Provenance)
		}
	}
}

func TestGivenAnApprovedRecordWhenEditedTwiceThenItsVersionRisesAndHistoryKeepsEachOldVersion(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")
	obs := doc.Observations[0]

	if _, err := e.clinical.UpdateObservation(e.ctx, sc, obs.ID, glucose("5.9")); err != nil {
		t.Fatal(err)
	}
	edited, err := e.clinical.LinkMetric(e.ctx, sc, obs.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	if edited.Version != 3 || edited.Source != domain.SourceReading || edited.ApprovedBy != sc.UserID || edited.ApprovedAt == nil {
		t.Fatalf("edited provenance = %+v", edited.Provenance)
	}
	hist, err := e.clinical.History(e.ctx, sc, domain.AuditObservation, obs.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].Version != 1 || hist[1].Version != 2 {
		t.Fatalf("history = %+v", hist)
	}
}

func TestGivenADraftWhenEditedThenItsVersionStaysAtOne(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "lab")

	edited, err := e.clinical.UpdateObservation(e.ctx, sc, d.Observations[0].ID, glucose("6,2"))
	if err != nil {
		t.Fatal(err)
	}

	if edited.Version != 1 || edited.SourceDetail != "test-model" {
		t.Fatalf("edited draft provenance = %+v", edited.Provenance)
	}
}

func TestGivenARecordTypedByHandWhenSavedThenItIsManualAndApprovedByItsAuthor(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")

	o, err := e.clinical.AddObservation(e.ctx, sc, doc.Document.ID, domain.ObservationInput{Analyte: "HbA1c", Value: "5.6", Unit: "%"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := e.clinical.AddFinding(e.ctx, sc, doc.Document.ID, domain.FindingInput{Kind: "diagnosis", Text: "TD: Tiền đái tháo đường"})
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []domain.Provenance{o.Provenance, f.Provenance} {
		if p.Source != domain.SourceManual || p.SourceDetail != "" || p.Version != 1 || p.ApprovedBy != sc.UserID || p.ApprovedAt == nil {
			t.Fatalf("manual provenance = %+v", p)
		}
	}
}

func TestGivenAnInpatientStayWhenItsEndDateIsSetThenItIsKeptUntilChanged(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	p := e.person(sc, "Nguyễn Văn Mẫu")
	end := "2025-03-18"

	v, err := e.records.CreateVisit(e.ctx, sc, domain.VisitInput{PersonID: p.ID, Date: "2025-03-14", EndDate: &end, Type: "inpatient"})
	if err != nil {
		t.Fatal(err)
	}
	if v.EndDate != end {
		t.Fatalf("created end date = %q", v.EndDate)
	}

	v, err = e.records.UpdateVisit(e.ctx, sc, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-03-15", Type: "inpatient"})
	if err != nil {
		t.Fatal(err)
	}
	if v.EndDate != end || v.Date != "2025-03-15" {
		t.Fatalf("an update without an end date changed it: %+v", v)
	}

	if _, err := e.records.UpdateVisit(e.ctx, sc, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-03-20", Type: "inpatient"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("visit date after its end date = %v", err)
	}
	before := "2025-03-10"
	if _, err := e.records.UpdateVisit(e.ctx, sc, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-03-15", EndDate: &before}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("end date before the visit = %v", err)
	}

	cleared := ""
	v, err = e.records.UpdateVisit(e.ctx, sc, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-03-15", EndDate: &cleared})
	if err != nil {
		t.Fatal(err)
	}
	if v.EndDate != "" {
		t.Fatalf("cleared end date = %q", v.EndDate)
	}
}

func TestGivenADocumentWhenItsPrintedDateIsSetThenAnEditWithoutADateKeepsIt(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")
	date := "2025-03-16"

	updated, err := e.clinical.UpdateDocument(e.ctx, sc, doc.Document.ID, DocumentInput{Kind: "lab_result", Date: &date})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Document.Date != date {
		t.Fatalf("date = %q", updated.Document.Date)
	}

	updated, err = e.clinical.UpdateDocument(e.ctx, sc, doc.Document.ID, DocumentInput{Kind: "lab_result", Title: "Xét nghiệm máu"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Document.Date != date || updated.Document.Title != "Xét nghiệm máu" {
		t.Fatalf("an edit without a date changed it: %+v", updated.Document)
	}

	future := "2999-01-01"
	if _, err := e.clinical.UpdateDocument(e.ctx, sc, doc.Document.ID, DocumentInput{Kind: "lab_result", Date: &future}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("future date = %v", err)
	}
}

func TestGivenRecordsApprovedBeforeD9WhenSomeoneConfirmsThemThenTheyNameTheApproverAndNeedNothingForExport(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")
	for _, table := range []string{"observations", "findings"} {
		if _, err := e.store.DB().Exec(`UPDATE `+table+` SET approved_by = NULL, approved_at = '' WHERE document_id = ?`, doc.Document.ID); err != nil {
			t.Fatal(err)
		}
	}
	before, err := e.clinical.Document(e.ctx, sc, doc.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := before.Observations[0].ExportGaps; len(got) != 1 || got[0] != domain.GapApprover {
		t.Fatalf("gaps before = %v", got)
	}
	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.clinical.ConfirmApprovals(e.ctx, viewer, doc.Document.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer confirm = %v", err)
	}

	after, err := e.clinical.ConfirmApprovals(e.ctx, sc, doc.Document.ID)
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range after.Observations {
		if o.ApprovedBy != sc.UserID || o.ApprovedAt == nil || o.Version != 1 || len(o.ExportGaps) != 0 {
			t.Fatalf("confirmed observation = %+v, gaps %v", o.Provenance, o.ExportGaps)
		}
	}
	for _, f := range after.Findings {
		if f.ApprovedBy != sc.UserID || len(f.ExportGaps) != 0 {
			t.Fatalf("confirmed finding = %+v", f.Provenance)
		}
	}
}

func TestGivenAnInpatientStayWhenItHasNoLastDayThenItNeedsOneForExport(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	p := e.person(sc, "Nguyễn Văn Mẫu")

	v, err := e.records.CreateVisit(e.ctx, sc, domain.VisitInput{PersonID: p.ID, Date: "2025-03-14", Type: "inpatient"})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.ExportGaps) != 1 || v.ExportGaps[0] != domain.GapEndDate {
		t.Fatalf("gaps = %v", v.ExportGaps)
	}

	end := "2025-03-18"
	v, err = e.records.UpdateVisit(e.ctx, sc, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-03-14", Type: "inpatient", EndDate: &end})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.ExportGaps) != 0 {
		t.Fatalf("gaps after the end date = %v", v.ExportGaps)
	}
}

func TestGivenVitalsTypedWithoutAPaperWhenSavedThenTheyAreManualAndApprovedByTheirAuthor(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")

	typed, err := e.clinical.AddMeasurements(e.ctx, sc, person.ID, vitals("2025-04-01", "weight", "72"))
	if err != nil {
		t.Fatal(err)
	}

	p := typed[0].Provenance
	if p.Source != domain.SourceManual || p.Version != 1 || p.ApprovedBy != sc.UserID || p.ApprovedAt == nil || len(typed[0].ExportGaps) != 0 {
		t.Fatalf("typed vital provenance = %+v, gaps %v", p, typed[0].ExportGaps)
	}
}
