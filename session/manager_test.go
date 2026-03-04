package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewManager_SetsMaxSessions(t *testing.T) {
	manager := NewManager(5)

	assert.NotNil(t, manager)
}

func TestManager_Add_And_Get_ReturnsStoredSession(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)

	manager.Add(session)
	retrieved, err := manager.Get(session.ID)

	assert.NoError(t, err)
	assert.Equal(t, session, retrieved)
}

func TestManager_Get_ReturnsErrorForUnknownID(t *testing.T) {
	manager := NewManager(5)
	unknownID := uuid.New()

	_, err := manager.Get(unknownID)

	expectedMessage := fmt.Sprintf("session not found: %s. Use browse_spawn or browse_connect to create a session.", unknownID)
	assert.EqualError(t, err, expectedMessage)
}

func TestManager_Add_ReturnsErrorWhenAtCapacity(t *testing.T) {
	manager := NewManager(2)
	manager.Add(NewSession(Spawned))
	manager.Add(NewSession(Spawned))

	err := manager.Add(NewSession(Spawned))

	assert.Error(t, err)
}

func TestManager_Remove_MakesSessionUnretrievable(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)
	manager.Add(session)

	manager.Remove(session.ID)
	_, err := manager.Get(session.ID)

	assert.Error(t, err)
}

func TestManager_List_ReturnsAllSessions(t *testing.T) {
	manager := NewManager(5)
	first := NewSession(Spawned)
	second := NewSession(ConnectedOwned)
	manager.Add(first)
	manager.Add(second)

	listed := manager.List()

	assert.Len(t, listed, 2)
	assert.Contains(t, listed, first)
	assert.Contains(t, listed, second)
}

func TestManager_Get_TouchesSession(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)
	manager.Add(session)
	originalLastActive := session.LastActive
	time.Sleep(time.Millisecond)

	retrieved, _ := manager.Get(session.ID)

	assert.True(t, retrieved.LastActive.After(originalLastActive))
}

func TestManager_ReapIdle_RemovesIdleSessions(t *testing.T) {
	manager := NewManager(5)
	idle := NewSession(Spawned)
	idle.LastActive = time.Now().Add(-time.Hour)
	manager.Add(idle)

	manager.ReapIdle(30 * time.Minute)

	_, err := manager.Get(idle.ID)
	assert.Error(t, err)
}

func TestManager_ReapIdle_KeepsActiveSessions(t *testing.T) {
	manager := NewManager(5)
	active := NewSession(Spawned)
	manager.Add(active)

	manager.ReapIdle(30 * time.Minute)

	retrieved, err := manager.Get(active.ID)
	assert.NoError(t, err)
	assert.Equal(t, active.ID, retrieved.ID)
}

func TestManager_ReapIdle_ReturnsCountOfReapedSessions(t *testing.T) {
	manager := NewManager(5)
	firstIdle := NewSession(Spawned)
	firstIdle.LastActive = time.Now().Add(-time.Hour)
	secondIdle := NewSession(Spawned)
	secondIdle.LastActive = time.Now().Add(-time.Hour)
	active := NewSession(Spawned)
	manager.Add(firstIdle)
	manager.Add(secondIdle)
	manager.Add(active)

	reaped := manager.ReapIdle(30 * time.Minute)

	assert.Equal(t, 2, reaped)
}

func TestManager_ReapIdle_DisconnectsReapedSessions(t *testing.T) {
	manager := NewManager(5)
	idle := NewSession(Spawned)
	idle.LastActive = time.Now().Add(-time.Hour)
	disconnected := false
	idle.browserCancel = func() { disconnected = true }
	manager.Add(idle)

	manager.ReapIdle(30 * time.Minute)

	assert.True(t, disconnected)
}

func TestManager_ShutdownAll_RemovesAllSessions(t *testing.T) {
	manager := NewManager(5)
	manager.Add(NewSession(Spawned))
	manager.Add(NewSession(ConnectedOwned))
	manager.Add(NewSession(Spawned))

	manager.ShutdownAll()

	listed := manager.List()
	assert.Empty(t, listed)
}

func TestManager_ShutdownAll_DisconnectsAllSessions(t *testing.T) {
	manager := NewManager(5)
	first := NewSession(Spawned)
	firstDisconnected := false
	first.browserCancel = func() { firstDisconnected = true }
	second := NewSession(Spawned)
	secondDisconnected := false
	second.browserCancel = func() { secondDisconnected = true }
	manager.Add(first)
	manager.Add(second)

	manager.ShutdownAll()

	assert.True(t, firstDisconnected)
	assert.True(t, secondDisconnected)
}
