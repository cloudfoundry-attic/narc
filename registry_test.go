package sshark

import (
	. "launchpad.net/gocheck"
)

type RSuite struct{}

func init() {
	Suite(&RSuite{})
}

func (s *RSuite) TestRegisterCRUD(c *C) {
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

func (s *RSuite) TestRegistryMarshalling(c *C) {
	registry := NewRegistry()

	session := &Session{
		Container: &FakeContainer{Handle: "to-s-32"},
		Port:      MappedPort(1111),
	}

	registry.Register("abc", session)

	json, err := registry.MarshalJSON()
	c.Assert(err, IsNil)

	c.Assert(
		string(json),
		Equals,
		`{"sessions":{"abc":{"port":1111,"container":"to-s-32"}}}`,
	)
}
