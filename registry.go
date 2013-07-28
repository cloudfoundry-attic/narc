package sshark

import (
	"sync"
)

type Sessions map[string]*Session

type Registry struct {
	sessions Sessions
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
