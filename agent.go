package sshark

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/nu7hatch/gouuid"
	"github.com/vito/gordon"
	"io/ioutil"
	"log"
	"time"
)

type AgentConfig struct {
	WardenSocketPath  string
	StateFilePath     string
	AdvertiseInterval time.Duration
}

type Agent struct {
	ID       *uuid.UUID
	Registry *Registry
	Config   AgentConfig
}

type AdvertiseMessage struct {
	ID string `json:"id"`
}

var SessionNotRegistered = errors.New("session not registered")

func NewAgent(config AgentConfig) (*Agent, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	agent := &Agent{
		ID:       id,
		Registry: NewRegistry(),
		Config:   config,
	}

	err = agent.writeState()

	return agent, err
}

func (a *Agent) StartSession(guid string, limits SessionLimits) (*Session, error) {
	session, err := a.createSession(limits)
	if err != nil {
		return nil, err
	}

	a.Registry.Register(guid, session)

	err = a.writeState()

	return session, err
}

func (a *Agent) StopSession(guid string) error {
	session, present := a.Registry.Lookup(guid)

	if !present {
		return SessionNotRegistered
	}

	err := session.Container.Destroy()
	if err != nil {
		return err
	}

	a.Registry.Unregister(guid)

	err = a.writeState()

	return err
}

func (a *Agent) HandleStarts(mbus cfmessagebus.MessageBus) error {
	directedStart := fmt.Sprintf("ssh.%s.start", a.ID.String())

	return mbus.Subscribe(directedStart, func(payload []byte) {
		var start startMessage

		err := json.Unmarshal(payload, &start)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		go a.handleStart(start)
	})
}

func (a *Agent) HandleStops(mbus cfmessagebus.MessageBus) error {
	return mbus.Subscribe("ssh.stop", func(payload []byte) {
		var stop stopMessage

		err := json.Unmarshal(payload, &stop)
		if err != nil {
			log.Printf("Failed to unmarshal ssh start: %s\n", err)
			return
		}

		go a.handleStop(stop)
	})
}

func (a *Agent) AdvertisePeriodically(mbus cfmessagebus.MessageBus) {
	for {
		select {
		case <-time.After(a.Config.AdvertiseInterval):
			a.sendAdvertisement(mbus)
		}
	}
}

func (a *Agent) sendAdvertisement(mbus cfmessagebus.MessageBus) {
	msg, err := json.Marshal(&AdvertiseMessage{ID: a.ID.String()})
	if err != nil {
		return
	}

	mbus.Publish("ssh.advertise", msg)
}

func (a *Agent) createSession(limits SessionLimits) (*Session, error) {
	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: a.Config.WardenSocketPath,
		},
	)

	err := client.Connect()
	if err != nil {
		return nil, err
	}

	container, err := NewWardenContainer(client)
	if err != nil {
		return nil, err
	}

	port, err := container.NetIn()
	if err != nil {
		return nil, err
	}

	err = container.LimitMemory(limits.MemoryLimitInBytes)
	if err != nil {
		return nil, err
	}

	err = container.LimitDisk(limits.DiskLimitInBytes)
	if err != nil {
		return nil, err
	}

	return &Session{
		Container: container,
		Port:      port,
	}, nil
}

type sessionLimitsMessage struct {
	MemoryInMegabytes uint64 `json:"memory"`
	DiskInMegabytes   uint64 `json:"disk"`
}

type startMessage struct {
	Session   string               `json:"session"`
	PublicKey string               `json:"public_key"`
	Limits    sessionLimitsMessage `json:"limits"`
}

type stopMessage struct {
	Session string `json:"session"`
}

func (a *Agent) handleStart(start startMessage) {
	log.Printf(
		"creating session %s\n",
		start.Session,
	)

	limits := SessionLimits{
		MemoryLimitInBytes: start.Limits.MemoryInMegabytes * 1024 * 1024,
		DiskLimitInBytes:   start.Limits.DiskInMegabytes * 1024 * 1024,
	}

	sess, err := a.StartSession(start.Session, limits)
	if err != nil {
		log.Printf("Failed to create session: %s\n", err)
		return
	}

	log.Printf(
		"loading public key into session %s\n",
		start.Session,
	)

	err = sess.LoadPublicKey(start.PublicKey)
	if err != nil {
		log.Printf("Failed to load public key: %s\n", err)
		return
	}

	log.Printf(
		"starting SSH server %s on port %d\n",
		start.Session,
		sess.Port,
	)

	err = sess.StartSSHServer()
	if err != nil {
		log.Printf("Failed to start SSH server: %s\n", err)
		return
	}
}

func (a *Agent) handleStop(stop stopMessage) {
	log.Printf(
		"stopping session %s\n",
		stop.Session,
	)

	err := a.StopSession(stop.Session)
	if err != nil {
		log.Printf("Failed to stop session: %s\n", err)
		return
	}
}

func (a *Agent) writeState() error {
	if a.Config.StateFilePath == "" {
		return nil
	}

	json, err := a.MarshalJSON()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(a.Config.StateFilePath, json, 0644)
}
