package sshark

import (
	"encoding/json"
)

type sessionJSON struct {
	Container string     `json:"container"`
	Port      MappedPort `json:"port"`
}

type agentJSON struct {
	ID       string    `json:"id"`
	Sessions *Registry `json:"sessions"`
}

func (s *Session) MarshalJSON() ([]byte, error) {
	return json.Marshal(&sessionJSON{
		Container: s.Container.ID(),
		Port:      s.Port,
	})
}

func (s *Sessions) MarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (r *Registry) MarshalJSON() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	sessions := make(map[string]*sessionJSON)

	for id, session := range r.sessions {
		sessions[id] = &sessionJSON{
			Port:      session.Port,
			Container: session.Container.ID(),
		}
	}

	return json.Marshal(sessions)
}

func (a *Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&agentJSON{
		ID:       a.ID.String(),
		Sessions: a.Registry,
	})
}
