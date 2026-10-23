package service

import (
	"context"
	"errors"
	"slices"
	"strings"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// customMetrics is the part of the store a tenant's catalog is built from.
type customMetrics interface {
	CustomMetrics(ctx context.Context, tenantID string) ([]domain.Metric, error)
}

// tenantCatalog is base with the custom metrics the tenant approved
// (HW-MET-8).
func tenantCatalog(ctx context.Context, r customMetrics, base *domain.Catalog, tenantID string) (*domain.Catalog, error) {
	custom, err := r.CustomMetrics(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return base.With(custom), nil
}

func (c *Clinical) catalogFor(ctx context.Context, tenantID string) (*domain.Catalog, error) {
	return tenantCatalog(ctx, c.Store, c.Catalog, tenantID)
}

// analyteLinks returns the remembered links for papers from providerID,
// by analyte key: the links approved for every provider, overridden by
// the provider's own (HW-READ-5, HW-MET-8).
func analyteLinks(ctx context.Context, r port.Repos, tenantID, providerID string) (map[string]string, error) {
	links, err := r.AnalyteLinks(ctx, tenantID, domain.AnyProvider)
	if err != nil {
		return nil, err
	}
	own, err := r.AnalyteLinks(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	for k, v := range own {
		links[k] = v
	}
	return links, nil
}

// linkKeysKey is the meta key set once the stored analyte links are keyed
// by domain.AnalyteKey.
const linkKeysKey = "analyte_link_keys"

// RekeyAnalyteLinks moves the stored analyte links from the old key, which
// dropped "#" and "%" and so gave "GRAN#" and "GRAN%" one link, to
// domain.AnalyteKey. Each link takes the keys of the names its metric's
// values print; a link no value explains keeps its old key and so stops
// matching. It runs once.
func (c *Clinical) RekeyAnalyteLinks(ctx context.Context) error {
	return c.Store.InTx(ctx, func(r port.Repos) error {
		done, err := r.Meta(ctx, linkKeysKey)
		if err != nil || done != "" {
			return err
		}
		links, err := r.AllAnalyteLinks(ctx)
		if err != nil {
			return err
		}
		for _, l := range links {
			names, err := r.LinkedAnalytes(ctx, l.TenantID, l.MetricCode)
			if err != nil {
				return err
			}
			keys := map[string]bool{}
			for _, n := range names {
				if domain.FoldKey(n) == l.Key {
					keys[domain.AnalyteKey(n)] = true
				}
			}
			if len(keys) == 0 || (len(keys) == 1 && keys[l.Key]) {
				continue
			}
			for k := range keys {
				if err := r.SetAnalyteLink(ctx, l.TenantID, l.ProviderID, k, l.MetricCode); err != nil {
					return err
				}
			}
			if !keys[l.Key] {
				if err := r.DeleteAnalyteLink(ctx, l.TenantID, l.ProviderID, l.Key); err != nil {
					return err
				}
			}
		}
		return r.SetMeta(ctx, linkKeysKey, "analyte")
	})
}

// scheduleSuggest queues a suggestion job for a tenant when AI reading is
// on; the job does nothing when no name is new.
func (p *Pipeline) scheduleSuggest(ctx context.Context, r port.Repos, tenantID string) error {
	if p == nil || p.reader() == nil {
		return nil
	}
	if err := r.EnqueueJob(ctx, domain.Job{ID: NewID(), TenantID: tenantID, Kind: domain.JobSuggest, SubjectID: tenantID}, p.Clock()); err != nil {
		return err
	}
	p.Wake()
	return nil
}

// runSuggest asks the AI what each printed name no metric knows is, by
// name and unit alone, and stores one suggestion per name for a person to
// approve (HW-MET-8). A new metric's LOINC code is kept only when the NLM
// service knows it, and its description is fetched by that code.
func (p *Pipeline) runSuggest(ctx context.Context, job domain.Job) error {
	reader := p.reader()
	if reader == nil {
		return nil
	}
	tenantID := job.TenantID
	catalog, err := tenantCatalog(ctx, p.Store, p.Catalog, tenantID)
	if err != nil {
		return err
	}
	obs, err := p.Store.UnlinkedObservations(ctx, tenantID)
	if err != nil {
		return err
	}
	asked, err := p.Store.SuggestionKeys(ctx, tenantID)
	if err != nil {
		return err
	}
	names := domain.UnknownNames(obs, asked)
	for batch := range slices.Chunk(names, domain.MaxSuggestNames) {
		res, err := reader.SuggestMetrics(ctx, batch, catalog.Metrics())
		if err != nil {
			return err
		}
		suggestions := res.Output.Clean(batch, catalog)
		for i := range suggestions {
			if err := p.checkLOINC(ctx, suggestions[i].Proposal); err != nil {
				return err
			}
		}
		now := p.Clock()
		err = p.Store.InTx(ctx, func(r port.Repos) error {
			for _, s := range suggestions {
				s.ID, s.Model, s.CreatedAt = NewID(), res.Model, now
				if err := r.CreateSuggestion(ctx, tenantID, s); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// checkLOINC keeps a proposal's LOINC code only when the NLM service knows
// it, then fetches the description by that code.
func (p *Pipeline) checkLOINC(ctx context.Context, prop *domain.MetricProposal) error {
	if prop == nil || prop.LOINC == "" {
		return nil
	}
	if p.LOINC == nil {
		prop.LOINC = ""
		return nil
	}
	name, ok, err := p.LOINC.Name(ctx, prop.LOINC)
	if err != nil {
		return err
	}
	if !ok {
		prop.LOINC = ""
		return nil
	}
	prop.LOINCName = name
	about, ok, err := p.LOINC.About(ctx, prop.LOINC)
	if err != nil {
		return err
	}
	if ok {
		prop.About = &about
	}
	return nil
}

// Suggestions lists the pending metric suggestions, each with how many
// values it would link (HW-MET-8).
func (c *Clinical) Suggestions(ctx context.Context, sc domain.Scope) ([]domain.MetricSuggestion, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	list, err := c.Store.Suggestions(ctx, sc.TenantID, domain.SuggestionPending)
	if err != nil {
		return nil, err
	}
	obs, err := c.Store.UnlinkedObservations(ctx, sc.TenantID)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, o := range obs {
		counts[domain.AnalyteKey(o.Analyte)]++
	}
	for i := range list {
		list[i].Values = counts[list[i].AnalyteKey]
	}
	return list, nil
}

// pendingSuggestions counts the pending suggestions.
func (c *Clinical) pendingSuggestions(ctx context.Context, tenantID string) (int, error) {
	list, err := c.Store.Suggestions(ctx, tenantID, domain.SuggestionPending)
	return len(list), err
}

// SuggestionEdit is a person's change to a suggested new metric before
// approving it; empty fields keep the suggestion's.
type SuggestionEdit struct {
	Name     string                `json:"name"`
	NameVI   string                `json:"nameVi"`
	Unit     *string               `json:"unit"`
	Category domain.MetricCategory `json:"category"`
}

// ApproveSuggestion applies a pending suggestion: a new metric is added to
// the tenant's catalog under the printed name, or the printed name is
// linked to the suggested metric for papers from every provider. The
// approved values with that name that no metric holds are linked now,
// each audited as the person's change.
func (c *Clinical) ApproveSuggestion(ctx context.Context, sc domain.Scope, id string, edit SuggestionEdit) (domain.MetricSuggestion, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.MetricSuggestion{}, err
	}
	catalog, err := c.catalogFor(ctx, sc.TenantID)
	if err != nil {
		return domain.MetricSuggestion{}, err
	}
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		s, err := r.Suggestion(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if s.Status != domain.SuggestionPending {
			return domain.Invalid("this suggestion was already %s", s.Status)
		}
		now := c.Clock()
		code := s.MetricCode
		switch s.Kind {
		case domain.SuggestLink:
			if _, ok := catalog.Metric(code); !ok {
				return domain.Invalid("unknown metric %q", code)
			}
		case domain.SuggestNew:
			if s.Proposal == nil {
				return errNoProposal
			}
			m, err := newMetric(*s.Proposal, edit, s.Analyte)
			if err != nil {
				return err
			}
			code = m.Code
			if _, ok := catalog.Metric(code); !ok {
				if err := r.CreateCustomMetric(ctx, sc.TenantID, m, sc.UserID, now); err != nil {
					return err
				}
			}
		default:
			return domain.Invalid("this suggestion has nothing to approve")
		}
		if err := r.SetAnalyteLink(ctx, sc.TenantID, domain.AnyProvider, s.AnalyteKey, code); err != nil {
			return err
		}
		obs, err := r.UnlinkedObservations(ctx, sc.TenantID)
		if err != nil {
			return err
		}
		for _, o := range obs {
			if domain.AnalyteKey(o.Analyte) != s.AnalyteKey {
				continue
			}
			if err := c.audit(ctx, r, sc, domain.AuditObservation, o.ID, "update", o.Version, o); err != nil {
				return err
			}
			o.MetricCode, o.Provenance, o.UpdatedAt = code, nextVersion(o.Provenance, o.Status), now
			if err := r.UpdateObservation(ctx, sc.TenantID, o); err != nil {
				return err
			}
		}
		return r.DecideSuggestion(ctx, sc.TenantID, id, domain.SuggestionApproved, sc.UserID, now)
	})
	if err != nil {
		return domain.MetricSuggestion{}, err
	}
	return c.Store.Suggestion(ctx, sc.TenantID, id)
}

// newMetric is the custom metric a suggestion proposes, with the person's
// edits. The printed name is its first alias, so later papers that print
// it match it.
func newMetric(p domain.MetricProposal, edit SuggestionEdit, analyte string) (domain.Metric, error) {
	if name := strings.TrimSpace(edit.Name); name != "" {
		p.Name = name
	}
	if name := strings.TrimSpace(edit.NameVI); name != "" {
		p.NameVI = name
	}
	if edit.Unit != nil {
		p.Unit = domain.NormalizeUnit(strings.TrimSpace(*edit.Unit))
	}
	if edit.Category != "" {
		p.Category = edit.Category
	}
	if !slices.Contains(domain.MetricCategories, p.Category) {
		return domain.Metric{}, domain.Invalid("unknown category %q", p.Category)
	}
	if len(p.Name) > 80 || len(p.NameVI) > 80 || len(p.Unit) > 30 {
		return domain.Metric{}, domain.Invalid("a name is at most 80 characters and a unit 30")
	}
	return domain.Metric{Code: domain.CustomMetricCode(p.Name), LOINC: p.LOINC, Name: p.Name, NameVI: p.NameVI, Category: p.Category,
		Unit: p.Unit, Aliases: []string{analyte}, Custom: true, About: p.About}, nil
}

// RejectSuggestion drops a pending suggestion; the name is not suggested
// again and its values stay under their printed name (HW-MET-7).
func (c *Clinical) RejectSuggestion(ctx context.Context, sc domain.Scope, id string) (domain.MetricSuggestion, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.MetricSuggestion{}, err
	}
	err := c.Store.InTx(ctx, func(r port.Repos) error {
		s, err := r.Suggestion(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if s.Status != domain.SuggestionPending {
			return domain.Invalid("this suggestion was already %s", s.Status)
		}
		return r.DecideSuggestion(ctx, sc.TenantID, id, domain.SuggestionRejected, sc.UserID, c.Clock())
	})
	if err != nil {
		return domain.MetricSuggestion{}, err
	}
	return c.Store.Suggestion(ctx, sc.TenantID, id)
}

var errNoProposal = errors.New("a new-metric suggestion has no proposal")
