package narc

import (
	"encoding/json"
	"errors"
	"log"
	"os/exec"

	"github.com/cloudfoundry/gibson"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/nu7hatch/gouuid"
)

type Agent struct {
	ID       *uuid.UUID
	Registry *Registry

	taskBackend TaskBackend

	routerClient gibson.RouterClient
	routerPort   int
}

type RouterRegistrar interface {
	Register(string, int)
	Unregister(string, int)
}

type TaskBackend interface {
	ProvideContainer(TaskLimits) (Container, error)
	ProvideCommand(Container) *exec.Cmd
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
var TaskAlreadyRegistered = errors.New("task already registered")

func NewAgent(taskBackend TaskBackend, routerClient gibson.RouterClient, port int) (*Agent, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Agent{
		ID:       id,
		Registry: NewRegistry(),
		taskBackend: taskBackend,
		routerClient: routerClient,
		routerPort:   port,
	}, nil
}

func (agent *Agent) HandleStarts(mbus cfmessagebus.MessageBus) error {
	return mbus.Subscribe("task.start", func(payload []byte) {
		var start startMessage

		err := json.Unmarshal(payload, &start)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		agent.handleStart(start)
	})
}

func (agent *Agent) HandleStops(mbus cfmessagebus.MessageBus) error {
	return mbus.Subscribe("task.stop", func(payload []byte) {
		var stop stopMessage

		err := json.Unmarshal(payload, &stop)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		agent.handleStop(stop)
	})
}

func (agent *Agent) handleStart(start startMessage) {
	log.Printf("creating task %s\n", start.Task)
	limits := TaskLimits{
		MemoryLimitInBytes: start.MemoryLimitInMegabytes*1024*1024,
		DiskLimitInBytes:   start.DiskLimitInMegabytes*1024*1024,
	}
	if !limits.IsValid() {
		log.Printf("Must specify memory and disk: %s\n", limits)
		return
	}

	_, err := agent.startTask(start.Task, start.SecureToken, limits)
	if err != nil {
		log.Printf("failed to create task: %s\n", err)
	}
}

func (agent *Agent) handleStop(stop stopMessage) {
	log.Printf("stopping task %s\n", stop.Task)

	err := agent.stopTask(stop.Task)
	if err != nil {
		log.Printf("failed to stop task: %s\n", err)
	}
}

func (agent *Agent) startTask(guid, secureToken string, limits TaskLimits) (*Task, error) {
	_, present := agent.Registry.Lookup(guid)
	if present {
		return nil, TaskAlreadyRegistered
	}

	container, err := agent.createTaskContainer(limits)
	if err != nil {
		return nil, err
	}

	task, err := NewTask(container, secureToken, agent.taskBackend.ProvideCommand(container))
	if err != nil {
		return nil, err
	}

	agent.Registry.Register(guid, task)

	agent.routerClient.Register(agent.routerPort, guid)

	task.OnComplete(func() {
		log.Println("task completed:", guid)
		agent.cleanUpGuid(guid)
	})

	return task, nil
}

func (a *Agent) stopTask(guid string) error {
	task, present := a.Registry.Lookup(guid)
	if !present {
		return TaskNotRegistered
	}

	a.cleanUpGuid(guid)

	err := task.Stop()
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) cleanUpGuid(guid string) {
	a.routerClient.Unregister(a.routerPort, guid)
	a.Registry.Unregister(guid)
}

func (agent *Agent) createTaskContainer(limits TaskLimits) (Container, error) {
	container, err := agent.taskBackend.ProvideContainer(limits)
	if err != nil {
		return nil, err
	}
	return container, nil
}
