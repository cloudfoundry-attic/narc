package narc

import (
	"github.com/cloudfoundry/gibson/fake_router_client"
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	. "launchpad.net/gocheck"
	"time"
)

type ASuite struct {
	Agent *Agent

	RouterClient *fake_gibson.FakeRouterClient
	MessageBus   *mock_cfmessagebus.MockMessageBus
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
	s.RouterClient = fake_gibson.NewFakeRouterClient()

	agent, err := NewAgent(FakeTaskBackend{}, s.RouterClient, 42)
	c.Assert(err, IsNil)

	s.Agent = agent

	s.MessageBus = mock_cfmessagebus.NewMockMessageBus()

	err = agent.HandleStarts(s.MessageBus)
	c.Assert(err, IsNil)

	err = agent.HandleStops(s.MessageBus)
	c.Assert(err, IsNil)
}

func (s *ASuite) TestAgentIDIsUnique(c *C) {
	agent1, err := NewAgent(WardenTaskBackend{}, nil, 0)
	c.Assert(err, IsNil)

	agent2, err := NewAgent(WardenTaskBackend{}, nil, 0)
	c.Assert(err, IsNil)

	c.Assert(agent1.ID, Not(Equals), agent2.ID)
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

func (s *ASuite) TestAgentIgnoresDuplicateStarts(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":32,"disk_limit":1}
	`))

	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-other-token","memory_limit":32,"disk_limit":1}
	`))

	task, found := s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, true)

	c.Assert(task.SecureToken, Equals, "some-token")
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

func (s *ASuite) TestAgentUnregistersTaskOnCompletion(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":32,"disk_limit":1}
	`))

	task, found := s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, true)

	c.Assert(task.SecureToken, Equals, "some-token")
	c.Assert(task.container, NotNil)

	c.Assert(s.RouterClient.IsRegistered(42, "some-guid"), Equals, true)

	task.Start()
	task.Stop()

	removedFromRegistry := make(chan bool)

	go func() {
		for {
			_, found := s.Agent.Registry.Lookup("some-guid")
			if !found {
				removedFromRegistry <- true
			}

			time.Sleep(1 * time.Millisecond)
		}
	}()

	select {
	case <-removedFromRegistry:
		c.Assert(s.RouterClient.IsRegistered(42, "some-guid"), Equals, false)
	case <-time.After(1 * time.Second):
		c.Error("Task was not removed from the registry.")
	}
}

func (s *ASuite) TestAgentRegisterAndUnregistersTaskWithRouter(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":32,"disk_limit":1}
	`))

	c.Assert(s.RouterClient.IsRegistered(42, "some-guid"), Equals, true)

	s.MessageBus.PublishSync("task.stop", []byte(`{"task":"some-guid"}`))

	c.Assert(s.RouterClient.IsRegistered(42, "some-guid"), Equals, false)
}

func (s *ASuite) TestAgentTaskCreationDoesDiskLimits(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":1,"disk_limit":32}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(*container.LimitedDisk, Equals, uint64(32*1024*1024))
}

func (s *ASuite) TestAgentNewTaskDoesNotCreateATaskWhenNoDiskLimit(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":1}
	`))

	task, found := s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, false)
	c.Assert(task, IsNil)
}

func (s *ASuite) TestAgentTaskCreationDoesMemoryLimits(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","memory_limit":3,"disk_limit":1}
	`))

	container := s.FakeContainerForGuid(c, "some-guid")
	c.Assert(*container.LimitedMemory, Equals, uint64(3*1024*1024))
}

func (s *ASuite) TestAgentNewTaskDoesNotCreateATaskWhenNoMemoryLimit(c *C) {
	s.MessageBus.PublishSync("task.start", []byte(`
	    {"task":"some-guid","secure_token":"some-token","disk_limit":4}
	`))

	task, found := s.Agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, false)
	c.Assert(task, IsNil)
}
