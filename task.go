package narc

import (
	"fmt"
	"github.com/kr/pty"
	"os"
	"os/exec"
)

type Task struct {
	Container   Container
	Limits      TaskLimits
	SecureToken string

	pty *os.File
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

func (t *Task) Run() (*os.File, error) {
	if t.pty != nil {
		return t.pty, nil
	}

	wshdSocket := fmt.Sprintf(
		"/opt/warden/containers/%s/run/wshd.sock",
		t.Container.ID(),
	)

	c := exec.Command(
		"sudo",
		"/opt/warden/warden/root/linux/skeleton/bin/wsh",
		"--socket", wshdSocket,
		"--user", "vcap",
	)

	pty, err := pty.Start(c)
	if err != nil {
		return nil, err
	}

	t.pty = pty

	return t.pty, nil
}
