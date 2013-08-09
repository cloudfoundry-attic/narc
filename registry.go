package narc

import (
	"sync"
)

type Tasks map[string]*Task

type Registry struct {
	tasks Tasks
	lock  sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]*Task),
	}
}

func (r *Registry) Register(id string, task *Task) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.tasks[id] = task
}

func (r *Registry) Unregister(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.tasks, id)
}

func (r *Registry) Lookup(id string) (*Task, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	val, ok := r.tasks[id]
	return val, ok
}
