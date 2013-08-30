package narc

import (
	"encoding/json"
	"github.com/cloudfoundry/gordon"
	"os/exec"
)

type WardenContainer struct {
	Handle string
	client *warden.Client
}

type CreateContainerMessage struct {
	WardenSocketPath string `json:"warden_socket_path"`
	DiskLimit        uint64 `json:"disk_limit"`
	MemoryLimit      uint64 `json:"memory_limit"`
	Network          bool   `json:"network"`
}

type CreateContainerResponse struct {
	Handle               string `json:"handle"`
	HostPort             int    `json:"host_port"`
	ContainerPort        int    `json:"container_port"`
	ConsoleHostPort      int    `json:"console_host_port"`
	ConsoleContainerPort int    `json:"console_container_port"`
}

func RunWithJson(request interface{}, response interface{}, cmd *exec.Cmd) error {
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	err := cmd.Start()
	if err != nil {
		return err
	}

	json.NewEncoder(stdin).Encode(request)
	stdin.Close()

	err = json.NewDecoder(stdout).Decode(response)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func NewWardenContainer(wardenSocketPath string, limits TaskLimits) (*WardenContainer, error) {
	message := CreateContainerMessage{
		WardenSocketPath: wardenSocketPath,
		MemoryLimit:      limits.MemoryLimitInBytes,
		DiskLimit:        limits.DiskLimitInBytes,
		Network:          true,
	}
	var response CreateContainerResponse
	cmd := exec.Command("create_warden_container.sh")

	err := RunWithJson(&message, &response, cmd)

	if err != nil {
		return nil, err
	}

	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: wardenSocketPath,
		},
	)

	err = client.Connect()
	if err != nil {
		return nil, err
	}

	return &WardenContainer{
		Handle: response.Handle,
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
