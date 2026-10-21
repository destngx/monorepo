package domain

import "time"

// JobKind is a kind of background work (HW-READ-1..3).
type JobKind string

const (
	// JobOCR reads one file's text and renders its pages; the subject is a file id.
	JobOCR JobKind = "ocr"
	// JobRead masks an inbox item's pages and sends them to the AI provider;
	// the subject is an inbox item id.
	JobRead JobKind = "read"
)

// JobStatus is where a job is in its life.
type JobStatus string

const (
	JobQueued  JobStatus = "queued"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

// Job is one unit of background work, kept in the database so it survives
// restarts.
type Job struct {
	ID            string
	TenantID      string
	Kind          JobKind
	SubjectID     string
	Status        JobStatus
	Attempts      int
	NextAttemptAt time.Time
	LastError     string
}

// MaxJobAttempts is how many times a job runs before it is marked failed.
const MaxJobAttempts = 5

// jobBackoff is the wait after each failed attempt (HW-READ-6).
var jobBackoff = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 6 * time.Hour}

// RetryAfter returns the wait before the next attempt once `attempts`
// attempts have failed, and false when the job should be marked failed.
func RetryAfter(attempts int) (time.Duration, bool) {
	if attempts < 1 || attempts >= MaxJobAttempts {
		return 0, false
	}
	return jobBackoff[min(attempts, len(jobBackoff))-1], true
}
