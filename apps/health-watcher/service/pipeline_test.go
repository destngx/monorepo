package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// whitePage is a small blank page image.
func whitePage(t testing.TB) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 140))
	for y := range 140 {
		for x := range 100 {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func line(text string, y float64) domain.Line {
	return domain.Line{Text: text, Confidence: 1, Quad: domain.RectQuad(0.1, y, 0.8, 0.02)}
}

// labLines is a synthetic lab report for the test person.
func labLines() []domain.Line {
	return []domain.Line{
		line("BỆNH VIỆN ĐA KHOA MẪU", 0.08),
		line("PHIẾU KẾT QUẢ XÉT NGHIỆM", 0.12),
		line("Họ tên: NGUYỄN VĂN MẪU Năm sinh: 1985", 0.16),
		line("Mã BN: 24-000123", 0.18),
		line("Chẩn đoán: Khám sức khỏe định kỳ", 0.20),
		line("Glucose (đói) 6.4 H mmol/L 3.9 - 6.1", 0.30),
	}
}

func joinLines(lines []domain.Line) string {
	texts := make([]string, len(lines))
	for i, l := range lines {
		texts[i] = l.Text
	}
	return strings.Join(texts, "\n")
}

// fakeOCR reads synthetic uploads: the label in the file picks the page.
type fakeOCR struct {
	err   error
	calls int
}

func (f *fakeOCR) Read(_ context.Context, path, workDir string) ([]port.OCRPage, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	// Like mac-ocr, refuse a symlink: it reads the link's own file type.
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("unsupported file type public.symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(data)
	lines := labLines()
	switch {
	case strings.Contains(src, "film"):
		lines = nil
	case strings.Contains(src, "noname"):
		lines = lines[4:]
	}
	img := filepath.Join(workDir, "page-1.jpg")
	if err := os.WriteFile(img, whitePage(nopTB{}), 0o600); err != nil {
		return nil, err
	}
	pg := port.OCRPage{Number: 1, Method: domain.MethodOCR, Text: joinLines(lines), Confidence: 1, Lines: lines,
		BarcodesChecked: true, ImagePath: img, Width: 100, Height: 140}
	if strings.Contains(src, "noimage") {
		pg.ImagePath = ""
	}
	return []port.OCRPage{pg}, nil
}

// nopTB lets whitePage run outside a test; encoding a blank image can't fail.
type nopTB struct{ testing.TB }

func (nopTB) Fatal(...any) { panic("encode test image") }

// fakeReader answers with a fixed reading and records what it was sent.
type fakeReader struct {
	out   *domain.ReadingOutput
	err   error
	calls [][]port.ReadPage
}

func labReading() domain.ReadingOutput {
	return domain.ReadingOutput{
		Document: domain.ReadingDocument{Kind: "lab_result", Title: "Phiếu xét nghiệm", Date: "2025-03-14", VisitType: "lab",
			ProviderName: "Bệnh viện Đa khoa Mẫu", ProviderKind: "hospital", Confidence: 90},
		Observations: []domain.ReadingObservation{{Page: 1, Analyte: "Glucose (đói)", Value: "6.4", Unit: "mmol/L",
			Reference: "3.9 - 6.1", Flag: "H", Confidence: 95}},
		Findings: []domain.ReadingFinding{{Page: 1, Kind: "diagnosis", Text: "Khám sức khỏe định kỳ", Confidence: 80}},
	}
}

func (f *fakeReader) Read(_ context.Context, pages []port.ReadPage) (port.ReadResult, error) {
	f.calls = append(f.calls, pages)
	if f.err != nil {
		return port.ReadResult{}, f.err
	}
	out := labReading()
	if f.out != nil {
		out = *f.out
	}
	return port.ReadResult{Model: "test-model", Output: out}, nil
}

func (e *env) drain() {
	e.t.Helper()
	if err := e.pipeline.Drain(e.ctx); err != nil {
		e.t.Fatalf("drain: %v", err)
	}
}

func (e *env) item(sc domain.Scope, id string) ItemDetail {
	e.t.Helper()
	d, err := e.inbox.Item(e.ctx, sc, id)
	if err != nil {
		e.t.Fatalf("item: %v", err)
	}
	return d
}

// uploadRead uploads one file, runs the pipeline and returns the item.
func (e *env) uploadRead(sc domain.Scope, label string) ItemDetail {
	e.t.Helper()
	resp := e.upload(sc, false, pdf(label))
	e.drain()
	return e.item(sc, resp.Items[0].ID)
}

func readingOf(t *testing.T, d ItemDetail) domain.Reading {
	t.Helper()
	if d.Item.Reading == nil {
		t.Fatal("no reading")
	}
	return *d.Item.Reading
}

func TestGivenALabReportWhenItIsReadThenDraftsAndAProposalAreReadyAndNothingPersonalIsSent(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")

	d := e.uploadRead(sc, "lab")

	rd := readingOf(t, d)
	if rd.Status != domain.ReadingDone || rd.Model != "test-model" || rd.PagesSent != 1 {
		t.Fatalf("reading = %+v", rd)
	}
	if rd.Proposal.PersonID != person.ID || rd.Proposal.Date != "2025-03-14" || rd.Proposal.Kind != "lab_result" {
		t.Fatalf("proposal = %+v", rd.Proposal)
	}
	if d.Item.OCR == nil || d.Item.OCR.Done != 1 || d.Item.OCR.Total != 1 {
		t.Fatalf("ocr = %+v", d.Item.OCR)
	}
	if len(d.Observations) != 1 || d.Observations[0].MetricCode != "glucose" || d.Observations[0].Status != domain.StatusDraft {
		t.Fatalf("observations = %+v", d.Observations)
	}
	if len(d.Findings) != 1 || d.Findings[0].Kind != "diagnosis" {
		t.Fatalf("findings = %+v", d.Findings)
	}
	if len(d.Pages) != 1 || d.Pages[0].Mask == nil || d.Pages[0].Mask.Status != domain.MaskReady {
		t.Fatalf("pages = %+v", d.Pages)
	}
	sent := e.reader.calls[0][0]
	for _, secret := range []string{"NGUYỄN VĂN MẪU", "24-000123"} {
		if strings.Contains(sent.Text, secret) {
			t.Fatalf("sent text holds %q: %q", secret, sent.Text)
		}
	}
	if !strings.Contains(sent.Text, "Glucose") || len(sent.JPEG) == 0 {
		t.Fatalf("sent = %q, %d image bytes", sent.Text, len(sent.JPEG))
	}
}

func TestGivenAReadItemWhenFiledThenDraftsAreApprovedWithTheVisitDateAndSearchable(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")

	filed, err := e.inbox.File(e.ctx, sc, d.Item.ID, domain.FileInput{Kind: "lab_result",
		Visit: &domain.VisitInput{PersonID: person.ID, Date: "2025-03-14", Type: "lab", ProviderName: "Bệnh viện Đa khoa Mẫu"}})
	if err != nil {
		t.Fatal(err)
	}

	doc, err := e.clinical.Document(e.ctx, sc, filed.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Observations) != 1 || doc.Observations[0].Status != domain.StatusApproved ||
		doc.Observations[0].SampleDate != "2025-03-14" || doc.Observations[0].PersonID != person.ID {
		t.Fatalf("observations = %+v", doc.Observations)
	}
	if len(doc.Pages) != 1 || len(doc.Findings) != 1 {
		t.Fatalf("pages = %d, findings = %d", len(doc.Pages), len(doc.Findings))
	}
	res, err := e.clinical.Search(e.ctx, sc, "kham suc khoe")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[domain.SearchKind]bool{}
	for _, h := range res.Hits {
		kinds[h.Kind] = true
		if h.DocumentID != filed.Document.ID {
			t.Fatalf("hit = %+v", h)
		}
	}
	if !kinds[domain.SearchFinding] || !kinds[domain.SearchPage] {
		t.Fatalf("hits = %+v", res.Hits)
	}
}

func TestGivenAFirstPageWithoutANameWhenReadThenItWaitsForACheckAndSendsAfterIt(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()

	d := e.uploadRead(sc, "noname")

	if rd := readingOf(t, d); rd.Status != domain.ReadingNeedsCheck || !strings.Contains(rd.Reason, "Page 1") {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != 0 {
		t.Fatal("pages were sent before a check")
	}
	if err := e.inbox.SendReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.drain()
	d = e.item(sc, d.Item.ID)
	if rd := readingOf(t, d); rd.Status != domain.ReadingDone {
		t.Fatalf("reading = %+v", rd)
	}
	if d.Pages[0].Mask.Status != domain.MaskChecked {
		t.Fatalf("mask = %+v", d.Pages[0].Mask)
	}
}

func TestGivenAPageWithoutAnImageWhenSendingThenItIsRefused(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "noimage")

	if rd := readingOf(t, d); rd.Status != domain.ReadingNeedsCheck {
		t.Fatalf("reading = %+v", rd)
	}
	if err := e.inbox.SendReading(e.ctx, sc, d.Item.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("send = %v", err)
	}
}

func TestGivenAFilmPhotoWithoutTextWhenReadThenItIsSkipped(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()

	d := e.uploadRead(sc, "film")

	if rd := readingOf(t, d); rd.Status != domain.ReadingSkipped || rd.Reason != domain.SkipNoText {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != 0 {
		t.Fatal("a blank page was sent")
	}
}

func TestGivenNoAIProviderWhenAFileIsReadThenOnlyOCRRunsAndReadingIsOff(t *testing.T) {
	e := newEnv(t)
	e.pipeline.Reader = nil
	sc := e.setup()

	d := e.uploadRead(sc, "lab")

	if rd := readingOf(t, d); rd.Status != domain.ReadingOff {
		t.Fatalf("reading = %+v", rd)
	}
	if d.AIEnabled || len(d.Pages) != 1 || d.Pages[0].Text == "" {
		t.Fatalf("detail = %+v", d)
	}
	if err := e.inbox.RetryReading(e.ctx, sc, d.Item.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("retry = %v", err)
	}
}

func TestGivenAFailingProviderWhenReadingThenItRetriesWithBackoffAndFailsAfterTheLastAttempt(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	e.reader.err = errors.New("provider down")

	d := e.uploadRead(sc, "lab")
	rd := readingOf(t, d)
	if rd.Status != domain.ReadingQueued || rd.Attempts != 1 || rd.Error != "provider down" || rd.NextAttemptAt == nil {
		t.Fatalf("reading = %+v", rd)
	}
	if got := rd.NextAttemptAt.Sub(e.now); got != time.Minute {
		t.Fatalf("next attempt in %s", got)
	}
	for range domain.MaxJobAttempts - 1 {
		e.now = e.now.Add(7 * time.Hour)
		e.drain()
	}
	rd = readingOf(t, e.item(sc, d.Item.ID))
	if rd.Status != domain.ReadingFailed || rd.Attempts != domain.MaxJobAttempts {
		t.Fatalf("reading = %+v", rd)
	}
	if len(e.reader.calls) != domain.MaxJobAttempts {
		t.Fatalf("calls = %d", len(e.reader.calls))
	}

	e.reader.err = nil
	if err := e.inbox.RetryReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.drain()
	if rd := readingOf(t, e.item(sc, d.Item.ID)); rd.Status != domain.ReadingDone || rd.Error != "" {
		t.Fatalf("reading = %+v", rd)
	}
}

func TestGivenAFailingOCRToolWhenReadingThenTheItemShowsTheError(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.ocr.err = errors.New("mac-ocr crashed")

	d := e.uploadRead(sc, "lab")

	if d.Item.OCR == nil || d.Item.OCR.Done != 0 || d.Item.OCR.Error != "mac-ocr crashed" {
		t.Fatalf("ocr = %+v", d.Item.OCR)
	}
	for range domain.MaxJobAttempts - 1 {
		e.now = e.now.Add(7 * time.Hour)
		e.drain()
	}
	if d = e.item(sc, d.Item.ID); d.Item.OCR.Failed != 1 {
		t.Fatalf("ocr = %+v", d.Item.OCR)
	}

	e.ocr.err = nil
	if err := e.inbox.RetryReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.drain()
	d = e.item(sc, d.Item.ID)
	if d.Item.OCR.Done != 1 || d.Item.OCR.Error != "" || readingOf(t, d).Status != domain.ReadingDone {
		t.Fatalf("ocr = %+v, reading = %+v", d.Item.OCR, d.Item.Reading)
	}
}

func TestGivenDraftsWhenSkippingThenTheyAreDroppedAndNotReadAgain(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	d := e.uploadRead(sc, "lab")

	if err := e.inbox.SkipReading(e.ctx, sc, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	e.drain()
	d = e.item(sc, d.Item.ID)
	if rd := readingOf(t, d); rd.Status != domain.ReadingSkipped || rd.Reason != domain.SkipByUser {
		t.Fatalf("reading = %+v", rd)
	}
	if len(d.Observations) != 0 || len(d.Findings) != 0 {
		t.Fatalf("drafts left: %d, %d", len(d.Observations), len(d.Findings))
	}
}

func TestGivenDrawnBoxesWhenSavedThenThePageIsRemaskedAndNeedsACheckAgain(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	d := e.uploadRead(sc, "lab")
	before := d.Pages[0].Mask

	box := domain.RectQuad(0.6, 0.5, 0.2, 0.1)
	pg, err := e.inbox.SetPageBoxes(e.ctx, sc, d.Item.ID, d.Pages[0].ID, []domain.Quad{box})
	if err != nil {
		t.Fatal(err)
	}
	if len(pg.Mask.Manual) != 1 || len(pg.Mask.Boxes) != len(before.Boxes)+1 {
		t.Fatalf("mask = %+v", pg.Mask)
	}
	if _, err := e.inbox.SetPageBoxes(e.ctx, sc, d.Item.ID, d.Pages[0].ID, []domain.Quad{{{X: 2, Y: 0}}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad box = %v", err)
	}
	img, err := e.records.OpenPageImage(e.ctx, sc, pg.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	_ = img.Close()
}

func TestGivenTwoItemsFromOneVisitWhenGroupedThenTheyFileAsOneVisit(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	e.upload(sc, false, pdf("lab-a"), pdf("lab-b"))
	e.drain()

	groups, err := e.inbox.Groups(e.ctx, sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Items) != 2 || groups[0].PersonID != person.ID || groups[0].Date != "2025-03-14" {
		t.Fatalf("groups = %+v", groups)
	}
	g := groups[0]
	visit, err := e.inbox.FileGroup(e.ctx, sc, GroupFileInput{
		Visit:     domain.VisitInput{PersonID: g.PersonID, Date: g.Date, Type: string(g.VisitType), ProviderName: g.ProviderName},
		Documents: []GroupDocument{{ItemID: g.Items[0].ID, Kind: "lab_result"}, {ItemID: g.Items[1].ID, Kind: "lab_result"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(visit.Documents) != 2 || visit.FlaggedCount != 2 {
		t.Fatalf("visit = %+v", visit)
	}
	items, err := e.inbox.Items(e.ctx, sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items left = %d", len(items))
	}
}

func TestGivenABacklogPersonWhenUploadingThenTheProposalUsesThem(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.person(sc, "Nguyễn Văn Mẫu")
	other := e.person(sc, "Trần Thị Mẫu")

	resp, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourceWeb, Files: []domain.Upload{pdf("lab")}, PersonID: other.ID})
	if err != nil {
		t.Fatal(err)
	}
	e.drain()

	if rd := readingOf(t, e.item(sc, resp.Items[0].ID)); rd.Proposal.PersonID != other.ID {
		t.Fatalf("proposal = %+v", rd.Proposal)
	}
	if _, err := e.inbox.Upload(e.ctx, sc, UploadRequest{Source: domain.SourceWeb, Files: []domain.Upload{pdf("x")}, PersonID: "nobody"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown person = %v", err)
	}
}

func TestGivenAStoppedEngineWhenStartedThenFilesWithoutPagesAreRead(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	e.upload(sc, false, pdf("lab"))

	ctx, cancel := context.WithCancel(e.ctx)
	wg, err := e.pipeline.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		items, err := e.inbox.Items(e.ctx, sc)
		if err != nil {
			t.Fatal(err)
		}
		if rd := items[0].Reading; rd != nil && rd.Status == domain.ReadingDone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not read: %+v", items[0].Reading)
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	wg.Wait()
}
