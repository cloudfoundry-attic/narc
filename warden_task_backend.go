package narc

import (
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/gordon"
)

type WardenTaskBackend struct {
	WardenContainersPath string
	WardenSocketPath     string
}

func (p WardenTaskBackend) ProvideContainer() (Container, error) {
	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: p.WardenSocketPath,
		},
	)

	err := client.Connect()
	if err != nil {
		return nil, err
	}

	return NewWardenContainer(client)
}

func (p WardenTaskBackend) ProvideCommand(container Container) *exec.Cmd {
	wshBin := fmt.Sprintf(
		"%s/%s/bin/wsh",
		p.WardenContainersPath,
		container.ID(),
	)

	wshdSocket := fmt.Sprintf(
		"%s/%s/run/wshd.sock",
		p.WardenContainersPath,
		container.ID(),
	)

	return exec.Command(
		"sudo", wshBin,
		"--socket", wshdSocket,
		"--user", "vcap",
	)
}
