package narc

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"errors"
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

	task, err := agent.StartTask("abc", "some-token", TaskLimits{})
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

	prompt, err := readWithTimeout(reader, '$', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))
}

func (s *PSSuite) TestProxyServerForwardsInput(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := readWithTimeout(reader, '$', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Write([]byte("echo hi\n"))

	hiInput, err := readWithTimeout(reader, '\n', 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(hiInput), Equals, " echo hi\r\n")

	hi, err := readWithTimeout(reader, '\n', 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(hi), Equals, "hi\r\n")
}

func (s *PSSuite) TestProxyServerDestroysContainerWhenProcessEnds(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := readWithTimeout(reader, '$', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Write([]byte("exit\n"))

	hiInput, err := readWithTimeout(reader, '\n', 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(hiInput), Equals, " exit\r\n")

	time.Sleep(1 * time.Second)

	_, err = s.task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerKeepsContainerOnDisconnect(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := readWithTimeout(reader, '$', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Close()

	time.Sleep(1 * time.Second)

	res, err := s.task.Container.Run("exit 42")
	c.Assert(err, IsNil)

	c.Assert(res.ExitStatus, Equals, uint32(42))
}

func (s *PSSuite) TestProxyServerAttachesToRunningProcess(c *C) {
	writer, reader := s.connectedWebSocket(c)

	prompt, err := readWithTimeout(reader, '$', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(prompt), Equals, fmt.Sprintf("vcap@%s:~$", s.task.Container.ID()))

	writer.Write([]byte("ruby -e 'a = 0; while true; sleep 1; a += 1; p a; end'\n"))

	loopInput, err := readWithTimeout(reader, '\n', 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(loopInput), Matches, " ruby -e.*\r\n")

	nextNumber, err := readWithTimeout(reader, '\n', 5*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(nextNumber), Equals, "1\r\n")

	writer.Close()

	time.Sleep(1 * time.Second)

	writer, reader = s.connectedWebSocket(c)

	nextNumber, err = readWithTimeout(reader, '\n', 5*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(nextNumber), Equals, "3\r\n") // TODO: just check that it's a higher number
}

func (s *PSSuite) TestProxyServerRejectsInvalidToken(c *C) {
	config, err := websocket.NewConfig("ws://localhost:7331", "http://localhost")
	config.Header.Add("X-Task-ID", "abc")
	config.Header.Add("X-Task-Token", "some-bogus-token")

	c.Assert(err, IsNil)

	ws, err := websocket.DialConfig(config)
	c.Assert(err, IsNil)

	reader := bufio.NewReader(ws)

	errorMessage, err := readWithTimeout(reader, '\n', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(errorMessage), Equals, "Invalid Token\n")

	_, err = reader.ReadByte()
	c.Assert(err, NotNil)
}

func (s *PSSuite) TestProxyServerRejectsUnknownTask(c *C) {
	config, err := websocket.NewConfig("ws://localhost:7331", "http://localhost")
	config.Header.Add("X-Task-ID", "def")

	c.Assert(err, IsNil)

	ws, err := websocket.DialConfig(config)
	c.Assert(err, IsNil)

	reader := bufio.NewReader(ws)

	errorMessage, err := readWithTimeout(reader, '\n', 1*time.Second)
	c.Assert(err, IsNil)
	c.Assert(string(errorMessage), Equals, "Unknown Task\n")

	_, err = reader.ReadByte()
	c.Assert(err, NotNil)
}

func (s *PSSuite) connectedWebSocket(c *C) (*websocket.Conn, *bufio.Reader) {
	config, err := websocket.NewConfig("ws://localhost:7331", "http://localhost")
	config.Header.Add("X-Task-ID", "abc")
	config.Header.Add("X-Task-Token", "some-token")

	c.Assert(err, IsNil)

	ws, err := websocket.DialConfig(config)
	c.Assert(err, IsNil)

	return ws, bufio.NewReader(ws)
}

func readWithTimeout(reader *bufio.Reader, delim byte, timeout time.Duration) ([]byte, error) {
	readResult := make(chan []byte)
	errChannel := make(chan error)

	go func() {
		res, err := reader.ReadBytes(delim)
		if err != nil {
			errChannel <- err
		}

		readResult <- res
	}()

	select {
	case err := <-errChannel:
		return nil, err
	case res := <-readResult:
		return res, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout!")
	}
}
