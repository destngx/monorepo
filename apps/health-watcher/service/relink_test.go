package service

import (
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Values filed while the catalog linked `Số lượng bạch cầu đoạn (GRAN#)`
// to the white cell count are matched again with the fixed catalog; values
// a person edited or linked by hand stay as they are.
func TestGivenValuesLinkedByAnOlderCatalogWhenTheEngineStartsThenUneditedOnesAreRelinkedOnceAndAudited(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	e.pipeline.Catalog = domain.NewCatalog("old", []domain.Metric{{Code: "wbc", Name: "White blood cells", Unit: "G/L",
		Aliases: []string{"số lượng bạch cầu đoạn", "bạch cầu mid", "bạch cầu lympho"}}})
	out := labReading()
	out.Observations = []domain.ReadingObservation{
		{Page: 1, Analyte: "Số lượng bạch cầu đoạn (GRAN#)", Value: "4.4", Unit: "G/L", Confidence: 97},
		{Page: 1, Analyte: "Bạch cầu Mid", Value: "0.7", Unit: "G/L", Confidence: 97},
		{Page: 1, Analyte: "Bạch cầu Lympho", Value: "2.5", Unit: "G/L", Confidence: 97},
	}
	e.reader.out = &out
	doc := e.filedLab(sc, person, "lab")
	gran, mid, lympho := doc.Observations[0], doc.Observations[1], doc.Observations[2]
	for _, o := range doc.Observations {
		if o.MetricCode != "wbc" {
			t.Fatalf("%s linked to %q by the old catalog", o.Analyte, o.MetricCode)
		}
	}
	edited := ObservationPatch{ObservationInput: domain.ObservationInput{Analyte: mid.Analyte, MetricCode: "wbc", Value: "0.8", Unit: "G/L"}}
	if _, err := e.clinical.UpdateObservation(e.ctx, sc, mid.ID, edited); err != nil {
		t.Fatal(err)
	}
	if err := e.store.InTx(e.ctx, func(r port.Repos) error {
		pr, err := r.ProviderByKey(e.ctx, sc.TenantID, domain.FoldKey("Bệnh viện Đa khoa Mẫu"))
		if err != nil {
			return err
		}
		return r.SetAnalyteLink(e.ctx, sc.TenantID, pr.ID, domain.FoldKey(lympho.Analyte), "wbc")
	}); err != nil {
		t.Fatal(err)
	}

	for want := range []int{1, 0} {
		n, err := e.clinical.RelinkCatalog(e.ctx)
		if err != nil || n != 1-want {
			t.Fatalf("run %d: relinked %d, %v", want+1, n, err)
		}
	}

	got, err := e.clinical.Document(e.ctx, sc, doc.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]string{}
	for _, o := range got.Observations {
		codes[o.ID] = o.MetricCode
	}
	if codes[gran.ID] != "" || codes[mid.ID] != "wbc" || codes[lympho.ID] != "wbc" {
		t.Fatalf("codes = gran %q, mid %q, lympho %q", codes[gran.ID], codes[mid.ID], codes[lympho.ID])
	}
	hist, err := e.clinical.History(e.ctx, sc, domain.AuditObservation, gran.ID)
	if err != nil || len(hist) != 1 || hist[0].UserID != "" {
		t.Fatalf("history = %+v, %v", hist, err)
	}
}
