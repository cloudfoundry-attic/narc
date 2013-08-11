package narc

import (
	ex "bitbucket.org/teythoon/expect"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io"
	. "launchpad.net/gocheck"
	"time"
)

type PSSuite struct {
	ProxyServer *ProxyServer

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

	err = s.ProxyServer.Start(7331)
	if err != nil {
		panic(err)
	}
}

func (s *PSSuite) TearDownTest(c *C) {
	s.agent.StopTask(s.taskID)
	s.ProxyServer.Stop()
}

func (s *PSSuite) TestProxyServerForwardsOutput(c *C) {
	_, reader := s.connectedTask(c)
	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))
}

func (s *PSSuite) TestProxyServerForwardsInput(c *C) {
	writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("echo hi\n"))

	expect(c, reader, ` echo hi\r\n`)
	expect(c, reader, `hi\r\n`)
}

func (s *PSSuite) TestProxyServerDestroysContainerWhenProcessEnds(c *C) {
	writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("exit\n"))

	expect(c, reader, ` exit\r\n`)

	time.Sleep(1 * time.Second)

	_, err := s.task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerKeepsContainerOnDisconnect(c *C) {
	writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Close()

	time.Sleep(1 * time.Second)

	res, err := s.task.Container.Run("exit 42")
	c.Assert(err, IsNil)

	c.Assert(res.ExitStatus, Equals, uint32(42))
}

func (s *PSSuite) TestProxyServerAttachesToRunningProcess(c *C) {
	writer, reader := s.connectedTask(c)

	expect(c, reader, fmt.Sprintf(`vcap@%s:~\$`, s.task.Container.ID()))

	writer.Write([]byte("ruby -e 'a = 9; 10.times { p a; sleep 1; a += 1; }'\n"))

	expect(c, reader, ` ruby -e '.*'\r\n`)
	expect(c, reader, `9\r\n`)

	writer.Close()

	time.Sleep(1 * time.Second)

	writer, reader = s.connectedTask(c)

	expect(c, reader, `\d{2}\r\n`)
}

func (s *PSSuite) TestProxyServerRejectsInvalidToken(c *C) {
	config := &ssh.ClientConfig{
		User: s.taskID,
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(PasswordAuth{"some-bogus-token"}),
		},
	}

	_, err := ssh.Dial("tcp", "localhost:7331", config)
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerRejectsUnknownTask(c *C) {
	config := &ssh.ClientConfig{
		User: "some-bogus-task-id",
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(PasswordAuth{s.task.SecureToken}),
		},
	}

	_, err := ssh.Dial("tcp", "localhost:7331", config)
	c.Assert(err, NotNil)
}

func (s *PSSuite) connectedTask(c *C) (io.WriteCloser, *ex.Reader) {
	config := &ssh.ClientConfig{
		User: s.taskID,
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(PasswordAuth{s.task.SecureToken}),
		},
	}

	client, err := ssh.Dial("tcp", "localhost:7331", config)
	c.Assert(err, IsNil)

	session, err := client.NewSession()
	c.Assert(err, IsNil)

	stdin, err := session.StdinPipe()
	c.Assert(err, IsNil)

	stdout, err := session.StdoutPipe()
	c.Assert(err, IsNil)

	reader := ex.New(bogusWriteCloser{stdout}, nil, 5*time.Second)

	return stdin, reader
}

type PasswordAuth struct {
	password string
}

func (p PasswordAuth) Password(user string) (string, error) {
	return p.password, nil
}
