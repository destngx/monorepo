package service

import (
	"errors"
	"strings"
	"testing"

	"apps/health-watcher/domain"
)

// filedLab reads a lab report for the test person and files it.
func (e *env) filedLab(sc domain.Scope, person domain.Person, label string) DocumentDetail {
	e.t.Helper()
	d := e.uploadRead(sc, label)
	filed, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14", Type: "lab", ProviderName: "Bệnh viện Đa khoa Mẫu"}})
	if err != nil {
		e.t.Fatalf("file: %v", err)
	}
	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		e.t.Fatalf("document: %v", err)
	}
	return doc
}

func glucose(value string) ObservationPatch {
	return ObservationPatch{ObservationInput: domain.ObservationInput{Analyte: "Glucose (đói)", MetricCode: "glucose", Value: value,
		Unit: "mmol/L", Reference: "3.9 - 6.1"}}
}

func TestGivenADraftWhenEditedOrRejectedThenNothingIsAuditedAndRejectedDraftsAreNotFiled(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")
	obs, find := d.Observations[0], d.Findings[0]

	edited, err := e.clinical.UpdateObservation(e.ctx, sc, obs.ID, glucose("6,2"))
	if err != nil {
		t.Fatal(err)
	}
	if edited.Value != "6,2" || *edited.ValueNum != 6.2 || edited.Confidence != 100 || edited.Status != domain.StatusDraft || edited.PageID != obs.PageID {
		t.Fatalf("edited = %+v", edited)
	}
	if _, err := e.clinical.UpdateFinding(e.ctx, sc, find.ID, FindingPatch{FindingInput: domain.FindingInput{Kind: "diagnosis",
		Text: find.Text}, Status: domain.StatusRejected}); err != nil {
		t.Fatal(err)
	}
	if hist, _ := e.clinical.History(e.ctx, sc, domain.AuditObservation, obs.ID); len(hist) != 0 {
		t.Fatalf("history = %+v", hist)
	}

	filed, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14"}})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Observations) != 1 || doc.Observations[0].Value != "6,2" || len(doc.Findings) != 0 {
		t.Fatalf("document = %+v", doc)
	}
}

func TestGivenAnApprovedObservationWhenEditedThenTheOldValueIsInItsHistory(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")
	obs := doc.Observations[0]

	if _, err := e.clinical.UpdateObservation(e.ctx, sc, obs.ID, ObservationPatch{ObservationInput: glucose("5.9").ObservationInput,
		Status: domain.StatusDraft}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("back to draft = %v", err)
	}
	edited, err := e.clinical.UpdateObservation(e.ctx, sc, obs.ID, glucose("5.9"))
	if err != nil {
		t.Fatal(err)
	}
	if edited.Status != domain.StatusApproved || edited.Value != "5.9" || edited.SampleDate != "2025-03-14" {
		t.Fatalf("edited = %+v", edited)
	}
	hist, err := e.clinical.History(e.ctx, sc, domain.AuditObservation, obs.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || hist[0].Action != "update" || hist[0].UserID != sc.UserID || !strings.Contains(hist[0].OldJSON, `"value":"6.4"`) {
		t.Fatalf("history = %+v", hist)
	}

	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.clinical.UpdateObservation(e.ctx, viewer, obs.ID, glucose("1")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer edit = %v", err)
	}
}

func TestGivenAFiledDocumentWhenDeletedThenEachApprovedRecordIsAuditedAndLeavesSearch(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")

	if err := e.records.DeleteDocument(e.ctx, sc, doc.Document.ID); err != nil {
		t.Fatal(err)
	}

	for entity, id := range map[domain.AuditEntity]string{domain.AuditObservation: doc.Observations[0].ID, domain.AuditFinding: doc.Findings[0].ID} {
		hist, err := e.clinical.History(e.ctx, sc, entity, id)
		if err != nil || len(hist) != 1 || hist[0].Action != "delete" {
			t.Fatalf("%s history = %+v, %v", entity, hist, err)
		}
	}
	res, err := e.clinical.Search(e.ctx, sc, "kham suc khoe")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 0 {
		t.Fatalf("hits = %+v", res.Hits)
	}
}

func TestGivenAFiledDocumentWhenAddingRecordsByHandThenTheyAreApprovedWithTheVisitDate(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	doc := e.filedLab(sc, person, "lab")

	o, err := e.clinical.AddObservation(e.ctx, sc, doc.Document.ID, domain.ObservationInput{Analyte: "HbA1c", Value: "5.6", Unit: "%",
		PageID: doc.Pages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.StatusApproved || o.MetricCode != "hba1c" || o.SampleDate != "2025-03-14" || o.PersonID != person.ID || o.PageNumber != 1 {
		t.Fatalf("observation = %+v", o)
	}
	f, err := e.clinical.AddFinding(e.ctx, sc, doc.Document.ID, domain.FindingInput{Kind: "prescription", Text: "Metformin",
		Drug: "Metformin", Dose: "500 mg"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != domain.StatusApproved || f.Dose != "500 mg" {
		t.Fatalf("finding = %+v", f)
	}
	if _, err := e.clinical.AddObservation(e.ctx, sc, doc.Document.ID, domain.ObservationInput{Analyte: "X", Value: "1", PageID: "nope"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown page = %v", err)
	}

	updated, err := e.clinical.UpdateDocument(e.ctx, sc, doc.Document.ID, DocumentInput{Kind: "prescription", Title: "Đơn thuốc"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Document.Kind != "prescription" || updated.Document.Title != "Đơn thuốc" || len(updated.Observations) != 2 {
		t.Fatalf("document = %+v", updated)
	}
}

func TestGivenALinkedAnalyteWhenTheSameProviderIsReadAgainThenTheLinkIsUsed(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	out := labReading()
	out.Observations[0].Analyte = "Chỉ số mẫu X1"
	e.reader.out = &out
	doc := e.filedLab(sc, person, "lab")
	obs := doc.Observations[0]
	if obs.MetricCode != "" {
		t.Fatalf("matched = %q", obs.MetricCode)
	}

	linked, err := e.clinical.LinkMetric(e.ctx, sc, obs.ID, "glucose")
	if err != nil {
		t.Fatal(err)
	}
	if linked.MetricCode != "glucose" {
		t.Fatalf("linked = %+v", linked)
	}
	if _, err := e.clinical.LinkMetric(e.ctx, sc, obs.ID, "nope"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown metric = %v", err)
	}

	again := e.uploadRead(sc, "lab-again")
	if len(again.Observations) != 1 || again.Observations[0].MetricCode != "glucose" {
		t.Fatalf("observations = %+v", again.Observations)
	}
}

func TestGivenASearchWhenTheQueryIsEmptyOrTooLongThenItIsRefused(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	for _, q := range []string{"", "  ", strings.Repeat("a", 201)} {
		if _, err := e.clinical.Search(e.ctx, sc, q); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("search %q = %v", q, err)
		}
	}
}

func TestGivenTwoTenantsWhenOneReachesForTheOthersReadingsThenEverythingIsNotFound(t *testing.T) {
	e := newEnv(t)
	a := e.setup()
	outsider := e.join(a, "outsider@example.test", "viewer")
	if err := e.tenants.RemoveMember(e.ctx, outsider, outsider.UserID); err != nil {
		t.Fatal(err)
	}
	tb, err := e.tenants.Create(e.ctx, outsider.UserID, "Tenant B")
	if err != nil {
		t.Fatal(err)
	}
	b := e.scope(outsider.UserID, tb.ID)
	person := e.person(a, "Nguyễn Văn Mẫu")
	doc := e.filedLab(a, person, "lab")
	item := e.uploadRead(a, "lab-2")
	pageID, obsID, findID := doc.Pages[0].ID, doc.Observations[0].ID, doc.Findings[0].ID

	checks := map[string]error{
		"item":   func() error { _, err := e.inbox.Item(e.ctx, b, item.Item.ID); return err }(),
		"send":   e.inbox.SendReading(e.ctx, b, item.Item.ID),
		"skip":   e.inbox.SkipReading(e.ctx, b, item.Item.ID),
		"retry":  e.inbox.RetryReading(e.ctx, b, item.Item.ID),
		"unfile": func() error { _, err := e.inbox.Unfile(e.ctx, b, doc.Document.ID); return err }(),
		"boxes": func() error {
			_, err := e.inbox.SetPageBoxes(e.ctx, b, item.Item.ID, item.Pages[0].ID, nil)
			return err
		}(),
		"off": func() error {
			_, err := e.inbox.SetPageOff(e.ctx, b, item.Item.ID, item.Pages[0].ID, nil)
			return err
		}(),
		"page image": func() error { _, err := e.records.OpenPageImage(e.ctx, b, pageID, false); return err }(),
		"masked":     func() error { _, err := e.records.OpenPageImage(e.ctx, b, item.Pages[0].ID, true); return err }(),
		"document":   func() error { _, err := e.clinical.Document(e.ctx, b, doc.Document.ID); return err }(),
		"update document": func() error {
			_, err := e.clinical.UpdateDocument(e.ctx, b, doc.Document.ID, DocumentInput{Kind: "other"})
			return err
		}(),
		"update observation": func() error { _, err := e.clinical.UpdateObservation(e.ctx, b, obsID, glucose("1")); return err }(),
		"delete observation": e.clinical.DeleteObservation(e.ctx, b, obsID),
		"link metric":        func() error { _, err := e.clinical.LinkMetric(e.ctx, b, obsID, "glucose"); return err }(),
		"update finding": func() error {
			_, err := e.clinical.UpdateFinding(e.ctx, b, findID, FindingPatch{FindingInput: domain.FindingInput{Text: "x"}})
			return err
		}(),
		"delete finding": e.clinical.DeleteFinding(e.ctx, b, findID),
		"add observation": func() error {
			_, err := e.clinical.AddObservation(e.ctx, b, doc.Document.ID, domain.ObservationInput{Analyte: "X", Value: "1"})
			return err
		}(),
		"add finding": func() error {
			_, err := e.clinical.AddFinding(e.ctx, b, doc.Document.ID, domain.FindingInput{Text: "x"})
			return err
		}(), "confirm approvals": func() error { _, err := e.clinical.ConfirmApprovals(e.ctx, b, doc.Document.ID); return err }(),
	}
	for name, err := range checks {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: err = %v, want not found", name, err)
		}
	}
	res, err := e.clinical.Search(e.ctx, b, "glucose")
	if err != nil || len(res.Hits) != 0 {
		t.Fatalf("B's search = %+v, %v", res.Hits, err)
	}
	groups, err := e.inbox.Groups(e.ctx, b)
	if err != nil || len(groups) != 0 {
		t.Fatalf("B's groups = %+v, %v", groups, err)
	}
	if d, err := e.clinical.Document(e.ctx, a, doc.Document.ID); err != nil || len(d.Observations) != 1 || len(d.Findings) != 1 {
		t.Fatalf("A's document = %+v, %v", d, err)
	}
}

func TestGivenANewIndexVersionWhenRebuildingThenSearchFindsTheSameRecords(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc := e.filedLab(sc, e.person(sc, "Nguyễn Văn Mẫu"), "lab")

	if rebuilt, err := e.store.RebuildSearchIndex(e.ctx, domain.SearchIndexVersion); err != nil || !rebuilt {
		t.Fatalf("first rebuild = %v, %v", rebuilt, err)
	}
	if rebuilt, err := e.store.RebuildSearchIndex(e.ctx, domain.SearchIndexVersion); err != nil || rebuilt {
		t.Fatalf("same version = %v, %v", rebuilt, err)
	}
	for q, kind := range map[string]domain.SearchKind{"kham suc khoe": domain.SearchFinding, "da khoa mau": domain.SearchVisit} {
		res, err := e.clinical.Search(e.ctx, sc, q)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, h := range res.Hits {
			found = found || (h.Kind == kind && h.DocumentID == doc.Document.ID) || (h.Kind == kind && h.VisitID == doc.Visit.ID)
		}
		if !found {
			t.Fatalf("%q hits = %+v", q, res.Hits)
		}
	}
}

// fileUploads reads uploads as one item for the test person and files it.
func (e *env) fileUploads(sc domain.Scope, person domain.Person, files ...domain.Upload) (DocumentDetail, string) {
	e.t.Helper()
	resp := e.upload(sc, true, files...)
	e.drain()
	filed, err := e.inbox.File(e.ctx, sc, resp.Items[0].ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14", Type: "lab"}})
	if err != nil {
		e.t.Fatalf("file: %v", err)
	}
	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		e.t.Fatalf("document: %v", err)
	}
	return doc, resp.Items[0].ID
}

func TestGivenAPDFPageWithoutResultsWhenRemovedThenTheOtherPagesAndValuesStay(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc, _ := e.fileUploads(sc, e.person(sc, "Nguyễn Văn Mẫu"), pdf("twopage lab"))
	if len(doc.Pages) != 2 {
		t.Fatalf("pages = %+v", doc.Pages)
	}
	appendix := doc.Pages[1]

	after, err := e.clinical.RemoveDocumentPage(e.ctx, sc, doc.Document.ID, appendix.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Pages) != 1 || after.Pages[0].ID != doc.Pages[0].ID || len(after.Document.Files) != 1 {
		t.Fatalf("after = %+v", after)
	}
	if len(after.Observations) != 1 || len(after.Findings) != 1 {
		t.Fatalf("records = %+v, %+v", after.Observations, after.Findings)
	}
	if res, err := e.clinical.Search(e.ctx, sc, "phu luc"); err != nil || len(res.Hits) != 0 {
		t.Fatalf("hits = %+v, %v", res, err)
	}
	if _, err := e.clinical.RemoveDocumentPage(e.ctx, sc, doc.Document.ID, appendix.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("remove again = %v", err)
	}

	// A new reading of the file keeps the page removed.
	fileID := doc.Document.Files[0].ID
	if _, err := e.store.SavePages(e.ctx, sc.TenantID, fileID, []domain.Page{doc.Pages[0], appendix}); err != nil {
		t.Fatal(err)
	}
	again, err := e.clinical.Document(e.ctx, sc, doc.Document.ID)
	if err != nil || len(again.Pages) != 1 {
		t.Fatalf("after a new reading = %+v, %v", again.Pages, err)
	}
	if res, err := e.clinical.Search(e.ctx, sc, "phu luc"); err != nil || len(res.Hits) != 0 {
		t.Fatalf("hits after a new reading = %+v, %v", res, err)
	}

	if _, err := e.clinical.RemoveDocumentPage(e.ctx, sc, doc.Document.ID, doc.Pages[0].ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("remove the last page = %v", err)
	}
}

func TestGivenAPhotoWithResultsWhenItsPageIsRemovedThenItsRecordsAreAuditedAndTheFileGoes(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	doc, itemID := e.fileUploads(sc, e.person(sc, "Nguyễn Văn Mẫu"), pdf("lab front"), pdf("lab back"))
	if len(doc.Pages) != 2 || len(doc.Observations) != 1 || doc.Observations[0].PageID != doc.Pages[0].ID {
		t.Fatalf("document = %+v", doc)
	}
	front := doc.Document.Files[0]

	after, err := e.clinical.RemoveDocumentPage(e.ctx, sc, doc.Document.ID, doc.Pages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Pages) != 1 || len(after.Document.Files) != 1 || after.Document.Files[0].ID == front.ID {
		t.Fatalf("after = %+v", after)
	}
	if len(after.Observations) != 0 || len(after.Findings) != 0 {
		t.Fatalf("records = %+v, %+v", after.Observations, after.Findings)
	}
	for entity, id := range map[domain.AuditEntity]string{domain.AuditObservation: doc.Observations[0].ID, domain.AuditFinding: doc.Findings[0].ID} {
		hist, err := e.clinical.History(e.ctx, sc, entity, id)
		if err != nil || len(hist) != 1 || hist[0].Action != "delete" {
			t.Fatalf("%s history = %+v, %v", entity, hist, err)
		}
	}
	if _, err := e.files.Open(e.ctx, front.SHA256); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bytes still there: %v", err)
	}

	// Back in the inbox, the item holds only the page that stayed.
	if _, err := e.inbox.Unfile(e.ctx, sc, doc.Document.ID); err != nil {
		t.Fatal(err)
	}
	if item := e.item(sc, itemID); len(item.Item.Files) != 1 || len(item.Pages) != 1 {
		t.Fatalf("item = %+v", item)
	}
}

func TestGivenAViewerOrAnotherDocumentWhenRemovingAPageThenItIsRefused(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	doc, _ := e.fileUploads(sc, person, pdf("twopage lab"))
	other, _ := e.fileUploads(sc, person, pdf("twopage lab other"))

	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.clinical.RemoveDocumentPage(e.ctx, viewer, doc.Document.ID, doc.Pages[1].ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer = %v", err)
	}
	if _, err := e.clinical.RemoveDocumentPage(e.ctx, sc, doc.Document.ID, other.Pages[1].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other document's page = %v", err)
	}
}

func TestGivenAReadItemWhenAPageIsRemovedThenItIsReadAgainWithoutIt(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "twopage lab")
	if len(d.Pages) != 2 || len(e.reader.calls) != 1 {
		t.Fatalf("pages = %+v, reads = %d", d.Pages, len(e.reader.calls))
	}

	after, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, d.Pages[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Pages) != 1 || len(after.Item.Files) != 1 || len(after.Observations) != 1 {
		t.Fatalf("after = %+v", after)
	}
	e.drain()
	if len(e.reader.calls) != 2 || len(e.reader.calls[1]) != 1 {
		t.Fatalf("reads = %+v", e.reader.calls)
	}
	if _, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, d.Pages[0].ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("remove the last page = %v", err)
	}
}

func TestGivenAPhotoWithDraftsWhenItsPageIsRemovedFromTheItemThenTheDraftsAndTheFileGo(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	resp := e.upload(sc, true, pdf("lab front"), pdf("lab back"))
	e.drain()
	d := e.item(sc, resp.Items[0].ID)
	if len(d.Pages) != 2 || len(d.Observations) != 1 || d.Observations[0].PageID != d.Pages[0].ID {
		t.Fatalf("item = %+v", d)
	}
	front := d.Item.Files[0]

	after, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, d.Pages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Pages) != 1 || len(after.Item.Files) != 1 || after.Item.Files[0].ID == front.ID {
		t.Fatalf("after = %+v", after)
	}
	if len(after.Observations) != 0 || len(after.Findings) != 0 {
		t.Fatalf("drafts = %+v, %+v", after.Observations, after.Findings)
	}
	if _, err := e.files.Open(e.ctx, front.SHA256); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bytes still there: %v", err)
	}
}

func TestGivenAViewerOrAFiledItemWhenRemovingAnItemPageThenItIsRefused(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "twopage lab")
	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.inbox.RemoveItemPage(e.ctx, viewer, d.Item.ID, d.Pages[1].ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer = %v", err)
	}
	other := e.uploadRead(sc, "twopage lab other")
	if _, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, other.Pages[1].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other item's page = %v", err)
	}
	if _, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: e.person(sc, "Nguyễn Văn Mẫu").ID, Date: "2025-03-14"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, d.Pages[1].ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("filed item = %v", err)
	}
}
