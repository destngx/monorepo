package service

import (
	"context"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// catalogVersionKey is the meta key of the catalog version the stored
// metric links were last matched with.
const catalogVersionKey = "catalog_version"

// RelinkCatalog matches approved values again when the catalog changed
// since the last start, so a fix to the catalog also fixes the values read
// before it (HW-READ-5). Only values no person changed are matched again,
// including ones an earlier relink changed: a value a person edited, or an
// analyte name a person linked by hand for that provider or for every
// provider, keeps its link. Values match the catalog with the tenant's
// custom metrics. Each change is audited with no user, as the engine's own. It returns how many values changed.
func (c *Clinical) RelinkCatalog(ctx context.Context) (int, error) {
	changed := 0
	err := c.Store.InTx(ctx, func(r port.Repos) error {
		seen, err := r.Meta(ctx, catalogVersionKey)
		if err != nil || seen == c.Catalog.Version {
			return err
		}
		obs, err := r.UneditedReadObservations(ctx)
		if err != nil {
			return err
		}
		links := map[[2]string]map[string]string{}
		catalogs := map[string]*domain.Catalog{}
		for _, to := range obs {
			key := [2]string{to.TenantID, to.ProviderID}
			if links[key] == nil {
				if links[key], err = analyteLinks(ctx, r, to.TenantID, to.ProviderID); err != nil {
					return err
				}
			}
			if _, ok := links[key][domain.AnalyteKey(to.Analyte)]; ok {
				continue
			}
			catalog := catalogs[to.TenantID]
			if catalog == nil {
				if catalog, err = tenantCatalog(ctx, r, c.Catalog, to.TenantID); err != nil {
					return err
				}
				catalogs[to.TenantID] = catalog
			}
			code := ""
			if m, ok := catalog.Match(to.Analyte, to.Unit, to.Value); ok {
				code = m.Code
			}
			if code == to.MetricCode {
				continue
			}
			sc := domain.Scope{TenantID: to.TenantID}
			if err := c.audit(ctx, r, sc, domain.AuditObservation, to.ID, "update", to.Version, to.Observation); err != nil {
				return err
			}
			o := to.Observation
			o.MetricCode, o.Provenance, o.UpdatedAt = code, nextVersion(o.Provenance, o.Status), c.Clock()
			if err := r.UpdateObservation(ctx, to.TenantID, o); err != nil {
				return err
			}
			changed++
		}
		return r.SetMeta(ctx, catalogVersionKey, c.Catalog.Version)
	})
	return changed, err
}
