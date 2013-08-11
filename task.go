package narc

import (
	"code.google.com/p/go.crypto/ssh"
	"github.com/kr/pty"
	"io"
	"os"
	"os/exec"
)

type Task struct {
	Container   Container
	Limits      TaskLimits
	SecureToken string
	Command     *exec.Cmd

	pty *os.File
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

func (t *Task) Attach(channel ssh.Channel) (chan bool, chan error, error) {
	pty, err := t.run()
	if err != nil {
		return nil, nil, err
	}

	commandDone := make(chan bool)
	channelClosed := make(chan error)

	go func() {
		io.Copy(channel, pty)
		commandDone <- true
	}()

	go func() {
		err := t.handleChannelRequests(pty, channel)
		channelClosed <- err
	}()

	return commandDone, channelClosed, nil
}

func (t *Task) run() (*os.File, error) {
	if t.pty != nil {
		return t.pty, nil
	}

	pty, err := pty.Start(t.Command)
	if err != nil {
		return nil, err
	}

	t.pty = pty

	return t.pty, nil
}

func (t *Task) handleChannelRequests(pty *os.File, channel ssh.Channel) error {
	for {
		_, err := io.Copy(pty, channel)
		if err == nil {
			return err
		}

		req, ok := err.(ssh.ChannelRequest)
		if !ok {
			return err
		}

		ok = false
		switch req.Request {
		case "pty-req":
			ok = t.handlePtyRequest(req.Payload)

		case "shell":
			ok = true

		case "window-change":
			ok = t.handleWindowChange(req.Payload)

		case "env":
			ok = true
		}

		if req.WantReply {
			channel.AckRequest(ok)
		}
	}

	panic("unreachable")
}

func (t *Task) handlePtyRequest(payload []byte) bool {
	cols, rows, ok := parsePtyRequest(payload)
	if !ok {
		return false
	}

	err := setWinSize(t.pty, cols, rows)
	if err != nil {
		return false
	}

	return true
}

func (t *Task) handleWindowChange(payload []byte) (ok bool) {
	cols, rows, ok := parseWindowChange(payload)
	if !ok {
		return
	}

	err := setWinSize(t.pty, cols, rows)
	if err != nil {
		ok = false
	}

	return
}
