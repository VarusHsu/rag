package security

import (
	"sync"
	"time"
)

type TokenRevoker interface {
	Revoke(token string, expiresAt time.Time)
	IsRevoked(token string) bool
}

type InMemoryTokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

func NewInMemoryTokenBlacklist() *InMemoryTokenBlacklist {
	return &InMemoryTokenBlacklist{tokens: make(map[string]time.Time)}
}

func (b *InMemoryTokenBlacklist) Revoke(token string, expiresAt time.Time) {
	if token == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupLocked(time.Now())
	b.tokens[token] = expiresAt
}

func (b *InMemoryTokenBlacklist) IsRevoked(token string) bool {
	if token == "" {
		return false
	}

	now := time.Now()

	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupLocked(now)

	expiresAt, ok := b.tokens[token]
	if !ok {
		return false
	}
	if !expiresAt.IsZero() && expiresAt.Before(now) {
		delete(b.tokens, token)
		return false
	}
	return true
}

func (b *InMemoryTokenBlacklist) cleanupLocked(now time.Time) {
	for token, expiresAt := range b.tokens {
		if !expiresAt.IsZero() && expiresAt.Before(now) {
			delete(b.tokens, token)
		}
	}
}
