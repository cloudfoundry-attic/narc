package sshark

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"github.com/vito/gordon"
	. "launchpad.net/gocheck"
)

type WCSuite struct{}

func init() {
	Suite(&WCSuite{})
}

func (w *WCSuite) TestNewWardenContainer(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.CreateResponse{Handle: proto.String("foo-handle")},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	c.Assert(wardenContainer.Handle, Equals, "foo-handle")

	c.Assert(
		string(fcp.WriteBuffer.Bytes()),
		Equals,
		string(messages(&warden.CreateRequest{}).Bytes()),
	)
}

func (w *WCSuite) TestNewWardenContainerFailure(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.ErrorResponse{Message: proto.String("NO")},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	_, err = NewWardenContainer(client)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "NO")
}

func (w *WCSuite) TestContainerDestroying(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.CreateResponse{Handle: proto.String("foo-handle")},
			&warden.DestroyResponse{},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	err = wardenContainer.Destroy()
	c.Assert(err, IsNil)

	c.Assert(
		string(fcp.WriteBuffer.Bytes()),
		Equals,
		string(
			messages(
				&warden.CreateRequest{},
				&warden.DestroyRequest{Handle: proto.String("foo-handle")},
			).Bytes(),
		),
	)
}

func (w *WCSuite) TestContainerDestroyingFailure(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.CreateResponse{Handle: proto.String("foo-handle")},
			&warden.ErrorResponse{Message: proto.String("unknown handle")},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	err = wardenContainer.Destroy()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "unknown handle")
}

func (w *WCSuite) TestRun(c *C) {
	firstWriteBuf := bytes.NewBuffer([]byte{})
	secondWriteBuf := bytes.NewBuffer([]byte{})

	fcp := &ManyConnectionProvider{
		ReadBuffers: []*bytes.Buffer{
			messages(&warden.CreateResponse{Handle: proto.String("foo-handle")}),
			messages(&warden.RunResponse{ExitStatus: proto.Uint32(42)}),
		},
		WriteBuffers: []*bytes.Buffer{
			firstWriteBuf,
			secondWriteBuf,
		},
	}

	client := warden.NewClient(fcp)
	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	jobInfo, err := wardenContainer.Run("dropbear")
	c.Assert(err, IsNil)

	c.Assert(jobInfo.ExitStatus, Equals, uint32(42))

	c.Assert(
		string(secondWriteBuf.Bytes()),
		Equals,
		string(
			messages(
				&warden.RunRequest{
					Handle: proto.String("foo-handle"),
					Script: proto.String("dropbear"),
				},
			).Bytes(),
		),
	)
}

func (w *WCSuite) TestContainerRunningFailure(c *C) {
	firstWriteBuf := bytes.NewBuffer([]byte{})
	secondWriteBuf := bytes.NewBuffer([]byte{})

	fcp := &ManyConnectionProvider{
		ReadBuffers: []*bytes.Buffer{
			messages(&warden.CreateResponse{Handle: proto.String("foo-handle")}),
			messages(&warden.ErrorResponse{Message: proto.String("fork bomb detected")}),
		},
		WriteBuffers: []*bytes.Buffer{
			firstWriteBuf,
			secondWriteBuf,
		},
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	_, err = wardenContainer.Run("foo")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "fork bomb detected")
}

func (w *WCSuite) TestContainerPortMapping(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.CreateResponse{Handle: proto.String("foo-handle")},
			&warden.NetInResponse{
				HostPort:      proto.Uint32(7331),
				ContainerPort: proto.Uint32(7331),
			},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	port, err := wardenContainer.NetIn()
	c.Assert(err, IsNil)

	c.Assert(port, Equals, MappedPort(7331))

	c.Assert(
		string(fcp.WriteBuffer.Bytes()),
		Equals,
		string(
			messages(
				&warden.CreateRequest{},
				&warden.NetInRequest{Handle: proto.String("foo-handle")},
			).Bytes(),
		),
	)
}

func (w *WCSuite) TestContainerPortMappingFailure(c *C) {
	fcp := &FakeConnectionProvider{
		ReadBuffer: messages(
			&warden.CreateResponse{Handle: proto.String("foo-handle")},
			&warden.ErrorResponse{Message: proto.String("fresh outta ports")},
		),
		WriteBuffer: bytes.NewBuffer([]byte{}),
	}

	client := warden.NewClient(fcp)

	err := client.Connect()
	c.Assert(err, IsNil)

	wardenContainer, err := NewWardenContainer(client)
	c.Assert(err, IsNil)

	_, err = wardenContainer.NetIn()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "fresh outta ports")
}
