package narc

import (
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
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
	}
}

func (s *ASuite) TestAgentTaskLifecycle(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	task, err := agent.StartTask(
		"some-guid",
		TaskLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	_, found := agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, true)

	c.Assert(task.Container, NotNil)

	result, err := task.Container.Run("exit 42")
	c.Assert(err, IsNil)
	c.Assert(result.ExitStatus, Equals, uint32(42))

	err = agent.StopTask("some-guid")
	c.Assert(err, IsNil)

	_, found = agent.Registry.Lookup("some-guid")
	c.Assert(found, Equals, false)

	_, err = task.Container.Run("")
	c.Assert(err, NotNil)
}

func (s *ASuite) TestAgentTaskMemoryLimitsMakesTaskDie(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	task, err := agent.StartTask(
		"some-guid",
		TaskLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	allocate, err := task.Container.Run(`ruby -e "'a'*10*1024*1024"`)
	c.Assert(err, IsNil)
	c.Assert(allocate.ExitStatus, Equals, uint32(0))

	allocate, err = task.Container.Run(`ruby -e "'a'*33*1024*1024"`)
	c.Assert(err, IsNil)
	c.Assert(allocate.ExitStatus, Equals, uint32(137)) // via kill -9
}

func (s *ASuite) TestAgentTaskDiskLimitsEnforcesQuota(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	task, err := agent.StartTask(
		"some-guid",
		TaskLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
			DiskLimitInBytes:   uint64(128 * 1024),
		},
	)

	defer agent.StopTask("some-guid")

	c.Assert(err, IsNil)

	allocate64, err := task.Container.Run(`ruby -e "print('a' * 1024 * 64)" > foo.txt`)
	c.Assert(err, IsNil)
	c.Assert(allocate64.ExitStatus, Equals, uint32(0))

	checkSize, err := task.Container.Run(`test $(du foo.txt | awk '{print $1}') = 64`)
	c.Assert(err, IsNil)
	c.Assert(checkSize.ExitStatus, Equals, uint32(0))

	allocate256, err := task.Container.Run(`ruby -e "print('a' * 1024 * 256)" > foo.txt`)
	c.Assert(err, IsNil)
	c.Assert(allocate256.ExitStatus, Equals, uint32(1))
}

func (s *ASuite) TestAgentTeardownNotExistantContainer(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	err = agent.StopTask("some-guid")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "task not registered")
}

func (s *ASuite) TestAgentTeardownAlreadyDestroyedContainer(c *C) {
	agent, err := NewAgent(s.Config)
	c.Assert(err, IsNil)

	task, err := agent.StartTask(
		"some-guid",
		TaskLimits{
			MemoryLimitInBytes: uint64(32 * 1024 * 1024),
		},
	)

	c.Assert(err, IsNil)

	err = task.Container.Destroy()
	c.Assert(err, IsNil)

	err = agent.StopTask("some-guid")
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

	mbus.Publish("task.start", []byte(`
	    {"task":"abc","memory_limit":128,"disk_limit":1}
	`))

	// give agent time to set everything up
	//
	// TODO: less lazy solution
	time.Sleep(1 * time.Second)

	sess, found := agent.Registry.Lookup("abc")
	c.Assert(found, Equals, true)

	info, err := sess.Container.Info()
	c.Assert(err, IsNil)
	c.Assert(info.MemoryLimitInBytes, Equals, uint64(128*1024*1024))

	canWriteTooBig, err := sess.Container.Run(`ruby -e 'print("a" * 1024 * 1024 * 2)' > foo`)
	c.Assert(err, IsNil)
	c.Assert(canWriteTooBig.ExitStatus, Equals, uint32(1))

	mbus.Publish("task.stop", []byte(`{"task":"abc"}`))

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
