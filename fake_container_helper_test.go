package narc

import (
	"errors"
)

type FakeContainer struct {
	LastCommand string
	ShouldError bool
	Handle      string

	LimitedMemory *uint64
	LimitedDisk   *uint64
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

func (c *FakeContainer) LimitMemory(limit uint64) error {
	c.LimitedMemory = &limit
	return nil
}

func (c *FakeContainer) LimitDisk(limit uint64) error {
	c.LimitedDisk = &limit
	return nil
}

func (c *FakeContainer) Info() (*ContainerInfo, error) {
	return &ContainerInfo{}, nil
}
