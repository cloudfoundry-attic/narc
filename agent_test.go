package sshark

import (
	"fmt"
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"time"
)

type ASuite struct {
	Config AgentConfig
}

func init() {
	Suite(&ASuite{})
}

func (s *ASuite) SetUpSuite(c *C) {
	s.Config = AgentConfig{
		WardenSocketPath: "/tmp/warden.sock",
		StateFilePath:    "agent-test-state.json",
	}
}

func (s *ASuite) TestAgentSessionLifecycle(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	session, err := agent.StartSession(
		"some-guid",
		SessionLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	_, found := agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, true)

	c.Assert(session.Container, NotNil)
	c.Assert(session.Port, Not(Equals), MappedPort(0))

	result, err := session.Container.Run("exit 42")
	c.Assert(err, IsNil)
	c.Assert(result.ExitStatus, Equals, uint32(42))

	err = agent.StopSession("some-guid")
	c.Assert(err, IsNil)

	_, found = agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, false)

	_, err = session.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *ASuite) TestAgentSessionMemoryLimitsMakesSessionDie(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	session, err := agent.StartSession(
		"some-guid",
		SessionLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	allocate, err := session.Container.Run(`ruby -e "'a'*10*1024*1024"`)
	c.Assert(err, IsNil)
	c.Assert(allocate.ExitStatus, Equals, uint32(0))

	allocate, err = session.Container.Run(`ruby -e "'a'*33*1024*1024"`)
	c.Assert(err, IsNil)
	c.Assert(allocate.ExitStatus, Equals, uint32(137)) // via kill -9
}

func (s *ASuite) TestAgentSessionDiskLimitsEnforcesQuota(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	session, err := agent.StartSession(
		"some-guid",
		SessionLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
			DiskLimitInBytes:   uint64(128 * 1024),
		},
	)

	c.Assert(err, IsNil)

	allocate64, err := session.Container.Run(`ruby -e "print('a' * 1024 * 64)" > foo.txt`)
	c.Assert(err, IsNil)
	c.Assert(allocate64.ExitStatus, Equals, uint32(0))

	checkSize, err := session.Container.Run(`test $(du foo.txt | awk '{print $1}') = 64`)
	c.Assert(err, IsNil)
	c.Assert(checkSize.ExitStatus, Equals, uint32(0))

	allocate256, err := session.Container.Run(`ruby -e "print('a' * 1024 * 256)" > foo.txt`)
	c.Assert(err, IsNil)
	c.Assert(allocate256.ExitStatus, Equals, uint32(1))
}

func (s *ASuite) TestAgentTeardownNotExistantContainer(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	err = agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "session not registered")
}

func (s *ASuite) TestAgentTeardownAlreadyDestroyedContainer(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	session, err := agent.StartSession(
		"some-guid",
		SessionLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	err = session.Container.Destroy()
	c.Assert(err, IsNil)

	err = agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown handle")
}

func (s *ASuite) TestAgentIDIsUnique(c *C) {
	agent1, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	agent2, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	c.Assert(agent1.ID, Not(Equals), agent2.ID)
}

func (s *ASuite) TestAgentHandlesStartsAndStops(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	err = agent.HandleStarts(mbus)
	c.Assert(err, IsNil)

	err = agent.HandleStops(mbus)
	c.Assert(err, IsNil)

	directedStart := fmt.Sprintf("ssh.%s.start", agent.ID.String())
	mbus.Publish(directedStart, []byte(`
	    {"session":"abc","public_key":"hello im a pubkey","memory_limit":128,"disk_limit":1}
	`))

	// give agent time to set everything up
	//
	// TODO: less lazy solution
	time.Sleep(1 * time.Second)

	sess, found := agent.Registry.Lookup("abc")
	c.Assert(found, Equals, true)

	hasPubkey, err := sess.Container.Run("cat ~/.ssh/authorized_keys")
	c.Assert(err, IsNil)
	c.Assert(hasPubkey.ExitStatus, Equals, uint32(0))

	checkPort := fmt.Sprintf("lsof -i :%d", sess.Port)
	runningServer, err := sess.Container.Run(checkPort)
	c.Assert(err, IsNil)
	c.Assert(runningServer.ExitStatus, Equals, uint32(0))

	info, err := sess.Container.Info()
	c.Assert(err, IsNil)
	c.Assert(info.MemoryLimitInBytes, Equals, uint64(128*1024*1024))

	canWriteTooBig, err := sess.Container.Run(`ruby -e 'print("a" * 1024 * 1024 * 2)' > foo`)
	c.Assert(err, IsNil)
	c.Assert(canWriteTooBig.ExitStatus, Equals, uint32(1))

	mbus.Publish("ssh.stop", []byte(`{"session":"abc"}`))

	// give agent time to set everything up
	//
	// TODO: less lazy solution
	time.Sleep(1 * time.Second)

	_, found = agent.Registry.Lookup("abc")
	c.Assert(found, Equals, false)

	_, err = sess.Container.Run("")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown handle")
}

// TODO: test for ssh.stop when backing container was destroyed out from
// under it

func (s *ASuite) TestAgentMarshalling(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	session1 := &Session{
		Container: &FakeContainer{Handle: "to-s-32"},
		Port:      MappedPort(1111),
	}

	session2 := &Session{
		Container: &FakeContainer{Handle: "to-s-64"},
		Port:      MappedPort(2222),
	}

	agent.Registry.Register("abc", session1)
	agent.Registry.Register("def", session2)

	json, err := agent.MarshalJSON()
	c.Assert(err, IsNil)

	c.Assert(
		string(json),
		Equals,
		fmt.Sprintf(`{"id":"%s","sessions":{"abc":{"container":"to-s-32","port":1111},"def":{"container":"to-s-64","port":2222}}}`, agent.ID.String()),
	)
}

func (s *ASuite) TestAgentStateSaving(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	state, err := ioutil.ReadFile(s.Config.StateFilePath)
	c.Assert(err, IsNil)

	c.Assert(
		string(state),
		Equals,
		fmt.Sprintf(
			`{"id":"%s","sessions":{}}`,
			agent.ID.String(),
		),
	)

	session, err := agent.StartSession(
		"abc",
		SessionLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)
	c.Assert(err, IsNil)

	state, err = ioutil.ReadFile(s.Config.StateFilePath)
	c.Assert(err, IsNil)

	c.Assert(
		string(state),
		Equals,
		fmt.Sprintf(
			`{"id":"%s","sessions":{"abc":{"container":"%s","port":%d}}}`,
			agent.ID.String(),
			session.Container.ID(),
			session.Port,
		),
	)

	err = agent.StopSession("abc")
	c.Assert(err, IsNil)

	state, err = ioutil.ReadFile(s.Config.StateFilePath)
	c.Assert(err, IsNil)

	c.Assert(
		string(state),
		Equals,
		fmt.Sprintf(`{"id":"%s","sessions":{}}`, agent.ID.String()),
	)
}

func (s *ASuite) TestAgentDisabledStateSaving(c *C) {
	noStateConfig := AgentConfig{
		WardenSocketPath: s.Config.WardenSocketPath,
		StateFilePath:    "",
	}

	_, err := NewAgent(noStateConfig)
	c.Assert(err, IsNil)
}
