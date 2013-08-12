package narc

import (
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"github.com/kr/pty"
	"github.com/nu7hatch/gouuid"
	"io"
	. "launchpad.net/gocheck"
	"os/exec"
	"time"
)

type PSSuite struct {
	ProxyServer *ProxyServer

	serverPort int

	agent    *Agent
	registry *Registry

	task   *Task
	taskID string
}

func init() {
	Suite(&PSSuite{})
}

func (s *PSSuite) SetUpTest(c *C) {
	agent, err := NewAgent(AgentConfig{
		WardenSocketPath: "/tmp/warden.sock",
	})
	if err != nil {
		panic(err)
	}

	s.agent = agent

	taskUUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	taskToken, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	s.taskID = taskUUID.String()

	task, err := agent.StartTask(s.taskID, taskToken.String(), TaskLimits{})
	if err != nil {
		panic(err)
	}

	s.task = task

	s.ProxyServer, err = NewProxyServer(agent)
	if err != nil {
		panic(err)
	}

	randomPort, err := grabEphemeralPort()
	if err != nil {
		randomPort = 7331
	}

	s.serverPort = randomPort

	err = s.ProxyServer.Start(randomPort)
	if err != nil {
		panic(err)
	}

	err = waitForPort(randomPort)
	if err != nil {
		panic(err)
	}
}

func (s *PSSuite) TearDownTest(c *C) {
	if s.task.Command.Process != nil {
		s.task.Command.Process.Kill()
	}

	s.agent.StopTask(s.taskID)
	s.ProxyServer.Stop()
}

func (s *PSSuite) TestProxyServerForwardsOutput(c *C) {
	_, _, reader := s.connectedTask(c)
	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))
}

func (s *PSSuite) TestProxyServerForwardsInput(c *C) {
	_, writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("echo hi\n"))

	expect(c, reader, ` echo hi\r\n`)
	expect(c, reader, `hi\r\n`)
}

func (s *PSSuite) TestProxyServerDestroysContainerWhenProcessEnds(c *C) {
	_, writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("exit\n"))

	expect(c, reader, ` exit\r\n`)

	time.Sleep(1 * time.Second)

	_, err := s.task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerDestroysContainerWhenProcessEndsWhileDetached(c *C) {
	client, writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("sleep 1; exit\n"))

	expect(c, reader, ` sleep 1; exit\r\n`)

	client.Process.Kill()

	time.Sleep(2 * time.Second)

	_, err := s.task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerKeepsContainerOnDisconnect(c *C) {
	_, writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Close()

	time.Sleep(1 * time.Second)

	res, err := s.task.Container.Run("exit 42")
	c.Assert(err, IsNil)

	c.Assert(res.ExitStatus, Equals, uint32(42))
}

func (s *PSSuite) TestProxyServerAttachesToRunningProcess(c *C) {
	client, writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte(`ruby -e '
Thread.new do
  while true
    puts "---"
    sleep 0.5
  end
end

while true
  puts gets.upcase
end'`))

	writer.Write([]byte("\n"))

	expect(c, reader, `---\r\n`)

	writer.Write([]byte("hello\n"))
	expect(c, reader, `HELLO\r\n`)

	client.Process.Kill()

	_, writer, reader = s.connectedTask(c)

	expect(c, reader, `---\r\n`)

	writer.Write([]byte("hello again\n"))
	expect(c, reader, `HELLO AGAIN\r\n`)
}

func (s *PSSuite) TestProxyServerRejectsInvalidToken(c *C) {
	config := &ssh.ClientConfig{
		User: s.taskID,
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(PasswordAuth{"some-bogus-token"}),
		},
	}

	_, err := ssh.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", s.serverPort), config)
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerRejectsUnknownTask(c *C) {
	config := &ssh.ClientConfig{
		User: "some-bogus-task-id",
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(PasswordAuth{s.task.SecureToken}),
		},
	}

	_, err := ssh.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", s.serverPort), config)
	c.Assert(err, NotNil)
}

func (s *PSSuite) connectedTask(c *C) (*exec.Cmd, io.WriteCloser, *Expector) {
	sshCmd := exec.Command(
		"ssh",
		"127.0.0.1",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-l", s.taskID,
		"-p", fmt.Sprintf("%d", s.serverPort),
	)

	pty, err := pty.Start(sshCmd)
	c.Assert(err, IsNil)

	// just so there's something sane
	setWinSize(pty, 80, 24)

	reader := NewExpector(pty, 5*time.Second)

	expect(c, reader, "password:")
	pty.Write([]byte(fmt.Sprintf("%s\n", s.task.SecureToken)))

	return sshCmd, pty, reader
}

type PasswordAuth struct {
	password string
}

func (p PasswordAuth) Password(user string) (string, error) {
	return p.password, nil
}
