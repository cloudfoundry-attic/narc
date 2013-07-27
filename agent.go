package sshark

import (
	"errors"
	"github.com/vito/gordon"
)

type Agent struct {
	Registry         *Registry
	WardenSocketPath string
}

var SessionNotRegistered = errors.New("session not registered")

func (a *Agent) StartSession(guid, publicKey string) (*Session, error) {
	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: a.WardenSocketPath,
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

	session := &Session{
		Container: container,
		Port:      port,
	}

	a.Registry.Register(guid, session)

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

	return nil
}
