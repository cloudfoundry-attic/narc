package sshark

import (
	"github.com/vito/gordon"
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
