package service

import (
	"errors"
	"image/jpeg"
	"testing"

	"apps/health-watcher/domain"
)

// filmReading is labReading with one X-ray film found on its page.
func filmReading() *domain.ReadingOutput {
	out := labReading()
	out.Pictures = []domain.ReadingPicture{
		{Page: 1, Kind: "xray", Caption: "Phổi thẳng", X0: 0.2, Y0: 0.2, X1: 0.8, Y1: 0.6},
		// Too small to be a picture: a logo or a stamp.
		{Page: 1, Kind: "photo", X0: 0.1, Y0: 0.1, X1: 0.12, Y1: 0.12},
		// No such page.
		{Page: 2, Kind: "ecg", X0: 0.1, Y0: 0.1, X1: 0.5, Y1: 0.5},
	}
	return &out
}

func TestGivenAReportWithAFilmWhenReadThenTheFilmIsKeptAsItsOwnPictureOnTheItemAndItsDocument(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	e.reader.out = filmReading()

	d := e.uploadRead(sc, "lab")

	if len(d.Pictures) != 1 {
		t.Fatalf("pictures = %+v", d.Pictures)
	}
	pic := d.Pictures[0]
	if pic.Kind != "xray" || pic.Caption != "Phổi thẳng" || pic.PageID != d.Pages[0].ID || pic.Position != 0 {
		t.Fatalf("picture = %+v", pic)
	}
	// A blank test page has no edge to fit, so the model's box is kept.
	if pic.Box != (domain.Rect{X0: 0.2, Y0: 0.2, X1: 0.8, Y1: 0.6}) || pic.Width != 60 || pic.Height != 56 {
		t.Fatalf("box = %+v, %dx%d", pic.Box, pic.Width, pic.Height)
	}
	body, err := e.records.OpenPictureImage(e.ctx, sc, pic.ID)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(body)
	_ = body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 60 || b.Dy() != 56 {
		t.Fatalf("image = %v", b)
	}

	filed, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "imaging_report",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14", Type: "imaging"}})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pictures) != 1 || doc.Pictures[0].ID != pic.ID {
		t.Fatalf("document pictures = %+v", doc.Pictures)
	}
}

func TestGivenAPictureWhenTheItemIsReadAgainThenTheNewReadingReplacesItAndItsImageGoes(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	e.reader.out = filmReading()
	d := e.uploadRead(sc, "lab")
	old := d.Pictures[0]

	e.reader.out = nil
	if err := e.inbox.RetryReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.drain()

	if got := e.item(sc, d.Item.ID).Pictures; len(got) != 0 {
		t.Fatalf("pictures = %+v", got)
	}
	if _, err := e.records.OpenPictureImage(e.ctx, sc, old.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("old picture: err = %v", err)
	}
	if _, err := e.files.Open(e.ctx, old.ImageSHA); err == nil {
		t.Fatal("the old picture's image is still stored")
	}
}

func TestGivenAPictureOnARemovedPageWhenTheItemIsShownThenThePictureIsGone(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	e.reader.out = filmReading()
	resp := e.upload(sc, true, pdf("lab"), pdf("lab2"))
	e.drain()
	d := e.item(sc, resp.Items[0].ID)
	if len(d.Pictures) != 2 {
		t.Fatalf("pictures = %+v", d.Pictures)
	}
	pic, kept := d.Pictures[0], d.Pictures[1]

	after, err := e.inbox.RemoveItemPage(e.ctx, sc, d.Item.ID, pic.PageID)
	if err != nil {
		t.Fatal(err)
	}

	if len(after.Pictures) != 1 || after.Pictures[0].ID != kept.ID {
		t.Fatalf("pictures = %+v", after.Pictures)
	}

	if _, err := e.records.OpenPictureImage(e.ctx, sc, pic.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("picture: err = %v", err)
	}
	if _, err := e.files.Open(e.ctx, pic.ImageSHA); err == nil {
		t.Fatal("the removed page's picture is still stored")
	}
}

func TestGivenAnotherTenantsPictureWhenOpeningItThenItIsNotFound(t *testing.T) {
	e := newEnv(t)
	a := e.setup()
	e.person(a, "Nguyễn Văn Mẫu")
	e.reader.out = filmReading()
	pic := e.uploadRead(a, "lab").Pictures[0]
	outsider := e.join(a, "outsider@example.test", "viewer")
	if err := e.tenants.RemoveMember(e.ctx, outsider, outsider.UserID); err != nil {
		t.Fatal(err)
	}
	tb, err := e.tenants.Create(e.ctx, outsider.UserID, "Tenant B")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.records.OpenPictureImage(e.ctx, e.scope(outsider.UserID, tb.ID), pic.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}
