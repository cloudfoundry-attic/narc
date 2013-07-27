package sshark

import (
	"errors"
)

type FakeContainer struct {
	LastCommand string
	ShouldError bool
	Handle      string
}

func (c *FakeContainer) ID() string {
	return c.Handle
}

func (c *FakeContainer) Destroy() error {
	return nil
}

func (c *FakeContainer) Run(command string) (*JobInfo, error) {
	if c.ShouldError {
		return nil, errors.New("uh oh")
	}

	c.LastCommand = command

	return &JobInfo{}, nil
}

func (c *FakeContainer) NetIn() (MappedPort, error) {
	return 0, nil
}
