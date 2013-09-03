package narc

import (
	"fmt"
	"os/exec"
)

type WardenTaskBackend struct {
	WardenContainersPath string
	WardenSocketPath     string
}

func (p WardenTaskBackend) ProvideContainer(limits TaskLimits) (Container, error) {
	return NewWardenContainer(p.WardenSocketPath, limits, &ContainerCreationRunnerInJson{})
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
