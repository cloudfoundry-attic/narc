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

	commandDone chan *os.ProcessState
	pty         *os.File
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

func NewTask(container Container, limits TaskLimits, secureToken string, command *exec.Cmd) (*Task, error) {
	if limits.MemoryLimitInBytes != 0 {
		err := container.LimitMemory(limits.MemoryLimitInBytes)
		if err != nil {
			return nil, err
		}
	}

	if limits.DiskLimitInBytes != 0 {
		err := container.LimitDisk(limits.DiskLimitInBytes)
		if err != nil {
			return nil, err
		}
	}

	return &Task{
		Container:   container,
		SecureToken: secureToken,
		Command:     command,
	}, nil
}

func (t *Task) Attach(channel ssh.Channel) (chan *os.ProcessState, chan error, error) {
	in, out, err := t.run()
	if err != nil {
		return nil, nil, err
	}

	channelClosed := make(chan error)

	go io.Copy(channel, out)

	go func() {
		err := t.handleChannelRequests(in, channel)
		channelClosed <- err
	}()

	return t.commandDone, channelClosed, nil
}

func (t *Task) run() (io.Writer, io.Reader, error) {
	if t.pty == nil {
		pty, err := pty.Start(t.Command)
		if err != nil {
			return nil, nil, err
		}

		t.pty = pty
		t.commandDone = t.reportExit()
	}

	return t.pty, t.pty, nil
}

func (t *Task) reportExit() chan *os.ProcessState {
	done := make(chan *os.ProcessState, 1)

	go func() {
		t.Command.Wait()
		done <- t.Command.ProcessState
	}()

	return done
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
