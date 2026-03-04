package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	maxSessions int
	sessions    map[uuid.UUID]*Session
	mutex       sync.Mutex
}

func NewManager(maxSessions int) *Manager {
	return &Manager{
		maxSessions: maxSessions,
		sessions:    make(map[uuid.UUID]*Session),
	}
}

func (manager *Manager) Add(session *Session) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if len(manager.sessions) >= manager.maxSessions {
		return fmt.Errorf("maximum sessions reached: %d", manager.maxSessions)
	}
	manager.sessions[session.ID] = session
	return nil
}

func (manager *Manager) Get(id uuid.UUID) (*Session, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	found, exists := manager.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s. Use browse_spawn or browse_connect to create a session.", id)
	}
	found.Touch()
	return found, nil
}

func (manager *Manager) Remove(id uuid.UUID) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	delete(manager.sessions, id)
}

func (manager *Manager) List() []*Session {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	result := make([]*Session, 0, len(manager.sessions))
	for _, session := range manager.sessions {
		result = append(result, session)
	}
	return result
}

func (manager *Manager) ReapIdle(maxIdle time.Duration) int {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	count := 0
	for id, session := range manager.sessions {
		if time.Since(session.LastActive) > maxIdle {
			session.Disconnect()
			delete(manager.sessions, id)
			count++
		}
	}
	return count
}

func (manager *Manager) ShutdownAll() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, session := range manager.sessions {
		session.Disconnect()
	}
	manager.sessions = make(map[uuid.UUID]*Session)
}
