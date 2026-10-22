package service

import (
	"errors"
	"testing"

	"apps/health-watcher/domain"
)

func (e *env) addFiles(sc domain.Scope, itemID string, files ...domain.Upload) UploadResponse {
	e.t.Helper()
	resp, err := e.inbox.AddFiles(e.ctx, sc, itemID, files, nil)
	if err != nil {
		e.t.Fatalf("add files: %v", err)
	}
	return resp
}

func TestGivenAReadItemWhenFilesAreAddedThenItIsReadAgainWithoutTheFilmPhoto(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")

	resp := e.addFiles(sc, d.Item.ID, pdf("lab-2"), pdf("film"))

	if len(resp.Items) != 1 || len(resp.Items[0].Files) != 3 || resp.Items[0].Files[2].OriginalName != "film.pdf" {
		t.Fatalf("items = %+v", resp.Items)
	}
	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingWaiting {
		t.Fatalf("reading after add = %+v", rd)
	}
	e.drain()
	d = e.item(sc, d.Item.ID)
	if rd := readingOf(t, d); rd.Status != domain.ReadingDone || rd.PagesSent != 2 {
		t.Fatalf("reading = %+v", rd)
	}
	if len(d.Pages) != 3 || len(e.reader.calls) != 2 || len(e.reader.calls[1]) != 2 {
		t.Fatalf("pages = %d, calls = %d", len(d.Pages), len(e.reader.calls))
	}
	if d.Pages[2].Mask != nil {
		t.Fatalf("the film photo was masked for sending: %+v", d.Pages[2].Mask)
	}
	if len(d.Observations) != 1 || len(d.Findings) != 1 {
		t.Fatalf("drafts = %d values, %d findings", len(d.Observations), len(d.Findings))
	}
}

func TestGivenFilesAddedWhileTheProviderReadsWhenItAnswersThenTheAnswerIsDroppedAndAllFilesAreRead(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	resp := e.upload(sc, false, pdf("lab"))
	itemID := resp.Items[0].ID
	e.reader.during = func() { e.addFiles(sc, itemID, pdf("lab-2")) }

	e.drain()

	d := e.item(sc, itemID)
	if rd := readingOf(t, d); rd.Status != domain.ReadingDone || rd.PagesSent != 2 {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != 2 || len(e.reader.calls[1]) != 2 {
		t.Fatalf("calls = %d", len(e.reader.calls))
	}
}

func TestGivenAnItemWhenAddingFilesThenDuplicatesSkippedReadingsAndFiledItemsAreRespected(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")

	dup := e.addFiles(sc, d.Item.ID, pdf("lab"))
	if !dup.Files[0].Duplicate || len(dup.Items[0].Files) != 1 {
		t.Fatalf("duplicate = %+v", dup)
	}
	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingDone {
		t.Fatalf("a duplicate restarted the reading: %+v", rd)
	}

	if err := e.inbox.SkipReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.addFiles(sc, d.Item.ID, pdf("lab-2"))
	e.drain()
	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingSkipped || rd.Reason != domain.SkipByUser {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != 1 {
		t.Fatalf("calls = %d", len(e.reader.calls))
	}

	if err := e.inbox.Discard(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.inbox.AddFiles(e.ctx, sc, d.Item.ID, []domain.Upload{pdf("lab-3")}, nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("add to a discarded item = %v", err)
	}
}

func TestGivenAFilmPhotoItemWhenItsReportIsAddedThenTheReportIsRead(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "film")

	e.addFiles(sc, d.Item.ID, pdf("lab"))
	e.drain()

	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingDone || rd.PagesSent != 1 {
		t.Fatalf("reading = %+v", rd)
	}
}
