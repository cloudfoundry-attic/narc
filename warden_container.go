package narc

import (
	"github.com/cloudfoundry/gordon"
)

type WardenContainer struct {
	Handle string
	client *warden.Client
}

func NewWardenContainer(client *warden.Client) (*WardenContainer, error) {
	response, err := client.Create()
	if err != nil {
		return nil, err
	}

	return &WardenContainer{
		Handle: *response.Handle,
		client: client,
	}, nil
}

func (c *WardenContainer) ID() string {
	return c.Handle
}

func (c *WardenContainer) Destroy() error {
	_, err := c.client.Destroy(c.Handle)
	return err
}

func (c *WardenContainer) Run(script string) (*JobInfo, error) {
	res, err := c.client.Run(c.Handle, script)
	if err != nil {
		return nil, err
	}

	return &JobInfo{
		ExitStatus: res.GetExitStatus(),
	}, nil
}

func (c *WardenContainer) NetIn() (MappedPort, error) {
	res, err := c.client.NetIn(c.Handle)
	if err != nil {
		return 0, err
	}

	return MappedPort(res.GetHostPort()), nil
}

func (c *WardenContainer) LimitMemory(limit uint64) error {
	_, err := c.client.LimitMemory(c.Handle, limit)

	return err
}

func (c *WardenContainer) LimitDisk(limit uint64) error {
	_, err := c.client.LimitDisk(c.Handle, limit)

	return err
}

func (c *WardenContainer) Info() (*ContainerInfo, error) {
	info, err := c.client.Info(c.Handle)
	if err != nil {
		return nil, err
	}

	return &ContainerInfo{
		MemoryLimitInBytes: info.GetMemoryStat().GetHierarchicalMemoryLimit(),
	}, nil
}
