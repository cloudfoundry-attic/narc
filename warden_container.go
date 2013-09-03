package narc

import (
	"github.com/cloudfoundry/gordon"
)

type WardenContainer struct {
	Handle           string
	client           *warden.Client
	wardenSocketPath string
}

func NewWardenContainer(wardenSocketPath string, limits TaskLimits, cmdRunner ContainerCreationRunner) (*WardenContainer, error) {
	message := CreateContainerMessage{
		WardenSocketPath: wardenSocketPath,
		MemoryLimit:      limits.MemoryLimitInBytes,
		DiskLimit:        limits.DiskLimitInBytes,
		Network:          true,
	}
	var response CreateContainerResponse
	err := cmdRunner.Run(&message, &response, "create_warden_container.sh")

	if err != nil {
		return nil, err
	}

	return &WardenContainer{
		Handle:           response.Handle,
		wardenSocketPath: wardenSocketPath,
	}, nil
}

func (c *WardenContainer) ID() string {
	return c.Handle
}

func (c *WardenContainer) Destroy() error {
	client, err := c.getClient()
	if err != nil {
		return err
	}
	_, err = client.Destroy(c.Handle)
	return err
}

func (c *WardenContainer) Run(script string) (*JobInfo, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}

	res, err := client.Run(c.Handle, script)
	if err != nil {
		return nil, err
	}

	return &JobInfo{
		ExitStatus: res.GetExitStatus(),
	}, nil
}

func (c *WardenContainer) getClient() (*warden.Client, error) {
	if c.client != nil {
		return c.client, nil
	}

	c.client = warden.NewClient(&warden.ConnectionInfo{SocketPath: c.wardenSocketPath})

	err := c.client.Connect()
	if err != nil {
		return nil, err
	}

	return c.client, nil
}
