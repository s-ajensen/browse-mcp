package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Provenance int

const (
	Spawned Provenance = iota
	ConnectedOwned
	ConnectedAttached
)

type Session struct {
	ID            uuid.UUID
	Provenance    Provenance
	CreatedAt     time.Time
	LastActive    time.Time
	CurrentURL    string
	Title         string
	BrowserCtx    context.Context
	allocCancel   context.CancelFunc
	browserCancel context.CancelFunc
}

func NewSession(provenance Provenance) *Session {
	now := time.Now()
	return &Session{
		ID:         uuid.New(),
		Provenance: provenance,
		CreatedAt:  now,
		LastActive: now,
	}
}

func (session *Session) Touch() {
	session.LastActive = time.Now()
}

func (session *Session) Disconnect() {
	if session.browserCancel != nil {
		session.browserCancel()
		session.browserCancel = nil
	}
	if session.allocCancel != nil && session.Provenance == Spawned {
		session.allocCancel()
		session.allocCancel = nil
	}
}
