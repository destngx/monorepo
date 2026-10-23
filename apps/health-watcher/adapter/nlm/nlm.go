// Package nlm looks up LOINC codes in the NLM Clinical Tables service and
// their descriptions in MedlinePlus Connect (HW-MET-8). Only the code is
// sent.
package nlm

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

const (
	tablesURL  = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"
	connectURL = "https://connect.medlineplus.gov/service"
	// maxQuote bounds a description; it is cut at a sentence end.
	maxQuote = 350
)

// Client calls the NLM services.
type Client struct {
	http       *http.Client
	tablesURL  string
	connectURL string
	now        func() time.Time
}

var _ port.LOINC = (*Client)(nil)

// New returns a client; httpClient may be nil.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{http: httpClient, tablesURL: tablesURL, connectURL: connectURL, now: time.Now}
}

// Name returns the LOINC long common name of code, false when the code is
// not a LOINC code.
func (c *Client) Name(ctx context.Context, code string) (string, bool, error) {
	q := url.Values{}
	q.Set("terms", code)
	q.Set("sf", "LOINC_NUM")
	q.Set("df", "LOINC_NUM,LONG_COMMON_NAME")
	q.Set("maxList", "5")
	var body []json.RawMessage
	if err := c.get(ctx, c.tablesURL+"?"+q.Encode(), &body); err != nil {
		return "", false, fmt.Errorf("look up LOINC %s: %w", code, err)
	}
	if len(body) < 4 {
		return "", false, fmt.Errorf("look up LOINC %s: unexpected answer", code)
	}
	var rows [][]string
	if err := json.Unmarshal(body[3], &rows); err != nil {
		return "", false, fmt.Errorf("decode LOINC %s: %w", code, err)
	}
	for _, r := range rows {
		if len(r) == 2 && r[0] == code {
			return r[1], true, nil
		}
	}
	return "", false, nil
}

// About returns the opening sentences of the first MedlinePlus page about
// code, word for word, false when MedlinePlus has none.
func (c *Client) About(ctx context.Context, code string) (domain.MetricAbout, bool, error) {
	q := url.Values{}
	q.Set("mainSearchCriteria.v.cs", "2.16.840.1.113883.6.1")
	q.Set("mainSearchCriteria.v.c", code)
	q.Set("knowledgeResponseType", "application/json")
	var body struct {
		Feed struct {
			Entry []struct {
				Title struct {
					Value string `json:"_value"`
				} `json:"title"`
				Link []struct {
					Href string `json:"href"`
				} `json:"link"`
				Summary struct {
					Value string `json:"_value"`
				} `json:"summary"`
			} `json:"entry"`
		} `json:"feed"`
	}
	if err := c.get(ctx, c.connectURL+"?"+q.Encode(), &body); err != nil {
		return domain.MetricAbout{}, false, fmt.Errorf("describe LOINC %s: %w", code, err)
	}
	for _, e := range body.Feed.Entry {
		if len(e.Link) == 0 {
			continue
		}
		quote := opening(e.Summary.Value)
		if quote == "" {
			continue
		}
		href, _, _ := strings.Cut(e.Link[0].Href, "?")
		return domain.MetricAbout{Quote: quote, Source: domain.AboutSource, Title: e.Title.Value, URL: href,
			Fetched: c.now().Format(time.DateOnly)}, true, nil
	}
	return domain.MetricAbout{}, false, nil
}

func (c *Client) get(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var (
	paragraph = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	anyTag    = regexp.MustCompile(`<[^>]*>`)
	sentence  = regexp.MustCompile(`[.!?](\s|$)`)
)

// opening is the first paragraph of a summary as plain text, cut after the
// last whole sentence that fits in maxQuote; "" when even the first
// sentence is longer.
func opening(summary string) string {
	m := paragraph.FindStringSubmatch(summary)
	if m == nil {
		return ""
	}
	text := strings.Join(strings.Fields(html.UnescapeString(anyTag.ReplaceAllString(m[1], ""))), " ")
	if len(text) <= maxQuote {
		return text
	}
	end := 0
	for _, loc := range sentence.FindAllStringIndex(text, -1) {
		if loc[0]+1 > maxQuote {
			break
		}
		end = loc[0] + 1
	}
	return text[:end]
}
