package service

import (
	"errors"
	"io"
	"strings"
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

func TestGivenFilesWhenUploadingThenDuplicatesAreReportedAndNotStoredTwice(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	first := e.upload(owner, false, pdf("a"), pdf("b"))
	if len(first.Items) != 2 || len(first.Files) != 2 || first.Files[0].Duplicate {
		t.Fatalf("first upload = %+v", first)
	}
	again := e.upload(owner, false, pdf("a"), pdf("c"), pdf("c"))
	if len(again.Items) != 1 || !again.Files[0].Duplicate || again.Files[1].Duplicate || !again.Files[2].Duplicate {
		t.Fatalf("second upload = %+v", again)
	}
	items, err := e.inbox.Items(e.ctx, owner)
	if err != nil || len(items) != 3 {
		t.Fatalf("items = %d, %v", len(items), err)
	}
}

func TestGivenPagesMarkedAsOneDocumentWhenUploadingThenTheyShareOneItemInOrder(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	resp := e.upload(owner, true, pdf("page1"), pdf("page2"), pdf("page3"))
	if len(resp.Items) != 1 || len(resp.Items[0].Files) != 3 || resp.Items[0].Files[2].OriginalName != "page3.pdf" {
		t.Fatalf("items = %+v", resp.Items)
	}
}

func TestGivenBadUploadsWhenUploadingThenNothingIsStored(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	e.inbox.Limits = Limits{MaxFileBytes: 64, MaxBatchBytes: 100}
	cases := map[string][]domain.Upload{
		"html":      {pdf("ok"), {Name: "page.html", Data: []byte("<html>")}},
		"too large": {{Name: "big.pdf", Data: []byte("%PDF-" + strings.Repeat("x", 80))}},
		"batch":     {pdf("1aaaaaaaaaaaaaaaaaaaaaaa"), pdf("2aaaaaaaaaaaaaaaaaaaaaaa"), pdf("3aaaaaaaaaaaaaaaaaaaaaaa")},
		"empty":     {{Name: "empty.pdf"}},
		"none":      {},
	}
	for name, files := range cases {
		if _, err := e.inbox.Upload(e.ctx, owner, UploadRequest{Files: files}); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if items, _ := e.inbox.Items(e.ctx, owner); len(items) != 0 {
		t.Fatalf("items stored: %d", len(items))
	}
}

func TestGivenAnUploadedPathWhenStoringThenOnlyTheBaseNameIsKept(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	resp := e.upload(owner, false, domain.Upload{Name: "../../Users/x/Desktop/ket-qua.pdf", Data: pdf("p").Data})
	if got := resp.Files[0].File.OriginalName; got != "ket-qua.pdf" {
		t.Fatalf("name = %q", got)
	}
}

func TestGivenAnInboxItemWhenFiledUnderANewVisitThenProviderIsReusedByFoldedName(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	child := e.person(owner, "Bé An")
	items := e.upload(owner, false, pdf("x"), pdf("y")).Items
	filed, err := e.inbox.File(e.ctx, owner, items[0].ID, domain.FileInput{Kind: "lab_result", Title: "Xét nghiệm máu",
		Visit: &domain.VisitInput{PersonID: child.ID, Date: "2025-06-01", Type: "lab", ProviderName: "BV Nhi Đồng 1", ProviderKind: "hospital"}})
	if err != nil {
		t.Fatal(err)
	}
	if filed.Visit.Provider == nil || filed.Visit.Provider.Name != "BV Nhi Đồng 1" || filed.Visit.DocumentCount != 1 {
		t.Fatalf("visit = %+v", filed.Visit)
	}
	second, err := e.inbox.File(e.ctx, owner, items[1].ID, domain.FileInput{Kind: "prescription",
		Visit: &domain.VisitInput{PersonID: child.ID, Date: "2025-07-01", ProviderName: "bv nhi dong 1"}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Visit.Provider.ID != filed.Visit.Provider.ID {
		t.Fatalf("provider not reused: %+v vs %+v", second.Visit.Provider, filed.Visit.Provider)
	}
	if _, err := e.inbox.File(e.ctx, owner, items[0].ID, domain.FileInput{VisitID: filed.Visit.ID}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("filing twice = %v", err)
	}
	visits, err := e.records.Visits(e.ctx, owner, child.ID, port.VisitFilter{})
	if err != nil || len(visits) != 2 || visits[0].Date != "2025-07-01" {
		t.Fatalf("timeline = %+v, %v", visits, err)
	}
	labs, _ := e.records.Visits(e.ctx, owner, child.ID, port.VisitFilter{Type: "lab"})
	if len(labs) != 1 {
		t.Fatalf("filtered = %d", len(labs))
	}
	if items, _ := e.inbox.Items(e.ctx, owner); len(items) != 0 {
		t.Fatalf("inbox still has %d", len(items))
	}
}

func TestGivenAFilingRequestWithoutOrWithBothVisitsWhenFilingThenItIsInvalid(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	item := e.upload(owner, false, pdf("x")).Items[0]
	if _, err := e.inbox.File(e.ctx, owner, item.ID, domain.FileInput{}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("no visit = %v", err)
	}
	if _, err := e.inbox.File(e.ctx, owner, item.ID, domain.FileInput{VisitID: "v", Visit: &domain.VisitInput{}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("both visits = %v", err)
	}
}

func readAll(t *testing.T, r io.ReadCloser) string {
	t.Helper()
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGivenADiscardedItemWhenNothingElseUsesItsFileThenTheBytesAreDeleted(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	item := e.upload(owner, false, pdf("gone")).Items[0]
	sha := item.Files[0].SHA256
	if err := e.inbox.Discard(e.ctx, owner, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.files.Open(e.ctx, sha); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bytes still there: %v", err)
	}
	again := e.upload(owner, false, pdf("gone"))
	if again.Files[0].Duplicate || len(again.Items) != 1 {
		t.Fatalf("re-upload after discard = %+v", again)
	}
}

func TestGivenTheSameFileInTwoTenantsWhenOneDeletesItThenTheOtherKeepsTheBytes(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	other, _ := e.tenants.Create(e.ctx, owner.UserID, "Other")
	otherScope := e.scope(owner.UserID, other.ID)
	a := e.upload(owner, false, pdf("shared")).Items[0]
	b := e.upload(otherScope, false, pdf("shared")).Items[0]
	if b.Files[0].ID == a.Files[0].ID {
		t.Fatal("tenants share a file row")
	}
	if err := e.inbox.Discard(e.ctx, owner, a.ID); err != nil {
		t.Fatal(err)
	}
	_, body, err := e.records.OpenFile(e.ctx, otherScope, b.Files[0].ID)
	if err != nil {
		t.Fatalf("other tenant lost the bytes: %v", err)
	}
	if !strings.Contains(readAll(t, body), "shared") {
		t.Fatal("wrong bytes")
	}
}

func TestGivenADocumentWhenDeletedThenTheVisitCanBeDeletedAndTheBytesGo(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	p := e.person(owner, "Ông")
	item := e.upload(owner, false, pdf("doc")).Items[0]
	filed, err := e.inbox.File(e.ctx, owner, item.ID, domain.FileInput{Visit: &domain.VisitInput{PersonID: p.ID, Date: "2024-01-02"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.records.DeleteVisit(e.ctx, owner, filed.Visit.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("delete visit with document = %v", err)
	}
	if err := e.records.DeletePerson(e.ctx, owner, p.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("delete person with visit = %v", err)
	}
	if err := e.records.DeleteDocument(e.ctx, owner, filed.Document.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.files.Open(e.ctx, item.Files[0].SHA256); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bytes still there: %v", err)
	}
	if err := e.records.DeleteVisit(e.ctx, owner, filed.Visit.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.records.DeletePerson(e.ctx, owner, p.ID); err != nil {
		t.Fatal(err)
	}
}

func TestGivenAViewerWhenChangingRecordsThenEveryChangeIsForbidden(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	viewer := e.join(owner, "viewer@example.test", "viewer")
	p := e.person(owner, "An")
	if _, err := e.records.People(e.ctx, viewer); err != nil {
		t.Fatalf("viewer read = %v", err)
	}
	checks := map[string]error{
		"create person": func() error {
			_, err := e.records.CreatePerson(e.ctx, viewer, domain.PersonInput{DisplayName: "X"})
			return err
		}(),
		"create visit": func() error {
			_, err := e.records.CreateVisit(e.ctx, viewer, domain.VisitInput{PersonID: p.ID, Date: "2025-01-01"})
			return err
		}(),
		"upload": func() error {
			_, err := e.inbox.Upload(e.ctx, viewer, UploadRequest{Files: []domain.Upload{pdf("v")}})
			return err
		}(),
		"delete person": e.records.DeletePerson(e.ctx, viewer, p.ID),
	}
	for name, err := range checks {
		if !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%s = %v", name, err)
		}
	}
}

func TestGivenAPersonLinkedToANonMemberWhenSavingThenItIsInvalid(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	if _, err := e.records.CreatePerson(e.ctx, owner, domain.PersonInput{DisplayName: "X", UserID: "someone-else"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("link to non-member = %v", err)
	}
}

func TestGivenAVisitWhenUpdatedThenFieldsAndProviderChange(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	p := e.person(owner, "An")
	v, err := e.records.CreateVisit(e.ctx, owner, domain.VisitInput{PersonID: p.ID, Date: "2025-01-01", Type: "dental", ProviderName: "Nha khoa A"})
	if err != nil {
		t.Fatal(err)
	}
	v, err = e.records.UpdateVisit(e.ctx, owner, v.ID, domain.VisitInput{PersonID: p.ID, Date: "2025-01-02", Type: "dental", Title: "Trám răng"})
	if err != nil || v.Date != "2025-01-02" || v.Title != "Trám răng" || v.Provider != nil {
		t.Fatalf("updated = %+v, %v", v, err)
	}
	providers, _ := e.records.Providers(e.ctx, owner)
	if len(providers) != 1 {
		t.Fatalf("providers = %+v", providers)
	}
}

func TestGivenAFiledItemWhenDiscardingThenItIsAConflictAndTheDocumentStays(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	p := e.person(owner, "An")
	item := e.upload(owner, false, pdf("kept")).Items[0]
	filed, err := e.inbox.File(e.ctx, owner, item.ID, domain.FileInput{Visit: &domain.VisitInput{PersonID: p.ID, Date: "2025-01-01"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.inbox.Discard(e.ctx, owner, item.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("discard filed item = %v", err)
	}
	if _, body, err := e.records.OpenFile(e.ctx, owner, filed.Document.Files[0].ID); err != nil {
		t.Fatalf("document file gone: %v", err)
	} else {
		_ = body.Close()
	}
}

// Two tenants share a file's bytes. One keeps uploading it while the other
// uploads and discards it; every stored file must stay readable.
func TestGivenConcurrentUploadsAndDiscardsOfOneFileWhenTheySettleThenNoStoredFileLosesItsBytes(t *testing.T) {
	e := newEnv(t)
	owner := e.setup()
	other, _ := e.tenants.Create(e.ctx, owner.UserID, "Other")
	b := e.scope(owner.UserID, other.ID)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			resp, err := e.inbox.Upload(e.ctx, b, UploadRequest{Files: []domain.Upload{pdf("shared")}})
			if err != nil || len(resp.Items) == 0 {
				continue
			}
			_ = e.inbox.Discard(e.ctx, b, resp.Items[0].ID)
		}
	}()
	for i := 0; i < 200; i++ {
		resp, err := e.inbox.Upload(e.ctx, owner, UploadRequest{Files: []domain.Upload{pdf("shared")}})
		if err != nil {
			t.Fatal(err)
		}
		_, body, err := e.records.OpenFile(e.ctx, owner, resp.Files[0].File.ID)
		if err != nil {
			t.Fatalf("round %d: the file just stored has no bytes: %v", i, err)
		}
		_ = body.Close()
		if len(resp.Items) > 0 {
			if err := e.inbox.Discard(e.ctx, owner, resp.Items[0].ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	<-done
}
