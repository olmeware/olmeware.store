package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup yields no row.
var ErrNotFound = errors.New("not found")

// Repo is the data-access layer for users and sessions.
type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

const userColumns = `id, email, password_hash, full_name, role, status, last_login_at, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role,
		&u.Status, &u.LastLoginAt, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a new user and returns it.
func (r *Repo) CreateUser(ctx context.Context, email, passwordHash, name, role string) (*User, error) {
	const q = `insert into users (email, password_hash, full_name, role)
		values (lower(btrim($1)), $2, btrim($3), $4)
		returning ` + userColumns
	return scanUser(r.db.QueryRow(ctx, q, email, passwordHash, name, role))
}

// GetUserByEmail returns a live (non-deleted) user by email.
func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `select ` + userColumns + ` from users
		where lower(email) = lower(btrim($1)) and deleted_at is null`
	return scanUser(r.db.QueryRow(ctx, q, email))
}

// GetUserByID returns a live user by id.
func (r *Repo) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `select ` + userColumns + ` from users where id = $1 and deleted_at is null`
	return scanUser(r.db.QueryRow(ctx, q, id))
}

// EmailExists reports whether a live user already uses the email.
func (r *Repo) EmailExists(ctx context.Context, email string) (bool, error) {
	const q = `select exists(select 1 from users
		where lower(email) = lower(btrim($1)) and deleted_at is null)`
	var exists bool
	err := r.db.QueryRow(ctx, q, email).Scan(&exists)
	return exists, err
}

// TouchLogin stamps last_login_at.
func (r *Repo) TouchLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `update users set last_login_at = now() where id = $1`, id)
	return err
}

// CreateSession persists a refresh-token session.
func (r *Repo) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash, userAgent, ip string, expiresAt time.Time) error {
	const q = `insert into user_sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at)
		values ($1, $2, nullif($3,''), nullif($4,'')::inet, $5)`
	_, err := r.db.Exec(ctx, q, userID, tokenHash, userAgent, ip, expiresAt)
	return err
}

// SessionUser resolves the active session for a refresh token hash and returns
// its user. Expired or revoked sessions are treated as not found.
func (r *Repo) SessionUser(ctx context.Context, tokenHash string) (*User, error) {
	q := `select ` + prefixed("u", userColumns) + `
		from user_sessions s join users u on u.id = s.user_id
		where s.refresh_token_hash = $1 and s.revoked_at is null
		  and s.expires_at > now() and u.deleted_at is null`
	return scanUser(r.db.QueryRow(ctx, q, tokenHash))
}

// RevokeSession marks a refresh-token session revoked (logout).
func (r *Repo) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx,
		`update user_sessions set revoked_at = now()
		 where refresh_token_hash = $1 and revoked_at is null`, tokenHash)
	return err
}

// prefixed rewrites a bare column list to be qualified by a table alias.
func prefixed(alias, columns string) string {
	out := ""
	col := ""
	flush := func() {
		if col != "" {
			if out != "" {
				out += ", "
			}
			out += alias + "." + col
			col = ""
		}
	}
	for _, ch := range columns {
		switch ch {
		case ',':
			flush()
		case ' ', '\t', '\n':
		default:
			col += string(ch)
		}
	}
	flush()
	return out
}
