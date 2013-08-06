package sshark

import (
	. "launchpad.net/gocheck"
)

type SSuite struct{}

func init() {
	Suite(&SSuite{})
}

func (s *SSuite) TestSessionLoadPublicKeySuccess(c *C) {
	container := &FakeContainer{}

	session := &Session{
		Container: container,
		Port:      MappedPort(123),
	}

	err := session.LoadPublicKey("super-secure")
	c.Assert(err, IsNil)

	c.Assert(container.LastCommand, Equals,
		"mkdir ~/.ssh; echo 'super-secure' >> ~/.ssh/authorized_keys",
	)
}

func (s *SSuite) TestSessionLoadPublicKeyFail(c *C) {
	container := &FakeContainer{
		ShouldError: true,
	}

	session := &Session{
		Container: container,
		Port:      MappedPort(123),
	}

	err := session.LoadPublicKey("super-secure")
	c.Assert(err, NotNil)
}

func (s *SSuite) TestStartSSHServerSuccess(c *C) {
	container := &FakeContainer{}

	session := &Session{
		Container: container,
		Port:      MappedPort(123),
	}

	err := session.StartSSHServer()
	c.Assert(err, IsNil)

	c.Assert(container.LastCommand, Matches, ".*ssh-keygen -t rsa -f .ssh/host_key.*")

	c.Assert(container.LastCommand, Matches,
		".*/usr/sbin/sshd -h \\$PWD/.ssh/host_key.*-p 123.*",
	)
}

func (s *SSuite) TestStartSSHServerFail(c *C) {
	container := &FakeContainer{
		ShouldError: true,
	}

	session := &Session{
		Container: container,
		Port:      MappedPort(123),
	}

	err := session.StartSSHServer()
	c.Assert(err, NotNil)
}

func (s *SSuite) TestSessionMarshalling(c *C) {
	container := &FakeContainer{
		Handle: "to-s-32",
	}

	session := &Session{
		Container: container,
		Port:      MappedPort(123),
	}

	json, err := session.MarshalJSON()
	c.Assert(err, IsNil)

	c.Assert(
		string(json),
		Equals,
		`{"container":"to-s-32","port":123}`,
	)
}
