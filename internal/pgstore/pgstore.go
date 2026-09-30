// Package pgstore provides PostgreSQL-backed implementations of all five
// transport store interfaces. Each store is safe for concurrent use.
//
// Call Open(ctx, databaseURL) to connect and run migrations. If databaseURL
// is empty, Open returns an error — the caller should fall back to the
// in-memory stores for local development.
//
// Migrations are idempotent (CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT
// EXISTS) so restarting the server never breaks an existing schema.
package pgstore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taxsoftware/internal/auth"
	"taxsoftware/internal/billing"
	"taxsoftware/internal/returns"
)

// DB holds the connection pool shared by all stores.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL and runs the schema migrations.
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgstore: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgstore: ping: %w", err)
	}
	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgstore: migrate: %w", err)
	}
	return db, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() { db.pool.Close() }

func (db *DB) migrate(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS accounts (
			id               TEXT PRIMARY KEY,
			nin_hmac_suffix  BYTEA,
			bvn_hmac_suffix  BYTEA,
			phone_number     TEXT NOT NULL DEFAULT '',
			email_address    TEXT NOT NULL DEFAULT '',
			credential_hash  BYTEA NOT NULL,
			status           TEXT NOT NULL,
			created_at       TIMESTAMPTZ NOT NULL,
			last_login_at    TIMESTAMPTZ,
			verified_at      TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS accounts_nin_idx
			ON accounts(encode(nin_hmac_suffix, 'hex'))
			WHERE nin_hmac_suffix IS NOT NULL;

		CREATE TABLE IF NOT EXISTS return_events (
			id           BIGSERIAL PRIMARY KEY,
			account_id   TEXT NOT NULL,
			return_id    TEXT NOT NULL,
			seq          INT NOT NULL,
			kind         TEXT NOT NULL,
			payload      JSONB NOT NULL,
			actor_kind   TEXT NOT NULL,
			actor_id     TEXT NOT NULL,
			occurred_at  TIMESTAMPTZ NOT NULL,
			UNIQUE(return_id, seq)
		);
		CREATE INDEX IF NOT EXISTS return_events_return_idx
			ON return_events(return_id, seq);
		CREATE INDEX IF NOT EXISTS return_events_account_idx
			ON return_events(account_id, id);

		CREATE TABLE IF NOT EXISTS sessions (
			token       TEXT PRIMARY KEY,
			account_id  TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS recovery_sessions (
			id            TEXT PRIMARY KEY,
			account_id    TEXT NOT NULL,
			method        TEXT NOT NULL,
			state         TEXT NOT NULL,
			otp_hash      BYTEA,
			attempt_count INT NOT NULL DEFAULT 0,
			expires_at    TIMESTAMPTZ NOT NULL,
			completed_at  TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS billing_events (
			id          TEXT PRIMARY KEY,
			account_id  TEXT NOT NULL,
			kind        TEXT NOT NULL,
			tier        TEXT NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			reference   TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS billing_events_account_idx
			ON billing_events(account_id, occurred_at);
	`)
	return err
}

// ---------------------------------------------------------------------------
// AccountStore
// ---------------------------------------------------------------------------

type AccountStore struct{ db *DB }

func (db *DB) Accounts() *AccountStore { return &AccountStore{db} }

func (s *AccountStore) Save(a auth.TaxpayerAccount) error {
	ctx := context.Background()
	_, err := s.db.pool.Exec(ctx, `
		INSERT INTO accounts
			(id, nin_hmac_suffix, bvn_hmac_suffix, phone_number, email_address,
			 credential_hash, status, created_at, last_login_at, verified_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			nin_hmac_suffix = EXCLUDED.nin_hmac_suffix,
			bvn_hmac_suffix = EXCLUDED.bvn_hmac_suffix,
			phone_number    = EXCLUDED.phone_number,
			email_address   = EXCLUDED.email_address,
			credential_hash = EXCLUDED.credential_hash,
			status          = EXCLUDED.status,
			last_login_at   = EXCLUDED.last_login_at,
			verified_at     = EXCLUDED.verified_at
	`,
		a.ID, a.NINHMACSuffix, a.BVNHMACSuffix, a.PhoneNumber, a.EmailAddress,
		a.CredentialHash, string(a.Status), a.CreatedAt, a.LastLoginAt, a.VerifiedAt,
	)
	return err
}

func (s *AccountStore) GetByID(id string) (auth.TaxpayerAccount, error) {
	ctx := context.Background()
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, nin_hmac_suffix, bvn_hmac_suffix, phone_number, email_address,
		       credential_hash, status, created_at, last_login_at, verified_at
		FROM accounts WHERE id = $1
	`, id)
	return scanAccount(row)
}

func (s *AccountStore) FindByNINHMAC(ninHMAC []byte) (auth.TaxpayerAccount, error) {
	ctx := context.Background()
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, nin_hmac_suffix, bvn_hmac_suffix, phone_number, email_address,
		       credential_hash, status, created_at, last_login_at, verified_at
		FROM accounts WHERE encode(nin_hmac_suffix, 'hex') = $1
	`, hex.EncodeToString(ninHMAC))
	return scanAccount(row)
}

func scanAccount(row pgx.Row) (auth.TaxpayerAccount, error) {
	var a auth.TaxpayerAccount
	var status string
	err := row.Scan(
		&a.ID, &a.NINHMACSuffix, &a.BVNHMACSuffix, &a.PhoneNumber, &a.EmailAddress,
		&a.CredentialHash, &status, &a.CreatedAt, &a.LastLoginAt, &a.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.TaxpayerAccount{}, fmt.Errorf("pgstore: account not found")
		}
		return auth.TaxpayerAccount{}, err
	}
	a.Status = auth.AccountStatus(status)
	return a, nil
}

// ---------------------------------------------------------------------------
// EventStore
// ---------------------------------------------------------------------------

type EventStore struct{ db *DB }

func (db *DB) Events() *EventStore { return &EventStore{db} }

func (s *EventStore) Append(accountID string, e returns.Event) error {
	ctx := context.Background()
	_, err := s.db.pool.Exec(ctx, `
		INSERT INTO return_events
			(account_id, return_id, seq, kind, payload, actor_kind, actor_id, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (return_id, seq) DO NOTHING
	`,
		accountID, e.ReturnID, e.Seq, string(e.Kind),
		[]byte(e.Payload), string(e.Actor.Kind), e.Actor.ID, e.OccurredAt,
	)
	return err
}

func (s *EventStore) Load(returnID string) ([]returns.Event, error) {
	ctx := context.Background()
	rows, err := s.db.pool.Query(ctx, `
		SELECT seq, return_id, kind, payload, actor_kind, actor_id, occurred_at
		FROM return_events WHERE return_id = $1 ORDER BY seq
	`, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []returns.Event
	for rows.Next() {
		var e returns.Event
		var kind, actorKind, actorID string
		var payload []byte
		if err := rows.Scan(&e.Seq, &e.ReturnID, &kind, &payload, &actorKind, &actorID, &e.OccurredAt); err != nil {
			return nil, err
		}
		e.Kind = returns.EventKind(kind)
		e.Payload = payload
		e.Actor = returns.Actor{Kind: returns.ActorKind(actorKind), ID: actorID}
		events = append(events, e)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("pgstore: return not found")
	}
	return events, rows.Err()
}

func (s *EventStore) OwnerOf(returnID string) (string, bool) {
	ctx := context.Background()
	var accountID string
	err := s.db.pool.QueryRow(ctx,
		`SELECT account_id FROM return_events WHERE return_id = $1 LIMIT 1`, returnID,
	).Scan(&accountID)
	if err != nil {
		return "", false
	}
	return accountID, true
}

func (s *EventStore) ListForAccount(accountID string) []string {
	ctx := context.Background()
	rows, err := s.db.pool.Query(ctx, `
		SELECT DISTINCT ON (return_id) return_id
		FROM return_events WHERE account_id = $1
		ORDER BY return_id, id
	`, accountID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// ---------------------------------------------------------------------------
// SessionStore
// ---------------------------------------------------------------------------

type SessionStore struct{ db *DB }

func (db *DB) Sessions() *SessionStore { return &SessionStore{db} }

func (s *SessionStore) Create(token, accountID string) error {
	ctx := context.Background()
	_, err := s.db.pool.Exec(ctx,
		`INSERT INTO sessions (token, account_id) VALUES ($1, $2) ON CONFLICT (token) DO NOTHING`,
		token, accountID,
	)
	return err
}

func (s *SessionStore) AccountID(token string) (string, bool) {
	ctx := context.Background()
	var accountID string
	err := s.db.pool.QueryRow(ctx,
		`SELECT account_id FROM sessions WHERE token = $1`, token,
	).Scan(&accountID)
	if err != nil {
		return "", false
	}
	return accountID, true
}

func (s *SessionStore) Delete(token string) error {
	ctx := context.Background()
	_, err := s.db.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// ---------------------------------------------------------------------------
// RecoveryStore
// ---------------------------------------------------------------------------

type RecoveryStore struct{ db *DB }

func (db *DB) Recovery() *RecoveryStore { return &RecoveryStore{db} }

func (s *RecoveryStore) Save(rs auth.RecoverySession) error {
	ctx := context.Background()
	_, err := s.db.pool.Exec(ctx, `
		INSERT INTO recovery_sessions
			(id, account_id, method, state, otp_hash, attempt_count, expires_at, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			state         = EXCLUDED.state,
			otp_hash      = EXCLUDED.otp_hash,
			attempt_count = EXCLUDED.attempt_count,
			completed_at  = EXCLUDED.completed_at
	`,
		rs.ID, rs.AccountID, string(rs.Method), string(rs.State),
		rs.OTPHash, rs.AttemptCount, rs.ExpiresAt, rs.CompletedAt,
	)
	return err
}

func (s *RecoveryStore) Get(id string) (auth.RecoverySession, error) {
	ctx := context.Background()
	var rs auth.RecoverySession
	var method, state string
	err := s.db.pool.QueryRow(ctx, `
		SELECT id, account_id, method, state, otp_hash, attempt_count, expires_at, completed_at
		FROM recovery_sessions WHERE id = $1
	`, id).Scan(
		&rs.ID, &rs.AccountID, &method, &state,
		&rs.OTPHash, &rs.AttemptCount, &rs.ExpiresAt, &rs.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.RecoverySession{}, fmt.Errorf("pgstore: recovery session not found")
		}
		return auth.RecoverySession{}, err
	}
	rs.Method = auth.RecoveryMethod(method)
	rs.State = auth.RecoveryState(state)
	return rs, nil
}

// ---------------------------------------------------------------------------
// BillingStore
// ---------------------------------------------------------------------------

type BillingStore struct{ db *DB }

func (db *DB) Billing() *BillingStore { return &BillingStore{db} }

func (s *BillingStore) Append(e billing.BillingEvent) error {
	ctx := context.Background()
	if e.ID == "" {
		e.ID = fmt.Sprintf("%s-%d", e.AccountID, time.Now().UnixNano())
	}
	_, err := s.db.pool.Exec(ctx, `
		INSERT INTO billing_events (id, account_id, kind, tier, occurred_at, reference)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO NOTHING
	`, e.ID, e.AccountID, string(e.Kind), string(e.Tier), e.OccurredAt, e.Reference)
	return err
}

func (s *BillingStore) Events(accountID string) ([]billing.BillingEvent, error) {
	ctx := context.Background()
	rows, err := s.db.pool.Query(ctx, `
		SELECT id, account_id, kind, tier, occurred_at, reference
		FROM billing_events WHERE account_id = $1 ORDER BY occurred_at
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []billing.BillingEvent
	for rows.Next() {
		var e billing.BillingEvent
		var kind, tier string
		if err := rows.Scan(&e.ID, &e.AccountID, &kind, &tier, &e.OccurredAt, &e.Reference); err != nil {
			return nil, err
		}
		e.Kind = billing.BillingEventKind(kind)
		e.Tier = billing.Tier(tier)
		events = append(events, e)
	}
	return events, rows.Err()
}
