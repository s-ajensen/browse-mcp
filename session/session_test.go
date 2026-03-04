package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProvenance_ValuesAreDistinct(t *testing.T) {
	assert.NotEqual(t, Spawned, ConnectedOwned)
	assert.NotEqual(t, Spawned, ConnectedAttached)
	assert.NotEqual(t, ConnectedOwned, ConnectedAttached)
}

func TestNewSession_SetsNonZeroID(t *testing.T) {
	created := NewSession(Spawned)

	assert.NotEqual(t, uuid.Nil, created.ID)
}

func TestNewSession_SetsProvenance(t *testing.T) {
	created := NewSession(ConnectedOwned)

	assert.Equal(t, ConnectedOwned, created.Provenance)
}

func TestNewSession_SetsCreatedAtToNow(t *testing.T) {
	before := time.Now()
	created := NewSession(Spawned)
	after := time.Now()

	assert.True(t, !created.CreatedAt.Before(before))
	assert.True(t, !created.CreatedAt.After(after))
}

func TestNewSession_SetsLastActiveToNow(t *testing.T) {
	before := time.Now()
	created := NewSession(Spawned)
	after := time.Now()

	assert.True(t, !created.LastActive.Before(before))
	assert.True(t, !created.LastActive.After(after))
}

func TestNewSession_CurrentURLIsEmpty(t *testing.T) {
	created := NewSession(Spawned)

	assert.Empty(t, created.CurrentURL)
}

func TestTouch_UpdatesLastActive(t *testing.T) {
	created := NewSession(Spawned)
	originalLastActive := created.LastActive
	time.Sleep(time.Millisecond)

	created.Touch()

	assert.True(t, created.LastActive.After(originalLastActive))
}

func TestTouch_DoesNotChangeCreatedAt(t *testing.T) {
	created := NewSession(Spawned)
	originalCreatedAt := created.CreatedAt
	time.Sleep(time.Millisecond)

	created.Touch()

	assert.Equal(t, originalCreatedAt, created.CreatedAt)
}

func sessionWithCancels(provenance Provenance) (*Session, *bool, *bool) {
	session := NewSession(provenance)
	allocCalled := false
	browserCalled := false
	session.allocCancel = func() { allocCalled = true }
	session.browserCancel = func() { browserCalled = true }
	return session, &allocCalled, &browserCalled
}

func TestDisconnect_Spawned_CallsBothCancels(t *testing.T) {
	session, allocCalled, browserCalled := sessionWithCancels(Spawned)

	session.Disconnect()

	assert.True(t, *allocCalled)
	assert.True(t, *browserCalled)
}

func TestDisconnect_ConnectedOwned_CallsOnlyBrowserCancel(t *testing.T) {
	session, allocCalled, browserCalled := sessionWithCancels(ConnectedOwned)

	session.Disconnect()

	assert.False(t, *allocCalled)
	assert.True(t, *browserCalled)
}

func TestDisconnect_ConnectedAttached_CallsOnlyBrowserCancel(t *testing.T) {
	session, allocCalled, browserCalled := sessionWithCancels(ConnectedAttached)

	session.Disconnect()

	assert.False(t, *allocCalled)
	assert.True(t, *browserCalled)
}

func TestDisconnect_IsIdempotent_CancelsCalledOnce(t *testing.T) {
	session := NewSession(Spawned)
	allocCount := 0
	browserCount := 0
	session.allocCancel = func() { allocCount++ }
	session.browserCancel = func() { browserCount++ }

	session.Disconnect()
	session.Disconnect()

	assert.Equal(t, 1, allocCount)
	assert.Equal(t, 1, browserCount)
}

func TestDisconnect_NilCancelFunctions_DoesNotPanic(t *testing.T) {
	session := NewSession(Spawned)

	assert.NotPanics(t, func() { session.Disconnect() })
}
