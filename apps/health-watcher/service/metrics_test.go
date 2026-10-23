package service

import (
	"errors"
	"strings"
	"testing"

	"apps/health-watcher/domain"
)

func vitals(date string, values ...string) domain.MeasurementInput {
	in := domain.MeasurementInput{Date: date}
	for i := 0; i+1 < len(values); i += 2 {
		in.Values = append(in.Values, domain.MeasurementValue{MetricCode: values[i], Value: values[i+1]})
	}
	return in
}

func TestGivenALabPaperAndTypedVitalsWhenTheOverviewIsReadThenEachMetricShowsItsLatestValue(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	doc := e.filedLab(sc, person, "lab")
	if _, err := e.clinical.AddObservation(e.ctx, sc, doc.Document.ID, domain.ObservationInput{Analyte: "Glucose", Value: "99",
		Unit: "mg/dL", Reference: "70 - 110", SampleDate: "2025-09-01"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.clinical.AddObservation(e.ctx, sc, doc.Document.ID, domain.ObservationInput{Analyte: "Chỉ số lạ", Value: "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.clinical.AddMeasurements(e.ctx, sc, person.ID, vitals("2025-03-10", "height", "170")); err != nil {
		t.Fatal(err)
	}
	typed, err := e.clinical.AddMeasurements(e.ctx, sc, person.ID, vitals("2025-04-01", "weight", "72,25", "bp_systolic", "128"))
	if err != nil {
		t.Fatal(err)
	}
	if len(typed) != 2 || typed[0].DocumentID != "" || typed[0].PersonID != person.ID || typed[0].Status != domain.StatusApproved ||
		typed[0].Analyte != "Body weight" || typed[0].Unit != "kg" {
		t.Fatalf("typed = %+v", typed)
	}

	ov, err := e.clinical.PersonMetrics(e.ctx, sc, person.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ov.Unlinked != 1 || len(ov.Groups) != 3 || ov.Groups[0].Category != domain.CatGlucose || ov.Groups[1].Category != domain.CatVitals {
		t.Fatalf("overview = %+v", ov)
	}
	// A name the catalog doesn't know is its own metric, never left out (HW-MET-7).
	other := ov.Groups[2]
	if other.Category != domain.CatPrinted || len(other.Items) != 1 || other.Items[0].Metric.Name != "Chỉ số lạ" ||
		other.Items[0].Latest.Printed != "1" || other.Items[0].Latest.Value == nil {
		t.Fatalf("printed group = %+v", other)
	}
	trend, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, other.Items[0].Metric.Code)
	if err != nil || len(trend.Points) != 1 || !trend.Metric.Printed || trend.About != nil {
		t.Fatalf("printed trend = %+v, %v", trend, err)
	}
	g := ov.Groups[0].Items[0]
	if g.Metric.Code != "glucose" || g.Count != 2 || g.Latest.Date != "2025-09-01" || g.Previous.Flag != "H" ||
		g.Latest.Provider != "Bệnh viện Đa khoa Mẫu" || g.Change == nil || *g.Change != -0.904885 {
		t.Fatalf("glucose = %+v", g)
	}
	codes := []string{}
	for _, it := range ov.Groups[1].Items {
		codes = append(codes, it.Metric.Code)
	}
	if len(codes) != 4 || codes[0] != "weight" || codes[1] != "height" || codes[2] != "bmi" || codes[3] != "bp_systolic" {
		t.Fatalf("vitals = %v", codes)
	}

	bmi, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, "bmi")
	if err != nil {
		t.Fatal(err)
	}
	if len(bmi.Points) != 1 || *bmi.Points[0].Value != 25 || bmi.Points[0].Source != domain.SourceComputed {
		t.Fatalf("bmi = %+v", bmi.Points)
	}
	none, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, "hba1c")
	if err != nil || len(none.Points) != 0 {
		t.Fatalf("hba1c = %+v, %v", none, err)
	}
	if _, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown metric = %v", err)
	}
}

func TestGivenTypedVitalsWhenTheirPersonIsDeletedThenItIsRefusedUntilTheyAreGone(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Trẻ Em Mẫu")
	typed, err := e.clinical.AddMeasurements(e.ctx, sc, person.ID, vitals("2025-04-01", "weight", "20"))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.records.DeletePerson(e.ctx, sc, person.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("delete person = %v", err)
	}
	if err := e.clinical.DeleteObservation(e.ctx, sc, typed[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := e.records.DeletePerson(e.ctx, sc, person.ID); err != nil {
		t.Fatal(err)
	}
}

func TestGivenAViewerOrAnotherTenantWhenTheyUseMetricsThenOnlyReadingInTheirOwnTenantWorks(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.clinical.AddMeasurements(e.ctx, viewer, person.ID, vitals("2025-04-01", "weight", "60")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer add = %v", err)
	}
	if _, err := e.clinical.PersonMetrics(e.ctx, viewer, person.ID); err != nil {
		t.Fatalf("viewer read = %v", err)
	}
	tb, err := e.tenants.Create(e.ctx, viewer.UserID, "Tenant B")
	if err != nil {
		t.Fatal(err)
	}
	b := e.scope(viewer.UserID, tb.ID)
	if _, err := e.clinical.PersonMetrics(e.ctx, b, person.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other tenant overview = %v", err)
	}
	if _, err := e.clinical.PersonMetric(e.ctx, b, person.ID, "weight"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other tenant trend = %v", err)
	}
	if _, err := e.clinical.AddMeasurements(e.ctx, b, person.ID, vitals("2025-04-01", "weight", "60")); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other tenant add = %v", err)
	}
}

func TestGivenAValueOnTheSecondFileOfADocumentWhenItsTrendIsReadThenThePointNamesItsPage(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	out := labReading()
	out.Pages = []domain.ReadingPage{{Page: 1, Title: "Phiếu chỉ định"}, {Page: 2, Title: "Sinh hóa máu"}}
	out.Observations[0].Page = 2
	e.reader.out = &out
	d := e.uploadRead(sc, "lab")
	e.addFiles(sc, d.Item.ID, pdf("lab-2"))
	e.drain()
	if _, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14", Type: "lab"}}); err != nil {
		t.Fatal(err)
	}

	trend, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, "glucose")
	if err != nil {
		t.Fatal(err)
	}
	if len(trend.Points) != 1 {
		t.Fatalf("points = %+v", trend.Points)
	}
	p := trend.Points[0]
	if p.PageID == "" || p.PageNumber != 2 || p.PageTitle != "Sinh hóa máu" {
		t.Fatalf("point = %+v", p)
	}
	if trend.About == nil || !strings.HasPrefix(trend.About.URL, "https://medlineplus.gov/") || trend.About.Quote == "" {
		t.Fatalf("about = %+v", trend.About)
	}
}
