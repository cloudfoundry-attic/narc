package narc

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"fmt"
	. "launchpad.net/gocheck"
	"time"
)

type PSSuite struct {
	ProxyServer *ProxyServer

	agent    *Agent
	registry *Registry
	task     *Task
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

	task, err := agent.StartTask("abc", TaskLimits{})
	if err != nil {
		panic(err)
	}

	s.task = task

	s.ProxyServer = NewProxyServer(agent)

	err = s.ProxyServer.Start(7331)
	if err != nil {
		panic(err)
	}
}

func (s *PSSuite) TearDownTest(c *C) {
	s.agent.StopTask("abc")
	s.ProxyServer.Stop()
}

func (s *PSSuite) TestProxyServerForwardsOutput(c *C) {
	_, reader := s.connectedWebSocket(c)

	prompt, err := reader.ReadBytes('$')
	c.Assert(err, IsNil)

	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))
}

func (s *PSSuite) TestProxyServerForwardsInput(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := reader.ReadBytes('$')
	c.Assert(err, IsNil)

	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Write([]byte("echo hi\n"))

	hiInput, err := reader.ReadBytes('\n')
	c.Assert(err, IsNil)
	c.Assert(string(hiInput), Equals, " echo hi\r\n")

	hi, err := reader.ReadBytes('\n')
	c.Assert(err, IsNil)
	c.Assert(string(hi), Equals, "hi\r\n")
}

func (s *PSSuite) TestProxyServerDestroysContainerWhenProcessEnds(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := reader.ReadBytes('$')
	c.Assert(err, IsNil)

	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Write([]byte("exit\n"))

	hiInput, err := reader.ReadBytes('\n')
	c.Assert(err, IsNil)
	c.Assert(string(hiInput), Equals, " exit\r\n")

	time.Sleep(1 * time.Second)

	_, err = s.task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerKeepsContainerOnDisconnect(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := reader.ReadBytes('$')
	c.Assert(err, IsNil)

	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Close()

	time.Sleep(1 * time.Second)

	res, err := s.task.Container.Run("exit 42")
	c.Assert(err, IsNil)

	c.Assert(res.ExitStatus, Equals, uint32(42))
}

func (s *PSSuite) TestProxyServerAttachesToRunningProcess(c *C) {

}

func (s *PSSuite) TestProxyServerRejectsInvalidToken(c *C) {

}

func (s *PSSuite) connectedWebSocket(c *C) (*websocket.Conn, *bufio.Reader) {
	config, err := websocket.NewConfig("ws://localhost:7331", "http://localhost")
	config.Header.Add("X-Task-ID", "abc")

	c.Assert(err, IsNil)

	ws, err := websocket.DialConfig(config)
	c.Assert(err, IsNil)

	return ws, bufio.NewReader(ws)
}
