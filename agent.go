package narc

import (
	"encoding/json"
	"errors"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/nu7hatch/gouuid"
	"log"
	"os/exec"
)

type Agent struct {
	ID       *uuid.UUID
	Registry *Registry

	taskBackend TaskBackend
}

type TaskBackend interface {
	ProvideContainer() (Container, error)
	ProvideCommand(Container) *exec.Cmd
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

type startMessage struct {
	Task                   string `json:"task"`
	SecureToken            string `json:"secure_token"`
	MemoryLimitInMegabytes uint64 `json:"memory_limit"`
	DiskLimitInMegabytes   uint64 `json:"disk_limit"`
}

type stopMessage struct {
	Task string `json:"task"`
}

var TaskNotRegistered = errors.New("task not registered")

func NewAgent(taskBackend TaskBackend) (*Agent, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Agent{
		ID:       id,
		Registry: NewRegistry(),

		taskBackend: taskBackend,
	}, nil
}

func (a *Agent) HandleStarts(mbus cfmessagebus.MessageBus) error {
	return mbus.Subscribe("task.start", func(payload []byte) {
		var start startMessage

		err := json.Unmarshal(payload, &start)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		a.handleStart(start)
	})
}

func (a *Agent) HandleStops(mbus cfmessagebus.MessageBus) error {
	return mbus.Subscribe("task.stop", func(payload []byte) {
		var stop stopMessage

		err := json.Unmarshal(payload, &stop)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		a.handleStop(stop)
	})
}

func (a *Agent) handleStart(start startMessage) {
	log.Printf("creating task %s\n", start.Task)

	limits := TaskLimits{
		MemoryLimitInBytes: start.MemoryLimitInMegabytes * 1024 * 1024,
		DiskLimitInBytes:   start.DiskLimitInMegabytes * 1024 * 1024,
	}

	_, err := a.startTask(start.Task, start.SecureToken, limits)
	if err != nil {
		log.Printf("failed to create task: %s\n", err)
		return
	}
}

func (a *Agent) handleStop(stop stopMessage) {
	log.Printf("stopping task %s\n", stop.Task)

	err := a.stopTask(stop.Task)
	if err != nil {
		log.Printf("failed to stop task: %s\n", err)
		return
	}
}

func (a *Agent) startTask(guid, secureToken string, limits TaskLimits) (*Task, error) {
	container, err := a.createTaskContainer(limits)
	if err != nil {
		return nil, err
	}

	task, err := NewTask(container, secureToken, a.taskBackend.ProvideCommand(container))
	if err != nil {
		return nil, err
	}

	a.Registry.Register(guid, task)

	task.OnComplete(func() {
		log.Println("task completed:", guid)

		a.Registry.Unregister(guid)
	})

	return task, nil
}

func (a *Agent) stopTask(guid string) error {
	task, present := a.Registry.Lookup(guid)
	if !present {
		return TaskNotRegistered
	}

	a.Registry.Unregister(guid)

	err := task.Stop()
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) createTaskContainer(limits TaskLimits) (Container, error) {
	container, err := a.taskBackend.ProvideContainer()
	if err != nil {
		return nil, err
	}

	if limits.MemoryLimitInBytes != 0 {
		err := container.LimitMemory(limits.MemoryLimitInBytes)
		if err != nil {
			return nil, err
		}
	}

	if limits.DiskLimitInBytes != 0 {
		err := container.LimitDisk(limits.DiskLimitInBytes)
		if err != nil {
			return nil, err
		}
	}

	return container, nil
}
