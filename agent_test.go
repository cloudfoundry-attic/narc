package sshark

import (
	. "launchpad.net/gocheck"
)

type ASuite struct{}

func init() {
	Suite(&ASuite{})
}

func (s *ASuite) TestAgentSessionLifecycle(c *C) {
	agent := NewAgent("/tmp/warden.sock")

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
	agent := NewAgent("/tmp/warden.sock")
	err := agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "session not registered")
}

func (s *ASuite) TestAgentTeardownAlreadyDestroyedContainer(c *C) {
	agent := NewAgent("/tmp/warden.sock")

	session, err := agent.StartSession("some-guid")
	c.Assert(err, IsNil)

	err = session.Container.Destroy()
	c.Assert(err, IsNil)

	err = agent.StopSession("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown handle")
}
