package sshark

import (
	. "launchpad.net/gocheck"
)

type SSuite struct{}

func init() {
	Suite(&SSuite{})
}

func (s *SSuite) TestRegisterCRUD(c *C) {
	registry := NewRegistry()
	session := &Session{}

	session2 := &Session{}

	registry.Register("123", session)

	sess, ok := registry.Lookup("123")
	c.Assert(ok, Equals, true)
	c.Assert(sess, Equals, session)

	registry.Unregister("123")

	sess, ok = registry.Lookup("123")
	c.Assert(ok, Equals, false)

	registry.Register("123", session)
	registry.Register("123", session2)

	sess, ok = registry.Lookup("123")
	c.Assert(sess, Equals, session2)
	c.Assert(sess, Not(Equals), session)
}
