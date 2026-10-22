package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Records manages people, providers, visits, documents and file access
// (HW-TEN-3, HW-REC-1..2, HW-REC-5, HW-FIND-1, HW-FIND-3).
type Records struct {
	Deps
}

// People lists the tenant's people.
func (s *Records) People(ctx context.Context, sc domain.Scope) ([]domain.Person, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	return s.Store.People(ctx, sc.TenantID)
}

// Person returns one person.
func (s *Records) Person(ctx context.Context, sc domain.Scope, id string) (domain.Person, error) {
	if err := sc.RequireRead(); err != nil {
		return domain.Person{}, err
	}
	return s.Store.Person(ctx, sc.TenantID, id)
}

func (s *Records) checkPersonInput(ctx context.Context, r port.Repos, sc domain.Scope, in domain.PersonInput) (domain.PersonInput, error) {
	in, err := in.Validate(s.today())
	if err != nil {
		return in, err
	}
	if in.UserID != "" {
		if _, err := r.RoleOf(ctx, sc.TenantID, in.UserID); errors.Is(err, domain.ErrNotFound) {
			return in, domain.Invalid("the linked user is not a member of this tenant")
		} else if err != nil {
			return in, err
		}
	}
	return in, nil
}

// CreatePerson adds a person. A linked user must be a member.
func (s *Records) CreatePerson(ctx context.Context, sc domain.Scope, in domain.PersonInput) (domain.Person, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Person{}, err
	}
	in, err := s.checkPersonInput(ctx, s.Store, sc, in)
	if err != nil {
		return domain.Person{}, err
	}
	p := domain.Person{ID: NewID(), DisplayName: in.DisplayName, BirthDate: in.BirthDate, Sex: domain.Sex(in.Sex),
		UserID: in.UserID, CreatedAt: s.Clock()}
	if err := s.Store.CreatePerson(ctx, sc.TenantID, p); err != nil {
		return domain.Person{}, err
	}
	return s.Store.Person(ctx, sc.TenantID, p.ID)
}

// UpdatePerson changes a person.
func (s *Records) UpdatePerson(ctx context.Context, sc domain.Scope, id string, in domain.PersonInput) (domain.Person, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Person{}, err
	}
	in, err := s.checkPersonInput(ctx, s.Store, sc, in)
	if err != nil {
		return domain.Person{}, err
	}
	if err := s.Store.UpdatePerson(ctx, sc.TenantID, id, in); err != nil {
		return domain.Person{}, err
	}
	return s.Store.Person(ctx, sc.TenantID, id)
}

// DeletePerson removes a person with no visits.
func (s *Records) DeletePerson(ctx context.Context, sc domain.Scope, id string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	return s.Store.InTx(ctx, func(r port.Repos) error {
		p, err := r.Person(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if p.VisitCount > 0 {
			return domain.Conflict("%s has %d visits; delete or move them first", p.DisplayName, p.VisitCount)
		}
		typed, err := r.CountTypedObservations(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if typed > 0 {
			return domain.Conflict("%s has %d typed-in measurements; delete them first", p.DisplayName, typed)
		}
		return r.DeletePerson(ctx, sc.TenantID, id)
	})
}

// Providers lists the tenant's providers.
func (s *Records) Providers(ctx context.Context, sc domain.Scope) ([]domain.Provider, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	return s.Store.Providers(ctx, sc.TenantID)
}

// provider finds a provider by folded name or creates it (HW-REC-5).
func provider(ctx context.Context, r port.Repos, tenantID, name, kind string) (*domain.Provider, error) {
	if name == "" {
		return nil, nil
	}
	key := domain.FoldKey(name)
	if key == "" {
		return nil, domain.Invalid("provider name needs letters or digits")
	}
	p, err := r.ProviderByKey(ctx, tenantID, key)
	if err == nil {
		return &p, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	p = domain.Provider{ID: NewID(), Name: name, Kind: domain.ProviderKind(kind)}
	if err := r.CreateProvider(ctx, tenantID, p, key); err != nil {
		return nil, err
	}
	return &p, nil
}

// Visits lists a person's visits, newest first (HW-FIND-1).
func (s *Records) Visits(ctx context.Context, sc domain.Scope, personID string, f port.VisitFilter) ([]domain.Visit, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	if _, err := s.Store.Person(ctx, sc.TenantID, personID); err != nil {
		return nil, err
	}
	for _, d := range []*string{&f.From, &f.To} {
		if *d != "" {
			v, err := domain.Date(*d)
			if err != nil {
				return nil, err
			}
			*d = v
		}
	}
	if f.Type != "" {
		if _, err := domain.ParseVisitType(f.Type); err != nil {
			return nil, err
		}
	}
	return s.Store.VisitsOf(ctx, sc.TenantID, personID, f)
}

// Visit returns a visit with its documents.
func (s *Records) Visit(ctx context.Context, sc domain.Scope, id string) (domain.Visit, error) {
	if err := sc.RequireRead(); err != nil {
		return domain.Visit{}, err
	}
	v, err := s.Store.Visit(ctx, sc.TenantID, id)
	if err != nil {
		return v, err
	}
	v.Documents, err = s.Store.DocumentsOf(ctx, sc.TenantID, id)
	return v, err
}

// createVisit validates in and stores a new visit inside a transaction.
func (s *Records) createVisit(ctx context.Context, r port.Repos, sc domain.Scope, in domain.VisitInput) (domain.Visit, error) {
	in, err := in.Validate(s.today())
	if err != nil {
		return domain.Visit{}, err
	}
	if _, err := r.Person(ctx, sc.TenantID, in.PersonID); err != nil {
		return domain.Visit{}, err
	}
	prov, err := provider(ctx, r, sc.TenantID, in.ProviderName, in.ProviderKind)
	if err != nil {
		return domain.Visit{}, err
	}
	v := domain.Visit{ID: NewID(), PersonID: in.PersonID, Date: in.Date, Type: domain.VisitType(in.Type), Title: in.Title,
		Note: in.Note, Provider: prov, CreatedAt: s.Clock()}
	if err := r.CreateVisit(ctx, sc.TenantID, v); err != nil {
		return domain.Visit{}, err
	}
	return v, nil
}

// CreateVisit adds a visit, creating its provider when new.
func (s *Records) CreateVisit(ctx context.Context, sc domain.Scope, in domain.VisitInput) (domain.Visit, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Visit{}, err
	}
	var id string
	err := s.Store.InTx(ctx, func(r port.Repos) error {
		v, err := s.createVisit(ctx, r, sc, in)
		id = v.ID
		return err
	})
	if err != nil {
		return domain.Visit{}, err
	}
	return s.Visit(ctx, sc, id)
}

// UpdateVisit changes a visit. Moving it to another person moves its
// documents with it.
func (s *Records) UpdateVisit(ctx context.Context, sc domain.Scope, id string, in domain.VisitInput) (domain.Visit, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Visit{}, err
	}
	in, err := in.Validate(s.today())
	if err != nil {
		return domain.Visit{}, err
	}
	err = s.Store.InTx(ctx, func(r port.Repos) error {
		v, err := r.Visit(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if v.DocumentCount > 0 && v.PersonID != in.PersonID {
			return domain.Conflict("this visit has documents; moving it to another person is not supported yet")
		}
		if _, err := r.Person(ctx, sc.TenantID, in.PersonID); err != nil {
			return err
		}
		prov, err := provider(ctx, r, sc.TenantID, in.ProviderName, in.ProviderKind)
		if err != nil {
			return err
		}
		v.PersonID, v.Date, v.Type, v.Title, v.Note, v.Provider = in.PersonID, in.Date, domain.VisitType(in.Type), in.Title, in.Note, prov
		return r.UpdateVisit(ctx, sc.TenantID, v)
	})
	if err != nil {
		return domain.Visit{}, err
	}
	return s.Visit(ctx, sc, id)
}

// DeleteVisit removes a visit with no documents.
func (s *Records) DeleteVisit(ctx context.Context, sc domain.Scope, id string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	return s.Store.InTx(ctx, func(r port.Repos) error {
		v, err := r.Visit(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if v.DocumentCount > 0 {
			return domain.Conflict("this visit has %d documents; delete them first", v.DocumentCount)
		}
		return r.DeleteVisit(ctx, sc.TenantID, id)
	})
}

// DeleteDocument removes a document and any of its files nothing else uses.
func (s *Records) DeleteDocument(ctx context.Context, sc domain.Scope, id string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	var d domain.Document
	err := s.Store.InTx(ctx, func(r port.Repos) error {
		var err error
		if d, err = r.Document(ctx, sc.TenantID, id); err != nil {
			return err
		}
		// Approved records go with the document, so each is audited (HW-REC-6).
		if err := auditDocumentRecords(ctx, r, s.Clock, sc, id); err != nil {
			return err
		}
		return r.DeleteDocument(ctx, sc.TenantID, id)
	})
	if err != nil {
		return err
	}
	return releaseFiles(ctx, s.Deps, sc.TenantID, d.Files)
}

// OpenFile returns a file and its bytes for a reader in the tenant.
func (s *Records) OpenFile(ctx context.Context, sc domain.Scope, id string) (domain.StoredFile, io.ReadCloser, error) {
	if err := sc.RequireRead(); err != nil {
		return domain.StoredFile{}, nil, err
	}
	f, err := s.Store.File(ctx, sc.TenantID, id)
	if err != nil {
		return f, nil, err
	}
	body, err := s.Files.Open(ctx, f.SHA256)
	if err != nil {
		return f, nil, fmt.Errorf("open file %s: %w", f.ID, err)
	}
	return f, body, nil
}

// releaseFiles deletes each file no document or unfiled inbox item uses,
// and its bytes when no tenant has the same hash.
func releaseFiles(ctx context.Context, d Deps, tenantID string, files []domain.StoredFile) error {
	d.FileLock.Lock()
	defer d.FileLock.Unlock()
	var errs []error
	for _, f := range files {
		inUse, err := d.Store.FileInUse(ctx, tenantID, f.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if inUse {
			continue
		}
		// The file's pages go with it, so their images are released too.
		pageImages, err := d.Store.PageImageSHAs(ctx, tenantID, f.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := d.Store.DeleteFile(ctx, tenantID, f.ID); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := dropUnused(ctx, d, append([]string{f.SHA256}, pageImages...)); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("release files: %w", errors.Join(errs...))
	}
	return nil
}

// dropUnused deletes the bytes of each hash no tenant's file or page uses
// any more. The caller holds FileLock.
func dropUnused(ctx context.Context, d Deps, shas []string) error {
	var errs []error
	seen := map[string]bool{}
	for _, sha := range shas {
		if sha == "" || seen[sha] {
			continue
		}
		seen[sha] = true
		used, err := d.Store.SHAInUse(ctx, sha)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !used {
			if err := d.Files.Delete(ctx, sha); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
