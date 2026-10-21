package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"apps/health-watcher/domain"
)

func (r repos) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (r repos) CreateUser(ctx context.Context, u domain.User, passwordHash string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO users (id, email, display_name, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.DisplayName, passwordHash, u.Admin, formatTime(u.CreatedAt))
	if isUnique(err) {
		return domain.Conflict("an account with this email already exists")
	}
	return exec(res, err, "user")
}

func (r repos) userBy(ctx context.Context, column, value string) (domain.User, string, error) {
	var u domain.User
	var hash, created string
	err := r.q.QueryRowContext(ctx, `SELECT id, email, display_name, password_hash, is_admin, created_at FROM users WHERE `+column+` = ?`, value).
		Scan(&u.ID, &u.Email, &u.DisplayName, &hash, &u.Admin, &created)
	if err := one(err, "user"); err != nil {
		return u, "", err
	}
	if u.CreatedAt, err = parseTime(created); err != nil {
		return u, "", err
	}
	return u, hash, nil
}

func (r repos) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	return r.userBy(ctx, "email", email)
}

func (r repos) UserByID(ctx context.Context, id string) (domain.User, string, error) {
	return r.userBy(ctx, "id", id)
}

func (r repos) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, email, display_name, is_admin, created_at FROM users ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		var u domain.User
		var created string
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Admin, &created); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if u.CreatedAt, err = parseTime(created); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return out, nil
}

func (r repos) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

func (r repos) SetUserAdmin(ctx context.Context, userID string, admin bool) error {
	res, err := r.q.ExecContext(ctx, `UPDATE users SET is_admin = ? WHERE id = ?`, admin, userID)
	return affected(res, err, "user")
}

func (r repos) SetPasswordHash(ctx context.Context, userID, passwordHash string) error {
	res, err := r.q.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return affected(res, err, "user")
}

func (r repos) RecordLoginFailure(ctx context.Context, email string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO login_failures (email, at) VALUES (?, ?)`, email, formatTime(at))
	return exec(res, err, "login failure")
}

func (r repos) CountLoginFailures(ctx context.Context, email string, since time.Time) (int, error) {
	var n int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM login_failures WHERE email = ? AND at >= ?`, email, formatTime(since)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count login failures: %w", err)
	}
	return n, nil
}

func (r repos) ClearLoginFailures(ctx context.Context, email string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM login_failures WHERE email = ?`, email)
	return exec(res, err, "login failures")
}

func (r repos) CreateSession(ctx context.Context, idHash string, s domain.Session, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO sessions (id_hash, user_id, csrf_token, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		idHash, s.UserID, s.CSRFToken, formatTime(at), formatTime(s.ExpiresAt))
	return exec(res, err, "session")
}

func (r repos) SessionByHash(ctx context.Context, idHash string) (domain.Session, error) {
	var s domain.Session
	var expires string
	err := r.q.QueryRowContext(ctx, `SELECT user_id, csrf_token, expires_at FROM sessions WHERE id_hash = ?`, idHash).
		Scan(&s.UserID, &s.CSRFToken, &expires)
	if err := one(err, "session"); err != nil {
		return s, err
	}
	s.ExpiresAt, err = parseTime(expires)
	return s, err
}

func (r repos) ExtendSession(ctx context.Context, idHash string, expiresAt time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE sessions SET expires_at = ? WHERE id_hash = ?`, formatTime(expiresAt), idHash)
	return affected(res, err, "session")
}

func (r repos) DeleteSession(ctx context.Context, idHash string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM sessions WHERE id_hash = ?`, idHash)
	return exec(res, err, "session")
}

func (r repos) DeleteUserSessions(ctx context.Context, userID string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return exec(res, err, "sessions")
}

func (r repos) CreateTenant(ctx context.Context, t domain.Tenant) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO tenants (id, name, created_at) VALUES (?, ?, ?)`, t.ID, t.Name, formatTime(t.CreatedAt))
	return exec(res, err, "tenant")
}

func (r repos) Tenant(ctx context.Context, id string) (domain.Tenant, error) {
	var t domain.Tenant
	var created string
	err := r.q.QueryRowContext(ctx, `SELECT id, name, created_at FROM tenants WHERE id = ?`, id).Scan(&t.ID, &t.Name, &created)
	if err := one(err, "tenant"); err != nil {
		return t, err
	}
	t.CreatedAt, err = parseTime(created)
	return t, err
}

func (r repos) RenameTenant(ctx context.Context, tenantID, name string) error {
	res, err := r.q.ExecContext(ctx, `UPDATE tenants SET name = ? WHERE id = ?`, name, tenantID)
	return affected(res, err, "tenant")
}

func (r repos) AddMember(ctx context.Context, tenantID, userID string, role domain.Role, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO memberships (tenant_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		tenantID, userID, string(role), formatTime(at))
	if isUnique(err) {
		return domain.Conflict("already a member of this tenant")
	}
	return exec(res, err, "membership")
}

func (r repos) MembershipsOf(ctx context.Context, userID string) ([]domain.Membership, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT t.id, t.name, t.created_at, m.role
		FROM memberships m JOIN tenants t ON t.id = m.tenant_id
		WHERE m.user_id = ? ORDER BY t.name COLLATE NOCASE, t.id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()
	out := []domain.Membership{}
	for rows.Next() {
		var m domain.Membership
		var created, role string
		if err := rows.Scan(&m.Tenant.ID, &m.Tenant.Name, &created, &role); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		if m.Tenant.CreatedAt, err = parseTime(created); err != nil {
			return nil, err
		}
		m.Role = domain.Role(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r repos) RoleOf(ctx context.Context, tenantID, userID string) (domain.Role, error) {
	var role string
	err := r.q.QueryRowContext(ctx, `SELECT role FROM memberships WHERE tenant_id = ? AND user_id = ?`, tenantID, userID).Scan(&role)
	if err := one(err, "tenant"); err != nil {
		return "", err
	}
	return domain.Role(role), nil
}

func (r repos) Members(ctx context.Context, tenantID string) ([]domain.Member, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT u.id, u.email, u.display_name, m.role, m.joined_at
		FROM memberships m JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = ? ORDER BY m.joined_at, u.id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	out := []domain.Member{}
	for rows.Next() {
		var m domain.Member
		var role, joined string
		if err := rows.Scan(&m.UserID, &m.Email, &m.DisplayName, &role, &joined); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		m.Role = domain.Role(role)
		if m.JoinedAt, err = parseTime(joined); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r repos) SetRole(ctx context.Context, tenantID, userID string, role domain.Role) error {
	res, err := r.q.ExecContext(ctx, `UPDATE memberships SET role = ? WHERE tenant_id = ? AND user_id = ?`, string(role), tenantID, userID)
	return affected(res, err, "member")
}

func (r repos) RemoveMember(ctx context.Context, tenantID, userID string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM memberships WHERE tenant_id = ? AND user_id = ?`, tenantID, userID)
	return affected(res, err, "member")
}

func (r repos) CountOwners(ctx context.Context, tenantID string) (int, error) {
	var n int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM memberships WHERE tenant_id = ? AND role = 'owner'`, tenantID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}
	return n, nil
}

func (r repos) CreateInvite(ctx context.Context, inv domain.Invite, tokenHash string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO invites (id, tenant_id, token_hash, role, created_by, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, inv.ID, inv.TenantID, tokenHash, string(inv.Role), inv.CreatedBy,
		formatTime(inv.CreatedAt), formatTime(inv.ExpiresAt))
	return exec(res, err, "invite")
}

const inviteColumns = `id, tenant_id, role, created_by, created_at, expires_at`

func scanInvite(scan func(...any) error) (domain.Invite, error) {
	var inv domain.Invite
	var role, created, expires string
	if err := scan(&inv.ID, &inv.TenantID, &role, &inv.CreatedBy, &created, &expires); err != nil {
		return inv, err
	}
	inv.Role = domain.Role(role)
	var err error
	if inv.CreatedAt, err = parseTime(created); err != nil {
		return inv, err
	}
	inv.ExpiresAt, err = parseTime(expires)
	return inv, err
}

func (r repos) InviteByHash(ctx context.Context, tokenHash string) (domain.Invite, error) {
	inv, err := scanInvite(r.q.QueryRowContext(ctx, `SELECT `+inviteColumns+` FROM invites WHERE token_hash = ? AND used_at IS NULL`, tokenHash).Scan)
	return inv, one(err, "invite")
}

func (r repos) UseInvite(ctx context.Context, id, userID string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE invites SET used_by = ?, used_at = ? WHERE id = ? AND used_at IS NULL`, userID, formatTime(at), id)
	return affected(res, err, "invite")
}

func (r repos) PendingInvites(ctx context.Context, tenantID string, now time.Time) ([]domain.Invite, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+inviteColumns+` FROM invites
		WHERE tenant_id = ? AND used_at IS NULL AND expires_at > ? ORDER BY created_at`, tenantID, formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()
	out := []domain.Invite{}
	for rows.Next() {
		inv, err := scanInvite(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r repos) DeleteInvite(ctx context.Context, tenantID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM invites WHERE tenant_id = ? AND id = ? AND used_at IS NULL`, tenantID, id)
	return affected(res, err, "invite")
}

func (r repos) CreateUploadToken(ctx context.Context, tok domain.UploadToken, tokenHash string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO upload_tokens (id, tenant_id, user_id, name, token_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, tok.ID, tok.TenantID, tok.UserID, tok.Name, tokenHash, formatTime(tok.CreatedAt))
	return exec(res, err, "upload token")
}

const uploadTokenColumns = `id, tenant_id, user_id, name, created_at, last_used_at`

func scanUploadToken(scan func(...any) error) (domain.UploadToken, error) {
	var tok domain.UploadToken
	var created string
	var lastUsed sql.NullString
	if err := scan(&tok.ID, &tok.TenantID, &tok.UserID, &tok.Name, &created, &lastUsed); err != nil {
		return tok, err
	}
	var err error
	if tok.CreatedAt, err = parseTime(created); err != nil {
		return tok, err
	}
	tok.LastUsedAt, err = parseTimePtr(lastUsed)
	return tok, err
}

func (r repos) UploadTokenByHash(ctx context.Context, tokenHash string) (domain.UploadToken, error) {
	tok, err := scanUploadToken(r.q.QueryRowContext(ctx, `SELECT `+uploadTokenColumns+` FROM upload_tokens WHERE token_hash = ?`, tokenHash).Scan)
	return tok, one(err, "upload token")
}

func (r repos) TouchUploadToken(ctx context.Context, id string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE upload_tokens SET last_used_at = ? WHERE id = ?`, formatTime(at), id)
	return affected(res, err, "upload token")
}

func (r repos) UploadTokens(ctx context.Context, tenantID, userID string) ([]domain.UploadToken, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+uploadTokenColumns+` FROM upload_tokens
		WHERE tenant_id = ? AND user_id = ? ORDER BY created_at`, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("list upload tokens: %w", err)
	}
	defer rows.Close()
	out := []domain.UploadToken{}
	for rows.Next() {
		tok, err := scanUploadToken(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan upload token: %w", err)
		}
		out = append(out, tok)
	}
	return out, rows.Err()
}

func (r repos) DeleteUserUploadTokens(ctx context.Context, tenantID, userID string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM upload_tokens WHERE tenant_id = ? AND user_id = ?`, tenantID, userID)
	return exec(res, err, "upload tokens")
}

func (r repos) DeleteUploadToken(ctx context.Context, tenantID, userID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM upload_tokens WHERE tenant_id = ? AND user_id = ? AND id = ?`, tenantID, userID, id)
	return affected(res, err, "upload token")
}
