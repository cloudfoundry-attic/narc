package narc

import (
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	. "launchpad.net/gocheck"
)

type ASuite struct {
	Agent      *Agent
	MessageBus *mock_cfmessagebus.MockMessageBus
}

func init() {
	Suite(&ASuite{})
}

func (s *ASuite) FakeContainerForGuid(c *C, guid string) *FakeContainer {
	task, found := s.Agent.Registry.Lookup(guid)
	c.Assert(found, Equals, true)

	container, ok := task.container.(*FakeContainer)
	c.Assert(ok, Equals, true)

	return container
}

func (s *ASuite) SetUpTest(c *C) {
	agent, err := NewAgent(FakeContainerProvider{})

	c.Assert(err, IsNil)

	s.Agent = agent
	s.MessageBus = mock_cfmessagebus.NewMockMessageBus()

	err = agent.HandleStarts(s.MessageBus)
	c.Assert(err, IsNil)

	err = agent.HandleStops(s.MessageBus)
	c.Assert(err, IsNil)
}

func (s *ASuite) TestAgentTaskCreationDoesDiskLimits(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":1,"disk_limit":32}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(*container.LimitedDisk, Equals, uint64(32*1024*1024))
}

func (s *ASuite) TestNewTaskDoesNotLimitDiskToZero(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":1}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(container.LimitedDisk, IsNil)
}

func (s *ASuite) TestAgentTaskCreationDoesMemoryLimits(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":3,"disk_limit":1}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(*container.LimitedMemory, Equals, uint64(3*1024*1024))
}

func (s *ASuite) TestNewTaskDoesNotLimitMemoryToZero(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","disk_limit":4}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(container.LimitedMemory, IsNil)
}

func (s *ASuite) TestAgentTaskLifecycle(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":32,"disk_limit":1}
	`))

	task, found := s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, true)

	c.Assert(task.SecureToken, Equals, "some-token")
	c.Assert(task.container, NotNil)

	container := s.FakeContainerForGuid(c, "some-guid")

	s.MessageBus.PublishSync("task.stop", []byte(`{"task":"some-guid"}`))

	_, found = s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, false)

	c.Assert(container.IsDestroyed(), Equals, true)
}

func (s *ASuite) TestAgentTeardownNotExistantContainer(c *C) {
	s.MessageBus.PublishSync("task.stop", []byte(`{"task":"abc"}`))

	_, found := s.Agent.Registry.Lookup("abc")
	c.Assert(found, Equals, false)

	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"abc","secure_token":"some-token","memory_limit":128,"disk_limit":1}
	`))

	_, found = s.Agent.Registry.Lookup("abc")
	c.Assert(found, Equals, true)
}

func (s *ASuite) TestAgentIDIsUnique(c *C) {
	agent1, err := NewAgent(WardenContainerProvider{})
	c.Assert(err, IsNil)

	agent2, err := NewAgent(WardenContainerProvider{})
	c.Assert(err, IsNil)

	c.Assert(agent1.ID, Not(Equals), agent2.ID)
}
