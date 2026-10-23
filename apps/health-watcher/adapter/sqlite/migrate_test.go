package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"apps/health-watcher/internal/sqlitefile"
)

// migrateTo applies the migrations up to and including version.
func migrateTo(t *testing.T, db *sql.DB, version int) {
	t.Helper()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.version > version {
			break
		}
		if _, err := db.Exec(m.sql); err != nil {
			t.Fatalf("migration %s: %v", m.name, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations VALUES (?, ?, '2026-10-21T00:00:00.000000000Z')`, m.version, m.name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGivenRecordsFromBeforeTheFHIRModelWhenMigratedThenSourceAndVersionAreBackfilled(t *testing.T) {
	db, err := sqlitefile.Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migrateTo(t, db, 9)
	const at = "2026-10-21T00:00:00.000000000Z"
	for _, q := range []string{
		`INSERT INTO users (id, email, display_name, password_hash, created_at) VALUES ('u1', 'a@example.test', 'A', 'x', '` + at + `')`,
		`INSERT INTO tenants (id, name, created_at) VALUES ('t1', 'Family', '` + at + `')`,
		`INSERT INTO inbox_items (id, tenant_id, status, source, uploaded_by, created_at) VALUES ('i1', 't1', 'filed', 'browser', 'u1', '` + at + `')`,
		`INSERT INTO readings (item_id, tenant_id, status, model, updated_at) VALUES ('i1', 't1', 'done', 'model-a', '` + at + `')`,
		// Read by AI, then edited twice after approval.
		`INSERT INTO observations (id, tenant_id, status, item_id, analyte, value, confidence, created_at, updated_at)
			VALUES ('o1', 't1', 'approved', 'i1', 'Glucose', '5.9', 100, '` + at + `', '` + at + `')`,
		`INSERT INTO audit_log (id, tenant_id, entity, entity_id, action, user_id, at, old_json)
			VALUES ('a2', 't1', 'observation', 'o1', 'update', 'u1', '2026-10-21T02:00:00.000000000Z', '{}')`,
		`INSERT INTO audit_log (id, tenant_id, entity, entity_id, action, user_id, at, old_json)
			VALUES ('a1', 't1', 'observation', 'o1', 'update', 'u1', '2026-10-21T01:00:00.000000000Z', '{}')`,
		// Typed by hand.
		`INSERT INTO observations (id, tenant_id, status, analyte, value, confidence, created_at, updated_at)
			VALUES ('o2', 't1', 'approved', 'HbA1c', '5.6', 100, '` + at + `', '` + at + `')`,
		`INSERT INTO findings (id, tenant_id, status, item_id, kind, text, confidence, created_at, updated_at)
			VALUES ('f1', 't1', 'approved', 'i1', 'diagnosis', 'Viêm họng', 80, '` + at + `', '` + at + `')`,
		`INSERT INTO findings (id, tenant_id, status, kind, text, confidence, created_at, updated_at)
			VALUES ('f2', 't1', 'approved', 'note', 'Tái khám', 100, '` + at + `', '` + at + `')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}

	if err := migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	type row struct {
		source, detail string
		version        int
	}
	read := func(table, id string) row {
		var r row
		if err := db.QueryRow(`SELECT source, source_detail, version FROM `+table+` WHERE id = ?`, id).Scan(&r.source, &r.detail, &r.version); err != nil {
			t.Fatal(err)
		}
		return r
	}
	for _, tc := range []struct {
		table, id string
		want      row
	}{
		{"observations", "o1", row{"reading", "model-a", 3}},
		{"observations", "o2", row{"manual", "", 1}},
		{"findings", "f1", row{"reading", "model-a", 1}},
		{"findings", "f2", row{"manual", "", 1}},
	} {
		if got := read(tc.table, tc.id); got != tc.want {
			t.Errorf("%s %s = %+v; want %+v", tc.table, tc.id, got, tc.want)
		}
	}
	for id, want := range map[string]int{"a1": 1, "a2": 2} {
		var v int
		if err := db.QueryRow(`SELECT version FROM audit_log WHERE id = ?`, id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != want {
			t.Errorf("audit %s version = %d; want %d", id, v, want)
		}
	}
}
