package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Pipeline is the background worker: it reads each new file with the OCR
// tool, masks each inbox item's pages on the Mac and sends them to the AI
// provider (HW-READ-1..3, HW-READ-6, HW-AI-7). Jobs live in the database,
// so work survives a restart.
type Pipeline struct {
	Deps
	// OCR, Reader and Workers may change while the pipeline runs; see
	// Configure.
	OCR    port.OCR
	Masker port.ImageMasker
	// Reader is nil when AI is not configured; items then stay OCR-only.
	Reader  port.Reader
	Catalog *domain.Catalog
	// WorkDir is a private directory for the OCR tool's files.
	WorkDir string
	Log     *slog.Logger
	// Workers is how many jobs run at once.
	Workers int
	// Poll is how often idle workers look for due retries.
	Poll time.Duration

	wakeOnce sync.Once
	wake     chan struct{}

	mu sync.RWMutex
	// changed is closed and replaced by each Configure, waking parked workers.
	changed chan struct{}
	// resumeOff is set when Configure turns AI reading on: the next worker
	// sends the items whose reading is off.
	resumeOff atomic.Bool
}

// Configure swaps the OCR tool, the AI reader (nil turns AI reading off)
// and the number of workers while the pipeline runs (HW-OPS-9). A job in
// progress finishes with the tools it started with.
func (p *Pipeline) Configure(ocr port.OCR, reader port.Reader, workers int) {
	p.mu.Lock()
	if p.Reader == nil && reader != nil {
		p.resumeOff.Store(true)
	}
	p.OCR, p.Reader, p.Workers = ocr, reader, workers
	if p.changed != nil {
		close(p.changed)
	}
	p.changed = make(chan struct{})
	p.mu.Unlock()
	p.Wake()
}

func (p *Pipeline) ocr() port.OCR {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.OCR
}

func (p *Pipeline) reader() port.Reader {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Reader
}

// workers is how many workers may claim jobs now, and a channel closed at
// the next Configure.
func (p *Pipeline) workers() (int, <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return min(max(domain.MinWorkers, p.Workers), domain.MaxWorkers), p.changed
}

func (p *Pipeline) wakeChan() chan struct{} {
	p.wakeOnce.Do(func() { p.wake = make(chan struct{}, 1) })
	return p.wake
}

// Wake tells an idle worker that a job was queued.
func (p *Pipeline) Wake() {
	select {
	case p.wakeChan() <- struct{}{}:
	default:
	}
}

// AIEnabled reports whether items are sent for AI reading.
func (p *Pipeline) AIEnabled() bool { return p.reader() != nil }

// Start requeues jobs a stopped engine left running, queues OCR for files
// that have none, and starts the most workers Configure may allow; only
// as many as Workers claim jobs, the rest wait. They stop with ctx.
func (p *Pipeline) Start(ctx context.Context) (*sync.WaitGroup, error) {
	if err := os.MkdirAll(p.WorkDir, 0o700); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}
	if err := p.Store.RequeueRunningJobs(ctx, p.Clock()); err != nil {
		return nil, err
	}
	missing, err := p.Store.FilesWithoutPages(ctx)
	if err != nil {
		return nil, err
	}
	for _, tf := range missing {
		if err := p.Store.EnqueueJob(ctx, domain.Job{ID: NewID(), TenantID: tf.TenantID, Kind: domain.JobOCR, SubjectID: tf.FileID}, p.Clock()); err != nil {
			return nil, err
		}
	}
	poll := p.Poll
	if poll <= 0 {
		poll = 5 * time.Second
	}
	var wg sync.WaitGroup
	for i := range domain.MaxWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.loop(ctx, poll, i)
		}()
	}
	return &wg, nil
}

func (p *Pipeline) loop(ctx context.Context, poll time.Duration, index int) {
	for ctx.Err() == nil {
		if n, changed := p.workers(); index >= n {
			select {
			case <-ctx.Done():
			case <-changed:
			}
			continue
		}
		ran, err := p.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			p.Log.Error("job queue", "error", err)
		}
		if ran {
			continue
		}
		select {
		case <-ctx.Done():
		case <-p.wakeChan():
		case <-time.After(poll):
		}
	}
}

// Drain runs due jobs until none is left; tests use it instead of Start.
func (p *Pipeline) Drain(ctx context.Context) error {
	for {
		ran, err := p.RunOnce(ctx)
		if err != nil || !ran {
			return err
		}
	}
}

// RunOnce claims and runs one due job. It reports false when none is due.
func (p *Pipeline) RunOnce(ctx context.Context) (bool, error) {
	if p.resumeOff.CompareAndSwap(true, false) {
		if err := p.resumeReadings(ctx); err != nil {
			p.resumeOff.Store(true)
			return false, err
		}
	}
	job, err := p.Store.ClaimJob(ctx, p.Clock())
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var runErr error
	switch job.Kind {
	case domain.JobOCR:
		runErr = p.runOCR(ctx, job)
	case domain.JobRead:
		runErr = p.runRead(ctx, job)
	default:
		runErr = fmt.Errorf("unknown job kind %q", job.Kind)
	}
	if ctx.Err() != nil {
		// Stopping: the job is requeued at the next start.
		return true, nil
	}
	if runErr == nil {
		return true, p.Store.CompleteJob(ctx, job.ID, p.Clock())
	}
	return true, p.failed(ctx, job, runErr)
}

// failed records a failed attempt: a retry with backoff, or a failure
// after the last attempt (HW-READ-6).
func (p *Pipeline) failed(ctx context.Context, job domain.Job, runErr error) error {
	attempts := job.Attempts + 1
	msg := clipError(runErr.Error())
	// Ids only: errors carry tool and provider messages, never page text (HW-OPS-5).
	p.Log.Warn("job failed", "job_id", job.ID, "kind", job.Kind, "attempt", attempts, "error", msg)
	now := p.Clock()
	wait, retry := domain.RetryAfter(attempts)
	if retry {
		if err := p.Store.RetryJob(ctx, job.ID, attempts, now.Add(wait), msg, now); err != nil {
			return err
		}
	} else if err := p.Store.FailJob(ctx, job.ID, attempts, msg, now); err != nil {
		return err
	}
	if job.Kind != domain.JobRead {
		return nil
	}
	rd, err := p.Store.Reading(ctx, job.TenantID, job.SubjectID)
	if errors.Is(err, domain.ErrNotFound) {
		rd = domain.Reading{ItemID: job.SubjectID}
	} else if err != nil {
		return err
	}
	rd.Status, rd.Error, rd.UpdatedAt = domain.ReadingQueued, msg, now
	if !retry {
		rd.Status = domain.ReadingFailed
	}
	return ignoreGone(p.Store.SaveReading(ctx, job.TenantID, rd))
}

func clipError(s string) string {
	if utf8.RuneCountInString(s) > 500 {
		return string([]rune(s)[:500]) + "…"
	}
	return s
}

// ignoreGone treats a record deleted meanwhile as done.
func ignoreGone(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	return err
}

// mimeExt names the OCR tool's input so it knows the file type.
var mimeExt = map[string]string{
	"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/heic": ".heic",
}

// runOCR reads one file's pages, stores their rendered images and text,
// and starts reading the inbox items that now have every page.
func (p *Pipeline) runOCR(ctx context.Context, job domain.Job) error {
	f, err := p.Store.File(ctx, job.TenantID, job.SubjectID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil // deleted meanwhile
	}
	if err != nil {
		return err
	}
	src, err := p.Files.Path(f.SHA256)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(p.WorkDir, "ocr-")
	if err != nil {
		return fmt.Errorf("create OCR directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	input := filepath.Join(dir, "input"+mimeExt[f.MIME])
	if err := linkOrCopy(src, input); err != nil {
		return err
	}
	renderDir := filepath.Join(dir, "pages")
	if err := os.Mkdir(renderDir, 0o700); err != nil {
		return fmt.Errorf("create render directory: %w", err)
	}
	read, err := p.ocr().Read(ctx, input, renderDir)
	if err != nil {
		return err
	}

	pages := make([]domain.Page, len(read))
	images := map[string][]byte{}
	for i, rp := range read {
		pg := domain.Page{ID: NewID(), FileID: f.ID, Number: rp.Number, Method: rp.Method, Text: rp.Text, Confidence: rp.Confidence,
			Width: rp.Width, Height: rp.Height, Lines: rp.Lines, Barcodes: rp.Barcodes, BarcodesChecked: rp.BarcodesChecked,
			CreatedAt: p.Clock()}
		if rp.ImagePath != "" {
			data, err := readInside(renderDir, rp.ImagePath)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			pg.ImageSHA = hex.EncodeToString(sum[:])
			images[pg.ImageSHA] = data
		}
		pages[i] = pg
	}

	if err := p.savePages(ctx, job.TenantID, f.ID, pages, images); err != nil {
		return ignoreGone(err)
	}
	items, err := p.Store.NewItemsWithFile(ctx, job.TenantID, f.ID)
	if err != nil {
		return err
	}
	for _, itemID := range items {
		if err := p.ScheduleReading(ctx, job.TenantID, itemID, false); err != nil {
			return err
		}
	}
	return nil
}

// linkOrCopy gives the OCR tool the stored file under a name with the
// right extension: a hard link, or a copy across file systems. Not a
// symlink: mac-ocr reads a symlink's own file type and refuses it.
func linkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open OCR input: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("copy OCR input: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy OCR input: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("copy OCR input: %w", err)
	}
	return nil
}

// readInside reads a file the OCR tool wrote, refusing a path outside its
// render directory.
func readInside(dir, path string) ([]byte, error) {
	rel, err := filepath.Rel(dir, filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("mac-ocr wrote a page image outside its directory")
	}
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return nil, fmt.Errorf("read page image: %w", err)
	}
	return data, nil
}

// savePages stores page images and rows under FileLock, so no release can
// delete an image before its row commits, then drops images nothing uses.
func (p *Pipeline) savePages(ctx context.Context, tenantID, fileID string, pages []domain.Page, images map[string][]byte) error {
	p.FileLock.Lock()
	defer p.FileLock.Unlock()
	var written []string
	for sha, data := range images {
		if err := p.Files.Put(ctx, sha, data); err != nil {
			return errors.Join(err, dropUnused(ctx, p.Deps, written))
		}
		written = append(written, sha)
	}
	var released []string
	err := p.Store.InTx(ctx, func(r port.Repos) error {
		var err error
		released, err = r.SavePages(ctx, tenantID, fileID, pages)
		return err
	})
	if err != nil {
		return errors.Join(err, dropUnused(ctx, p.Deps, written))
	}
	return dropUnused(ctx, p.Deps, released)
}

// ScheduleReading starts an item's AI reading once every file has pages.
// A reading already under way, done or skipped is left alone unless force.
func (p *Pipeline) ScheduleReading(ctx context.Context, tenantID, itemID string, force bool) error {
	prog, err := p.Store.OCRProgress(ctx, tenantID, itemID)
	if err != nil {
		return err
	}
	now := p.Clock()
	rd, err := p.Store.Reading(ctx, tenantID, itemID)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		rd = domain.Reading{ItemID: itemID}
	case err != nil:
		return err
	case !force && rd.Status != domain.ReadingWaiting && rd.Status != domain.ReadingOff:
		return nil
	}
	rd.UpdatedAt, rd.Error = now, ""
	switch {
	case prog.Done < prog.Total:
		rd.Status, rd.Reason = domain.ReadingWaiting, ""
		return ignoreGone(p.Store.SaveReading(ctx, tenantID, rd))
	case p.reader() == nil:
		rd.Status, rd.Reason = domain.ReadingOff, ""
		return ignoreGone(p.Store.SaveReading(ctx, tenantID, rd))
	}
	rd.Status, rd.Reason = domain.ReadingQueued, ""
	if err := p.Store.SaveReading(ctx, tenantID, rd); err != nil {
		return ignoreGone(err)
	}
	if err := p.Store.EnqueueJob(ctx, domain.Job{ID: NewID(), TenantID: tenantID, Kind: domain.JobRead, SubjectID: itemID}, now); err != nil {
		return err
	}
	p.Wake()
	return nil
}

// resumeReadings sends the unfiled items read while AI reading was off.
func (p *Pipeline) resumeReadings(ctx context.Context) error {
	items, err := p.Store.OffReadings(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		if err := p.ScheduleReading(ctx, it.TenantID, it.ItemID, false); err != nil {
			return err
		}
	}
	if len(items) > 0 {
		p.Log.Info("AI reading turned on: sending items read while it was off", "items", len(items))
	}
	return nil
}

// itemPage is a page of an inbox item, in reading order.
type itemPage struct {
	page  domain.Page
	first bool // the first page of its file
}

// itemPages lists an item's pages: files in order, pages in order. It
// reports false when a file has no pages yet.
func (p *Pipeline) itemPages(ctx context.Context, tenantID string, item domain.InboxItem) ([]itemPage, bool, error) {
	var out []itemPage
	for _, f := range item.Files {
		pages, err := p.Store.PagesOf(ctx, tenantID, f.ID)
		if err != nil {
			return nil, false, err
		}
		if len(pages) == 0 {
			return nil, false, nil
		}
		for i, pg := range pages {
			out = append(out, itemPage{page: pg, first: i == 0})
		}
	}
	return out, true, nil
}

func peopleNames(people []domain.Person) []string {
	names := make([]string, len(people))
	for i, pr := range people {
		names[i] = pr.DisplayName
	}
	return names
}

// maskPage builds and stores a page's mask: the black boxes, the masked
// text and the masked image (HW-AI-7). Manual boxes, the rule boxes turned
// off and a person's check carry over from the page's previous mask.
func (p *Pipeline) maskPage(ctx context.Context, tenantID string, ip itemPage, people []string) (domain.PageMask, error) {
	pg := ip.page
	in := domain.MaskInput{Lines: pg.Lines, Text: pg.Text, Barcodes: pg.Barcodes, BarcodesChecked: pg.BarcodesChecked,
		HasImage: pg.HasImage, FirstPage: ip.first, Method: pg.Method, People: people, Width: pg.Width, Height: pg.Height}
	if pg.Mask != nil {
		in.Manual = pg.Mask.Manual
		in.Off = pg.Mask.OffKeys()
		in.Checked = pg.Mask.Status == domain.MaskChecked
	}
	res := domain.MaskPage(in)
	// A checked page whose mask changes, after new rules or a new person,
	// must be checked again.
	if in.Checked && (res.Text != pg.Mask.Text || !slices.Equal(res.Boxes, pg.Mask.Boxes)) {
		res.Status = domain.MaskNeedsCheck
	}
	m := domain.PageMask{Version: domain.MaskRulesVersion, Status: res.Status, Reasons: res.Reasons, Boxes: res.Boxes, Auto: res.Auto, Manual: in.Manual,
		Text: res.Text}
	for _, masked := range res.Masked {
		if masked {
			m.MaskedLines++
		}
	}
	var image []byte
	if pg.HasImage {
		raw, err := p.readBytes(ctx, pg.ImageSHA)
		if err != nil {
			return m, err
		}
		if image, err = p.Masker.Mask(raw, res.Boxes); err != nil {
			return m, err
		}
		sum := sha256.Sum256(image)
		m.ImageSHA = hex.EncodeToString(sum[:])
	}
	p.FileLock.Lock()
	defer p.FileLock.Unlock()
	if image != nil {
		if err := p.Files.Put(ctx, m.ImageSHA, image); err != nil {
			return m, err
		}
	}
	prev, err := p.Store.SetPageMask(ctx, tenantID, pg.ID, m)
	if err != nil {
		return m, errors.Join(err, dropUnused(ctx, p.Deps, []string{m.ImageSHA}))
	}
	return m, dropUnused(ctx, p.Deps, []string{prev})
}

func (p *Pipeline) readBytes(ctx context.Context, sha string) ([]byte, error) {
	rc, err := p.Files.Open(ctx, sha)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read stored image: %w", err)
	}
	return data, nil
}

// runRead masks an item's pages and, when every page may be sent, asks the
// AI provider to read them. The answer becomes drafts (HW-READ-4).
func (p *Pipeline) runRead(ctx context.Context, job domain.Job) error {
	reader := p.reader()
	tenantID, itemID := job.TenantID, job.SubjectID
	item, err := p.Store.InboxItem(ctx, tenantID, itemID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.Status != domain.InboxNew {
		return nil
	}
	rd, err := p.Store.Reading(ctx, tenantID, itemID)
	if errors.Is(err, domain.ErrNotFound) {
		rd = domain.Reading{ItemID: itemID}
	} else if err != nil {
		return err
	}
	if rd.Status == domain.ReadingSkipped {
		return nil
	}
	save := func(status domain.ReadingStatus, reason string) error {
		rd.Status, rd.Reason, rd.Error, rd.UpdatedAt = status, reason, "", p.Clock()
		return ignoreGone(p.Store.SaveReading(ctx, tenantID, rd))
	}
	// An admin turned AI reading off after this job was queued.
	if reader == nil {
		return save(domain.ReadingOff, "")
	}

	pages, ready, err := p.itemPages(ctx, tenantID, item)
	if err != nil {
		return err
	}
	if !ready {
		return save(domain.ReadingWaiting, "")
	}
	chars := 0
	texts := make([]string, len(pages))
	for i, ip := range pages {
		texts[i] = ip.page.Text
		chars += utf8.RuneCountInString(strings.Join(strings.Fields(ip.page.Text), ""))
	}
	if chars < domain.MinReadableChars {
		return save(domain.ReadingSkipped, domain.SkipNoText)
	}

	people, err := p.Store.People(ctx, tenantID)
	if err != nil {
		return err
	}
	names := peopleNames(people)
	personID := item.PersonID
	if personID == "" {
		personID = domain.MatchPerson(texts, people)
	}

	send := pages[:min(len(pages), domain.MaxReadPages)]
	masks := make([]domain.PageMask, len(send))
	var reasons []string
	for i, ip := range send {
		m, err := p.maskPage(ctx, tenantID, ip, names)
		if err != nil {
			return err
		}
		masks[i] = m
		if m.Status == domain.MaskNeedsCheck || !ip.page.HasImage {
			for _, r := range m.Reasons {
				reasons = append(reasons, fmt.Sprintf("Page %d: %s", i+1, r))
			}
		}
	}
	if len(reasons) > 0 {
		return save(domain.ReadingNeedsCheck, strings.Join(reasons, "\n"))
	}

	readPages := make([]port.ReadPage, len(send))
	for i := range send {
		img, err := p.readBytes(ctx, masks[i].ImageSHA)
		if err != nil {
			return err
		}
		readPages[i] = port.ReadPage{Number: i + 1, JPEG: img, Text: masks[i].Text}
	}
	res, err := reader.Read(ctx, readPages)
	if err != nil {
		return err
	}
	return p.storeReading(ctx, tenantID, item, rd, send, personID, res, len(pages) > len(send))
}

// storeReading turns the provider's answer into the item's proposal and
// drafts, replacing any earlier drafts, unless the item was filed or
// discarded while the provider was reading.
func (p *Pipeline) storeReading(ctx context.Context, tenantID string, item domain.InboxItem, rd domain.Reading, sent []itemPage,
	personID string, res port.ReadResult, truncated bool) error {
	today := p.today()
	proposal := res.Output.Proposal(today)
	proposal.PersonID = personID
	obs, obsPages, obsConf := res.Output.CleanObservations(len(sent), today, p.Catalog)
	finds, findPages, findConf := res.Output.CleanFindings(len(sent))
	now := p.Clock()
	return p.Store.InTx(ctx, func(r port.Repos) error {
		current, err := r.InboxItem(ctx, tenantID, item.ID)
		if err != nil {
			return ignoreGone(err)
		}
		if current.Status != domain.InboxNew {
			return nil
		}
		links, err := providerLinks(ctx, r, tenantID, proposal.ProviderName)
		if err != nil {
			return err
		}
		if err := r.DeleteItemDrafts(ctx, tenantID, item.ID); err != nil {
			return err
		}
		for i, in := range obs {
			if code, ok := links[domain.FoldKey(in.Analyte)]; ok {
				in.MetricCode = code
			}
			o := observationFrom(in, now)
			o.ID, o.Status, o.ItemID, o.PageID, o.Confidence = NewID(), domain.StatusDraft, item.ID, sent[obsPages[i]-1].page.ID, obsConf[i]
			if err := r.CreateObservation(ctx, tenantID, o); err != nil {
				return err
			}
		}
		for i, in := range finds {
			f := findingFrom(in, now)
			f.ID, f.Status, f.ItemID, f.PageID, f.Confidence = NewID(), domain.StatusDraft, item.ID, sent[findPages[i]-1].page.ID, findConf[i]
			if err := r.CreateFinding(ctx, tenantID, f); err != nil {
				return err
			}
		}
		rd.Status, rd.Model, rd.Proposal, rd.PagesSent, rd.Error, rd.UpdatedAt = domain.ReadingDone, res.Model, &proposal, len(sent), "", now
		rd.Reason = ""
		if truncated {
			rd.Reason = fmt.Sprintf("Only the first %d pages were read.", domain.MaxReadPages)
		}
		return r.SaveReading(ctx, tenantID, rd)
	})
}

// providerLinks returns the analyte links remembered for a provider name,
// or the links for papers without a provider (HW-READ-5).
func providerLinks(ctx context.Context, r port.Repos, tenantID, providerName string) (map[string]string, error) {
	providerID := ""
	if key := domain.FoldKey(providerName); key != "" {
		pr, err := r.ProviderByKey(ctx, tenantID, key)
		switch {
		case err == nil:
			providerID = pr.ID
		case !errors.Is(err, domain.ErrNotFound):
			return nil, err
		}
	}
	return r.AnalyteLinks(ctx, tenantID, providerID)
}

// observationFrom builds an observation from a validated input, parsing
// the printed value and range.
func observationFrom(in domain.ObservationInput, now time.Time) domain.Observation {
	low, high := domain.ParseRange(in.Reference)
	return domain.Observation{Analyte: in.Analyte, MetricCode: in.MetricCode, Value: in.Value, ValueNum: domain.ParseNumber(in.Value),
		Unit: in.Unit, RefText: in.Reference, RefLow: low, RefHigh: high, Flag: in.Flag, SampleDate: in.SampleDate, PageID: in.PageID,
		CreatedAt: now, UpdatedAt: now}
}

func findingFrom(in domain.FindingInput, now time.Time) domain.Finding {
	return domain.Finding{Kind: domain.FindingKind(in.Kind), Text: in.Text, ICD10: in.ICD10, Drug: in.Drug, Dose: in.Dose,
		Frequency: in.Frequency, Duration: in.Duration, PageID: in.PageID, CreatedAt: now, UpdatedAt: now}
}
