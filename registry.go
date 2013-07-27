package sshark

import (
	"encoding/json"
	"sync"
)

type Registry struct {
	sessions map[string]*Session
	lock     sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*Session),
	}
}

func (r *Registry) Register(id string, session *Session) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.sessions[id] = session
}

func (r *Registry) Unregister(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.sessions, id)
}

func (r *Registry) Lookup(id string) (*Session, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	val, ok := r.sessions[id]
	return val, ok
}

type sessionJSON struct {
	Port      MappedPort `json:"port"`
	Container string     `json:"container"`
}

type registryJSON struct {
	Sessions map[string]*sessionJSON `json:"sessions"`
}

func (r *Registry) MarshalJSON() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registry := &registryJSON{
		Sessions: make(map[string]*sessionJSON),
	}

	for id, session := range r.sessions {
		registry.Sessions[id] = &sessionJSON{
			Port:      session.Port,
			Container: session.Container.ID(),
		}
	}

	return json.Marshal(registry)
}
