package narc

import (
	"errors"
	"os/exec"
	"sync"
)

type FakeTaskBackend struct {
	Command *exec.Cmd
}

func (b FakeTaskBackend) ProvideContainer(limits TaskLimits) (Container, error) {
	return &FakeContainer{
		LimitedDisk:   &limits.DiskLimitInBytes,
		LimitedMemory: &limits.MemoryLimitInBytes,
	}, nil
}

func (b FakeTaskBackend) ProvideCommand(container Container) *exec.Cmd {
	if b.Command != nil {
		return b.Command
	}

	return exec.Command("bash", "-c", "read foo; echo $foo")
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
