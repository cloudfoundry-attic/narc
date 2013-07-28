package sshark

import (
	"fmt"
	. "launchpad.net/gocheck"
	"time"
)

type ASuite struct{}

func init() {
	Suite(&ASuite{})
}

func (s *ASuite) TestAgentSessionLifecycle(c *C) {
	agent, err := NewAgent("/tmp/warden.sock")
	c.Assert(err, IsNil)

	session, err := agent.StartSession("some-guid")
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

func (s *ASuite) TestAgentTeardownNotExistantContainer(c *C) {
	agent, err := NewAgent("/tmp/warden.sock")
	c.Assert(err, IsNil)

	err = agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "session not registered")
}

func (s *ASuite) TestAgentTeardownAlreadyDestroyedContainer(c *C) {
	agent, err := NewAgent("/tmp/warden.sock")
	c.Assert(err, IsNil)

	session, err := agent.StartSession("some-guid")
	c.Assert(err, IsNil)

	err = session.Container.Destroy()
	c.Assert(err, IsNil)

	err = agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown handle")
}

func (s *ASuite) TestAgentIDIsUnique(c *C) {
	agent1, err := NewAgent("")
	c.Assert(err, IsNil)

	agent2, err := NewAgent("")
	c.Assert(err, IsNil)

	c.Assert(agent1.ID, Not(Equals), agent2.ID)
}

func (s *ASuite) TestAgentHandlesStartsAndStops(c *C) {
	mbus := NewMockMessageBus()

	agent, err := NewAgent("/tmp/warden.sock")
	c.Assert(err, IsNil)

	err = agent.HandleStarts(mbus)
	c.Assert(err, IsNil)

	err = agent.HandleStops(mbus)
	c.Assert(err, IsNil)

	directedStart := fmt.Sprintf("ssh.%s.start", agent.ID.String())
	mbus.Publish(directedStart, []byte(`
    {"session":"abc","public_key":"hello im a pubkey"}
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

func (s *RSuite) TestAgentMarshalling(c *C) {
	agent, err := NewAgent("/tmp/warden.sock")
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
