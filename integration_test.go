package narc

import (
	. "launchpad.net/gocheck"
)

type WCISuite struct{}

func init() {
	Suite(&WCISuite{})
}

func (w *WCISuite) TestNewWardenContainerSuccessEndToEnd(c *C) {
	wardenContainer, err := NewWardenContainer("/tmp/warden.sock",
		TaskLimits{MemoryLimitInBytes: 4, DiskLimitInBytes: 5}, &ContainerCreationRunnerInJson{})
	_, err = wardenContainer.Run("ls")
	c.Assert(err, IsNil)
	err = wardenContainer.Destroy()
	c.Assert(err, IsNil)
}

