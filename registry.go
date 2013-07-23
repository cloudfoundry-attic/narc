package sshark

import (
	"sync"
)

type Registry struct {
	sessions  map[string]*Session
	writeLock sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*Session),
	}
}

func (r *Registry) Register(id string, session *Session) {
	r.writeLock.Lock()
	defer r.writeLock.Unlock()

	r.sessions[id] = session
}

func (r *Registry) Unregister(id string) {
	r.writeLock.Lock()
	defer r.writeLock.Unlock()

	delete(r.sessions, id)
}

func (r *Registry) Lookup(id string) (*Session, bool) {
	val, ok := r.sessions[id]
	return val, ok
}
