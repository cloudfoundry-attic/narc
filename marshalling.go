package sshark

import (
	"encoding/json"
)

type sessionLimitsJSON struct {
	MemoryLimitInMegabytes uint64 `json:"memory"`
	DiskLimitInMegabytes   uint64 `json:"disk"`
}

type sessionJSON struct {
	Container string            `json:"container"`
	Port      MappedPort        `json:"port"`
	Limits    sessionLimitsJSON `json:"limits"`
}

type agentJSON struct {
	ID       string    `json:"id"`
	Sessions *Registry `json:"sessions"`
}

func (s *Session) MarshalJSON() ([]byte, error) {
	return json.Marshal(makeSessionJSON(s))
}

func (s *Sessions) MarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (r *Registry) MarshalJSON() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	sessions := make(map[string]*sessionJSON)

	for id, session := range r.sessions {
		sessions[id] = makeSessionJSON(session)
	}

	return json.Marshal(sessions)
}

func (a *Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&agentJSON{
		ID:       a.ID.String(),
		Sessions: a.Registry,
	})
}

func makeSessionJSON(s *Session) *sessionJSON {
	return &sessionJSON{
		Container: s.Container.ID(),
		Port:      s.Port,
		Limits: sessionLimitsJSON{
			MemoryLimitInMegabytes: s.Limits.MemoryLimitInBytes / 1024 / 1024,
			DiskLimitInMegabytes:   s.Limits.DiskLimitInBytes / 1024 / 1024,
		},
	}
}
