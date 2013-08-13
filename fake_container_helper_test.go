package narc

import (
	"errors"
	"sync"
)

type FakeContainerProvider struct{}

func (p FakeContainerProvider) ProvideContainer() (Container, error) {
	return &FakeContainer{}, nil
}

type FakeContainer struct {
	Handle      string
	LastCommand string
	ShouldError bool

	LimitedMemory *uint64
	LimitedDisk   *uint64

	destroyed bool

	sync.RWMutex
}

func (c *FakeContainer) ID() string {
	return c.Handle
}

func (c *FakeContainer) Destroy() error {
	c.Lock()
	defer c.Unlock()

	c.destroyed = true

	return nil
}

func (c *FakeContainer) IsDestroyed() bool {
	c.RLock()
	defer c.RUnlock()

	return c.destroyed
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
