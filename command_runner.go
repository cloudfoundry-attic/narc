package narc

import (
	"encoding/json"
	"os/exec"
)

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

type ContainerCreationRunner interface {
	Run(request *CreateContainerMessage, response *CreateContainerResponse, cmd string) error
}

type ContainerCreationRunnerInJson struct {}

func (runner *ContainerCreationRunnerInJson) Run(request *CreateContainerMessage, response *CreateContainerResponse, executable string) error {
	cmd := exec.Command(executable)
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
