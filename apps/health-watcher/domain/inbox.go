package domain

import (
	"bytes"
	"net/http"
	"time"
)

// Inbox limits (HW-IMP-2). The engine reads overrides from settings.
const (
	DefaultMaxFileMB  = 50
	DefaultMaxBatchMB = 200
	MaxFilesPerItem   = 60
	MaxNoteLength     = 2000
)

// InboxStatus is where an inbox item is in its life.
type InboxStatus string

const (
	InboxNew       InboxStatus = "new"
	InboxFiled     InboxStatus = "filed"
	InboxDiscarded InboxStatus = "discarded"
)

// InboxSource says how an item arrived.
type InboxSource string

const (
	SourceWeb      InboxSource = "web"
	SourceShortcut InboxSource = "shortcut"
	// SourcePhone items come from the phone capture page opened by a QR
	// code on the Mac (HW-IMP-9).
	SourcePhone InboxSource = "phone"
)

// InboxItem holds files waiting to be filed under a visit (HW-IMP-1).
type InboxItem struct {
	ID         string       `json:"id"`
	Status     InboxStatus  `json:"status"`
	Source     InboxSource  `json:"source"`
	Note       string       `json:"note"`
	UploadedBy string       `json:"uploadedBy"`
	Files      []StoredFile `json:"files"`
	DocumentID string       `json:"documentId,omitempty"`
	// PersonID is the person picked for a backlog upload (HW-IMP-6).
	PersonID  string       `json:"personId,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	OCR       *OCRProgress `json:"ocr,omitempty"`
	Reading   *Reading     `json:"reading,omitempty"`
}

// OCRProgress counts an item's files by where their text reading is.
type OCRProgress struct {
	Total  int    `json:"total"`
	Done   int    `json:"done"`
	Failed int    `json:"failed"`
	Error  string `json:"error,omitempty"`
}

// Upload is one file as received, before it is stored.
type Upload struct {
	Name string
	Data []byte
}

// UploadResult reports one received file: stored, or a duplicate of a file
// the tenant already has.
type UploadResult struct {
	Name      string     `json:"name"`
	File      StoredFile `json:"file"`
	Duplicate bool       `json:"duplicate"`
}

// SniffMIME returns the type of an accepted file from its bytes, or a
// validation error (HW-IMP-2). The name is never trusted.
func SniffMIME(data []byte) (string, error) {
	switch {
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return "application/pdf", nil
	case isHEIC(data):
		return "image/heic", nil
	}
	switch mime := http.DetectContentType(data); mime {
	case "image/jpeg", "image/png", "image/webp":
		return mime, nil
	}
	return "", Invalid("only PDF, JPEG, PNG, WebP and HEIC files are accepted")
}

// isHEIC matches an ISO media file whose major or compatible brand is HEIF.
func isHEIC(data []byte) bool {
	if len(data) < 16 || !bytes.Equal(data[4:8], []byte("ftyp")) {
		return false
	}
	brands := data[8:min(len(data), 64)]
	for _, b := range []string{"heic", "heix", "heim", "heis", "mif1", "msf1"} {
		if bytes.Contains(brands, []byte(b)) {
			return true
		}
	}
	return false
}

// FileInput files an inbox item as a document under a visit: an existing
// one (VisitID) or a new one (Visit).
type FileInput struct {
	VisitID string      `json:"visitId"`
	Visit   *VisitInput `json:"visit"`
	Kind    string      `json:"kind"`
	Title   string      `json:"title"`
}
