package service

import (
	"errors"
	"testing"

	"apps/health-watcher/domain"
)

func (e *env) stage(sc domain.Scope, files ...domain.Upload) []domain.UploadResult {
	e.t.Helper()
	res, err := e.inbox.Stage(e.ctx, sc, files)
	if err != nil {
		e.t.Fatalf("stage: %v", err)
	}
	return res
}

func TestGivenPhotosSentAheadWhenSendIsPressedThenTheyBecomeOneItemReadOnce(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")

	a := e.stage(sc, pdf("lab"))[0].File.ID
	b := e.stage(sc, pdf("lab-2"))[0].File.ID
	e.drain()
	if len(e.reader.calls) != 0 {
		t.Fatal("a staged photo was sent to the AI before Send")
	}
	if items, _ := e.inbox.Items(e.ctx, sc); len(items) != 0 {
		t.Fatalf("staged photos made items: %+v", items)
	}

	resp, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourcePhone, OneItem: true, FileIDs: []string{a, b}})
	if err != nil || len(resp.Items) != 1 || len(resp.Items[0].Files) != 2 {
		t.Fatalf("send = %+v, %v", resp, err)
	}
	e.drain()
	d := e.item(sc, resp.Items[0].ID)
	if rd := readingOf(t, d); rd.Status != domain.ReadingDone || rd.PagesSent != 2 {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != 1 {
		t.Fatalf("calls = %d", len(e.reader.calls))
	}

	// Sent files are no longer staged: sending them again is refused.
	if _, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourcePhone, FileIDs: []string{a}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("sent twice: %v", err)
	}
}

func TestGivenAPhotoStagedTwiceWhenTheSecondArrivesThenItIsTheSameFileNotADuplicate(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	first := e.stage(sc, pdf("lab"))[0]
	again := e.stage(sc, pdf("lab"))[0]
	if again.Duplicate || again.File.ID != first.File.ID {
		t.Fatalf("again = %+v", again)
	}

	other := e.join(sc, "other@example.test", "editor")
	if res := e.stage(other, pdf("lab"))[0]; !res.Duplicate {
		t.Fatalf("another user's staged photo = %+v", res)
	}
	if _, err := e.inbox.Upload(e.ctx, other, UploadRequest{Source: domain.SourceWeb, FileIDs: []string{first.File.ID}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("another user sent my photo: %v", err)
	}
}

func TestGivenPhotosSentAheadToAnItemWhenAddedThenTheItemIsReadAgainWithThem(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")
	x := e.stage(sc, pdf("lab-2"))[0].File.ID
	e.drain()

	if _, err := e.inbox.AddFiles(e.ctx, sc, d.Item.ID, nil, []string{x}); err != nil {
		t.Fatal(err)
	}
	e.drain()
	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingDone || rd.PagesSent != 2 {
		t.Fatalf("reading = %+v", rd)
	}
}

func TestGivenAPhotoNeverSentWhenADayPassesThenItIsDeleted(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	old := e.stage(sc, pdf("lab"))[0].File
	e.drain()

	e.now = e.now.Add(StagedLifetime + 1)
	e.stage(sc, pdf("lab-2"))

	if _, err := e.store.File(e.ctx, sc.TenantID, old.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("old staged file = %v", err)
	}
	if _, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourcePhone, FileIDs: []string{old.ID}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expired photo sent: %v", err)
	}
}
