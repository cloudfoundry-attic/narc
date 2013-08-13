package narc

import (
	. "launchpad.net/gocheck"
)

type RSuite struct{}

func init() {
	Suite(&RSuite{})
}

func (s *RSuite) TestRegistryCRUD(c *C) {
	registry := NewRegistry()

	task1 := &Task{}
	task2 := &Task{}

	registry.Register("123", task1)

	sess, ok := registry.Lookup("123")
	c.Assert(ok, Equals, true)
	c.Assert(sess, Equals, task1)

	registry.Unregister("123")

	sess, ok = registry.Lookup("123")
	c.Assert(ok, Equals, false)

	registry.Register("123", task1)
	registry.Register("123", task2)

	sess, ok = registry.Lookup("123")
	c.Assert(sess, Equals, task2)
	c.Assert(sess, Not(Equals), task1)
}
