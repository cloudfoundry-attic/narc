package narc

import (
	"code.google.com/p/go.crypto/ssh"
	"github.com/kr/pty"
	"io"
	"os"
	"os/exec"
)

type Task struct {
	SecureToken  string
	ProcessState *os.ProcessState

	container Container
	command   *exec.Cmd

	onCompleteCallbacks []func()

	pty *os.File
}

func NewTask(container Container, secureToken string, command *exec.Cmd) (*Task, error) {
	return &Task{
		SecureToken: secureToken,

		container: container,
		command:   command,
	}, nil
}

func (t *Task) Start() (io.Writer, io.Reader, error) {
	if t.pty == nil {
		pty, err := pty.Start(t.command)
		if err != nil {
			return nil, nil, err
		}

		t.pty = pty

		go t.reportExit()
	}

	return t.pty, t.pty, nil
}

func (t *Task) Attach(channel ssh.Channel) error {
	in, out, err := t.Start()
	if err != nil {
		return err
	}

	go io.Copy(channel, out)
	go t.handleChannelRequests(in, channel)

	return nil
}

func (t *Task) Stop() error {
	if t.command.Process != nil {
		t.command.Process.Kill()
	}

	return t.container.Destroy()
}

func (t *Task) OnComplete(callback func()) {
	t.onCompleteCallbacks = append(t.onCompleteCallbacks, callback)
}

func (t *Task) reportExit() {
	t.command.Wait()
	t.ProcessState = t.command.ProcessState

	t.container.Destroy()

	for _, callback := range t.onCompleteCallbacks {
		go callback()
	}
}

func (t *Task) handleChannelRequests(in io.Writer, channel ssh.Channel) error {
	for {
		_, err := io.Copy(in, channel)
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
