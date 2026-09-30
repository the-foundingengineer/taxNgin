package transport

import (
	"encoding/hex"
	"errors"
	"sync"

	"taxsoftware/internal/auth"
	"taxsoftware/internal/billing"
	"taxsoftware/internal/returns"
)

var (
	ErrNotFound = errors.New("transport: not found")
	ErrConflict = errors.New("transport: conflict")
)

// AccountStore persists TaxpayerAccounts, indexed by ID and NIN HMAC.
type AccountStore interface {
	Save(a auth.TaxpayerAccount) error
	GetByID(id string) (auth.TaxpayerAccount, error)
	FindByNINHMAC(ninHMAC []byte) (auth.TaxpayerAccount, error)
}

// EventStore persists the append-only event log, keyed by return ID.
type EventStore interface {
	Append(accountID string, e returns.Event) error
	Load(returnID string) ([]returns.Event, error)
	OwnerOf(returnID string) (string, bool)
	// ListForAccount returns all return IDs owned by accountID, in creation order.
	ListForAccount(accountID string) []string
}

// SessionStore maps session tokens to account IDs.
type SessionStore interface {
	Create(token, accountID string) error
	AccountID(token string) (string, bool)
	Delete(token string) error
}

// RecoveryStore persists in-flight account recovery sessions.
type RecoveryStore interface {
	Save(s auth.RecoverySession) error
	Get(id string) (auth.RecoverySession, error)
}

// BillingStore persists billing events per account.
type BillingStore interface {
	Events(accountID string) ([]billing.BillingEvent, error)
	Append(e billing.BillingEvent) error
}

// ---------------------------------------------------------------------------
// In-memory implementations
// ---------------------------------------------------------------------------

type MemAccountStore struct {
	mu     sync.RWMutex
	byID   map[string]auth.TaxpayerAccount
	ninIdx map[string]string // hex(ninHMAC) → accountID
}

func NewMemAccountStore() *MemAccountStore {
	return &MemAccountStore{
		byID:   make(map[string]auth.TaxpayerAccount),
		ninIdx: make(map[string]string),
	}
}

func (s *MemAccountStore) Save(a auth.TaxpayerAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[a.ID] = a
	if len(a.NINHMACSuffix) > 0 {
		s.ninIdx[hex.EncodeToString(a.NINHMACSuffix)] = a.ID
	}
	return nil
}

func (s *MemAccountStore) GetByID(id string) (auth.TaxpayerAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.byID[id]
	if !ok {
		return auth.TaxpayerAccount{}, ErrNotFound
	}
	return a, nil
}

func (s *MemAccountStore) FindByNINHMAC(ninHMAC []byte) (auth.TaxpayerAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.ninIdx[hex.EncodeToString(ninHMAC)]
	if !ok {
		return auth.TaxpayerAccount{}, ErrNotFound
	}
	a, ok := s.byID[id]
	if !ok {
		return auth.TaxpayerAccount{}, ErrNotFound
	}
	return a, nil
}

type MemEventStore struct {
	mu       sync.RWMutex
	events   map[string][]returns.Event // returnID → events
	owner    map[string]string          // returnID → accountID
	accounts map[string][]string        // accountID → returnIDs (insertion order)
}

func NewMemEventStore() *MemEventStore {
	return &MemEventStore{
		events:   make(map[string][]returns.Event),
		owner:    make(map[string]string),
		accounts: make(map[string][]string),
	}
}

func (s *MemEventStore) Append(accountID string, e returns.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.owner[e.ReturnID]; !exists {
		s.accounts[accountID] = append(s.accounts[accountID], e.ReturnID)
	}
	s.events[e.ReturnID] = append(s.events[e.ReturnID], e)
	s.owner[e.ReturnID] = accountID
	return nil
}

func (s *MemEventStore) ListForAccount(accountID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.accounts[accountID]
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

func (s *MemEventStore) Load(returnID string) ([]returns.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	evts, ok := s.events[returnID]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]returns.Event, len(evts))
	copy(out, evts)
	return out, nil
}

func (s *MemEventStore) OwnerOf(returnID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.owner[returnID]
	return id, ok
}

type MemSessionStore struct {
	mu  sync.RWMutex
	tok map[string]string // token → accountID
}

func NewMemSessionStore() *MemSessionStore {
	return &MemSessionStore{tok: make(map[string]string)}
}

func (s *MemSessionStore) Create(token, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tok[token] = accountID
	return nil
}

func (s *MemSessionStore) AccountID(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.tok[token]
	return id, ok
}

func (s *MemSessionStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tok, token)
	return nil
}

type MemRecoveryStore struct {
	mu       sync.RWMutex
	sessions map[string]auth.RecoverySession
}

func NewMemRecoveryStore() *MemRecoveryStore {
	return &MemRecoveryStore{sessions: make(map[string]auth.RecoverySession)}
}

func (s *MemRecoveryStore) Save(rs auth.RecoverySession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[rs.ID] = rs
	return nil
}

func (s *MemRecoveryStore) Get(id string) (auth.RecoverySession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rs, ok := s.sessions[id]
	if !ok {
		return auth.RecoverySession{}, ErrNotFound
	}
	return rs, nil
}

type MemBillingStore struct {
	mu     sync.RWMutex
	events map[string][]billing.BillingEvent // accountID → events
}

func NewMemBillingStore() *MemBillingStore {
	return &MemBillingStore{events: make(map[string][]billing.BillingEvent)}
}

func (s *MemBillingStore) Events(accountID string) ([]billing.BillingEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]billing.BillingEvent(nil), s.events[accountID]...), nil
}

func (s *MemBillingStore) Append(e billing.BillingEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[e.AccountID] = append(s.events[e.AccountID], e)
	return nil
}
