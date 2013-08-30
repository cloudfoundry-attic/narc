package narc

import (
	. "launchpad.net/gocheck"
)

type TLSuite struct {
}

func init() {
	Suite(&TLSuite{})
}

func (s *TLSuite) TestTaskLimitsValidWhenGreaterThanZero(c *C) {
	task := TaskLimits{MemoryLimitInBytes: 100, DiskLimitInBytes: 200}
	c.Assert(task.IsValid(), Equals, true)
}

func (s *TLSuite) TestTaskLimitsInvalidWhenMemoryIsNonPositive(c *C) {
	task := TaskLimits{MemoryLimitInBytes: 0, DiskLimitInBytes: 200}
	c.Assert(task.IsValid(), Equals, false)
}

func (s *TLSuite) TestTaskLimitsInvalidWhenDiskIsNonPositive(c *C) {
	task := TaskLimits{MemoryLimitInBytes: 999, DiskLimitInBytes: 0}
	c.Assert(task.IsValid(), Equals, false)
}
