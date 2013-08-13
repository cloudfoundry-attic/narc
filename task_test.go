package narc

import (
	"code.google.com/p/go.crypto/ssh"
	. "launchpad.net/gocheck"
	"os"
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
	container := &FakeContainer{}
	task, _ := NewTask(container, "floofy_flubber", exec.Command("echo", "hi"))

	channel := NewFakeChannel([]ssh.ChannelRequest{})

	reader := NewExpector(channel.readPipe, 1*time.Second)

	err := task.Attach(channel)
	c.Assert(err, IsNil)

	expect(c, reader, `hi\r\n`)
}

func (s *TSuite) TestTaskRedirectsStderr(c *C) {
	container := &FakeContainer{}
	task, _ := NewTask(
		container,
		"floofy_flubber",
		exec.Command("ruby", "-e", `$stderr.puts "hi"`),
	)

	channel := NewFakeChannel([]ssh.ChannelRequest{})

	reader := NewExpector(channel.readPipe, 1*time.Second)

	err := task.Attach(channel)
	c.Assert(err, IsNil)

	expect(c, reader, `hi\r\n`)
}

func (s *TSuite) TestTaskAcceptsPTYRequests(c *C) {
	container := &FakeContainer{}
	task, _ := NewTask(
		container,
		"floofy_flubber",
		exec.Command(
			"bash", "-c", "echo hello; sleep 1; tput cols; tput lines",
		),
	)

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

	reader := NewExpector(channel.readPipe, 5*time.Second)

	err := task.Attach(channel)
	c.Assert(err, IsNil)

	time.Sleep(100 * time.Millisecond)

	c.Assert(channel.Acked, Equals, true)

	expect(c, reader, `hello\r\n`)
	expect(c, reader, `100\r\n`)
	expect(c, reader, `50\r\n`)
}

func (s *TSuite) TestTaskAcceptsWindowChange(c *C) {
	container := &FakeContainer{}
	task, _ := NewTask(
		container,
		"floofy_flubber",
		exec.Command(
			"bash", "-c", "echo hello; sleep 1; tput cols; tput lines",
		),
	)

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

	reader := NewExpector(channel.readPipe, 1*time.Second)

	err := task.Attach(channel)
	c.Assert(err, IsNil)

	time.Sleep(100 * time.Millisecond)

	c.Assert(channel.Acked, Equals, true)

	expect(c, reader, `hello\r\n`)
	expect(c, reader, `100\r\n`)
	expect(c, reader, `50\r\n`)
}

func (s *TSuite) TestTaskReportsCompletion(c *C) {
	container := &FakeContainer{}
	task, _ := NewTask(
		container,
		"floofy_flubber",
		exec.Command("bash", "-c", "exit 42"),
	)

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

	done := make(chan *os.ProcessState)

	c.Assert(task.ProcessState, IsNil)

	task.OnComplete(func() { done <- task.ProcessState })

	err := task.Attach(channel)
	c.Assert(err, IsNil)

	select {
	case status := <-done:
		c.Assert(status.Sys().(syscall.WaitStatus).ExitStatus(), Equals, 42)

	case <-time.After(1 * time.Second):
		c.Error("Was not notified of task completion!")
	}
}

func (s *TSuite) TestTaskStopDestroysContainer(c *C) {
	container := &FakeContainer{}

	task, _ := NewTask(container, "floofy_flubber", exec.Command("ls"))

	c.Assert(container.Destroyed, Equals, false)

	task.Stop()

	c.Assert(container.Destroyed, Equals, true)
}

func (s *TSuite) TestTaskCompletionDestroysContainer(c *C) {
	container := &FakeContainer{}

	task, _ := NewTask(container, "floofy_flubber", exec.Command("bash", "-c", "exit 0"))

	c.Assert(container.Destroyed, Equals, false)

	called := make(chan bool)

	task.OnComplete(func() { called <- true })

	_, _, err := task.Start()
	c.Assert(err, IsNil)

	select {
	case <-called:
		c.Assert(container.Destroyed, Equals, true)
	case <-time.After(1 * time.Second):
		c.Error("Was not notified of task completion!")
	}
}

func (s *TSuite) TestTaskStopReportsCompletionOfStartedTasks(c *C) {
	container := &FakeContainer{}

	task, _ := NewTask(container, "floofy_flubber", exec.Command("sleep", "100"))

	called := make(chan bool)

	task.OnComplete(func() { called <- true })

	task.Start()
	task.Stop()

	select {
	case <-called:
	case <-time.After(1 * time.Second):
		c.Error("Was not notified of task completion!")
	}
}

func (s *TSuite) TestTaskStopDoesNotReportCompletionOfUnstartedTasks(c *C) {
	container := &FakeContainer{}

	task, _ := NewTask(container, "floofy_flubber", exec.Command("sleep", "100"))

	called := make(chan bool)

	task.OnComplete(func() { called <- true })

	task.Stop()

	select {
	case <-called:
		c.Error("Was notified of task completion!")
	case <-time.After(100 * time.Millisecond):
	}
}
