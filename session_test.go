package sshark

import (
	"errors"
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
		"echo 'super-secure' >> ~/.ssh/authorized_keys",
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

	c.Assert(container.LastCommand, Equals,
		"dropbearkey -t rsa -f .koala; dropbear -F -E -r .koala -p :123",
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

type FakeContainer struct {
	LastCommand string
	ShouldError bool
}

func (c *FakeContainer) Destroy() error {
	return nil
}

func (c *FakeContainer) Spawn(command string) (JobId, error) {
	return 42, nil
}

func (c *FakeContainer) Stream(job JobId) (chan *StreamOutput, error) {
	return make(chan *StreamOutput), nil
}

func (c *FakeContainer) Run(command string) (*JobInfo, error) {
	if c.ShouldError {
		return nil, errors.New("uh oh")
	}

	c.LastCommand = command

	return &JobInfo{}, nil
}

func (c *FakeContainer) NetIn() (MappedPort, error) {
	return 0, nil
}
