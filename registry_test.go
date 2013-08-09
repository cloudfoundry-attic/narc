package narc

import (
	. "launchpad.net/gocheck"
)

type RSuite struct{}

func init() {
	Suite(&RSuite{})
}

func (s *RSuite) TestRegisterCRUD(c *C) {
	registry := NewRegistry()
	task := &Task{}

	task2 := &Task{}

	registry.Register("123", task)

	sess, ok := registry.Lookup("123")
	c.Assert(ok, Equals, true)
	c.Assert(sess, Equals, task)

	registry.Unregister("123")

	sess, ok = registry.Lookup("123")
	c.Assert(ok, Equals, false)

	registry.Register("123", task)
	registry.Register("123", task2)

	sess, ok = registry.Lookup("123")
	c.Assert(sess, Equals, task2)
	c.Assert(sess, Not(Equals), task)
}
