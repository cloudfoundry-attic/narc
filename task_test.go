package narc

import (
	ex "bitbucket.org/teythoon/expect"
	"code.google.com/p/go.crypto/ssh"
	. "launchpad.net/gocheck"
	"os/exec"
	"syscall"
	"time"
)

type TSuite struct{}

func init() {
	Suite(&TSuite{})
}

type ptyRequestMessage struct {
	term          string
	columns       uint32
	rows          uint32
	widthPixels   uint32
	heightPixels  uint32
	terminalModes string
}

type windowChangeMessage struct {
	columns      uint32
	rows         uint32
	widthPixels  uint32
	heightPixels uint32
}

func (s *TSuite) TestTaskRedirectsStdout(c *C) {
	task := &Task{Command: exec.Command("echo", "hi")}

	channel := NewFakeChannel([]ssh.ChannelRequest{})

	reader := ex.New(bogusWriteCloser{channel.readPipe}, nil, 1*time.Second)

	done, closed, err := task.Attach(channel)
	c.Assert(err, IsNil)

	expect(c, reader, `hi\r\n`)

	<-done
	<-closed
}

func (s *TSuite) TestTaskRedirectsStderr(c *C) {
	task := &Task{Command: exec.Command("ruby", "-e", `$stderr.puts "hi"`)}

	channel := NewFakeChannel([]ssh.ChannelRequest{})

	reader := ex.New(bogusWriteCloser{channel.readPipe}, nil, 1*time.Second)

	done, closed, err := task.Attach(channel)
	c.Assert(err, IsNil)

	expect(c, reader, `hi\r\n`)

	<-done
	<-closed
}

func (s *TSuite) TestTaskAcceptsPTYRequests(c *C) {
	task := &Task{
		Command: exec.Command(
			"bash", "-c", "echo hello; sleep 1; tput cols; tput lines",
		),
	}

	channel := NewFakeChannel(
		[]ssh.ChannelRequest{
			ssh.ChannelRequest{
				Request:   "pty-req",
				WantReply: true,
				Payload: marshal(ptyRequestMessage{
					term:    "xterm",
					columns: 100,
					rows:    50,
				}),
			},
		},
	)

	reader := ex.New(bogusWriteCloser{channel.readPipe}, nil, 1*time.Second)

	done, closed, err := task.Attach(channel)
	c.Assert(err, IsNil)

	time.Sleep(100 * time.Millisecond)

	c.Assert(channel.Acked, Equals, true)

	expect(c, reader, `hello\r\n`)
	expect(c, reader, `100\r\n`)
	expect(c, reader, `50\r\n`)

	<-done
	<-closed
}

func (s *TSuite) TestTaskAcceptsWindowChange(c *C) {
	task := &Task{
		Command: exec.Command(
			"bash", "-c", "echo hello; sleep 1; tput cols; tput lines",
		),
	}

	channel := NewFakeChannel(
		[]ssh.ChannelRequest{
			ssh.ChannelRequest{
				Request:   "window-change",
				WantReply: true,
				Payload: marshal(windowChangeMessage{
					columns: 100,
					rows:    50,
				}),
			},
		},
	)

	reader := ex.New(bogusWriteCloser{channel.readPipe}, nil, 1*time.Second)

	done, closed, err := task.Attach(channel)
	c.Assert(err, IsNil)

	time.Sleep(100 * time.Millisecond)

	c.Assert(channel.Acked, Equals, true)

	expect(c, reader, `hello\r\n`)
	expect(c, reader, `100\r\n`)
	expect(c, reader, `50\r\n`)

	<-done
	<-closed
}

func (s *TSuite) TestTaskReportsCompletion(c *C) {
	task := &Task{
		Command: exec.Command("bash", "-c", "exit 42"),
	}

	channel := NewFakeChannel(
		[]ssh.ChannelRequest{
			ssh.ChannelRequest{
				Request:   "window-change",
				WantReply: true,
				Payload: marshal(windowChangeMessage{
					columns: 100,
					rows:    50,
				}),
			},
		},
	)

	done, closed, err := task.Attach(channel)
	c.Assert(err, IsNil)

	status := <-done
	c.Assert(status.Sys().(syscall.WaitStatus).ExitStatus(), Equals, 42)

	<-closed
}
