package sshark

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"github.com/vito/gordon"
	. "launchpad.net/gocheck"
	"net"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type FailingConnectionProvider struct{}

func (c *FailingConnectionProvider) ProvideConnection() (*warden.Connection, error) {
	return nil, errors.New("nope!")
}

type FakeConnectionProvider struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
}

func (c *FakeConnectionProvider) ProvideConnection() (*warden.Connection, error) {
	return warden.NewConnection(
		&fakeConn{
			ReadBuffer:  c.ReadBuffer,
			WriteBuffer: c.WriteBuffer,
		},
	), nil
}

type ManyConnectionProvider struct {
	ReadBuffers  []*bytes.Buffer
	WriteBuffers []*bytes.Buffer
}

func (c *ManyConnectionProvider) ProvideConnection() (*warden.Connection, error) {
	if len(c.ReadBuffers) == 0 {
		return nil, errors.New("no more connections")
	}

	rbuf := c.ReadBuffers[0]
	c.ReadBuffers = c.ReadBuffers[1:]

	wbuf := c.WriteBuffers[0]
	c.WriteBuffers = c.WriteBuffers[1:]

	return warden.NewConnection(
		&fakeConn{
			ReadBuffer:  rbuf,
			WriteBuffer: wbuf,
		},
	), nil
}

func messages(msgs ...proto.Message) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	for _, msg := range msgs {
		payload, err := proto.Marshal(msg)
		if err != nil {
			panic(err.Error())
		}

		message := &warden.Message{
			Type:    warden.Message_Type(message2type(msg)).Enum(),
			Payload: payload,
		}

		messagePayload, err := proto.Marshal(message)
		if err != nil {
			panic("failed to marshal message")
		}

		buf.Write([]byte(fmt.Sprintf("%d\r\n%s\r\n", len(messagePayload), messagePayload)))
	}

	return buf
}

// TODO: yikes
func message2type(msg proto.Message) int32 {
	switch msg.(type) {
	case *warden.ErrorResponse:
		return 1

	case *warden.CreateRequest, *warden.CreateResponse:
		return 11
	case *warden.StopRequest, *warden.StopResponse:
		return 12
	case *warden.DestroyRequest, *warden.DestroyResponse:
		return 13
	case *warden.InfoRequest, *warden.InfoResponse:
		return 14

	case *warden.SpawnRequest, *warden.SpawnResponse:
		return 21
	case *warden.LinkRequest, *warden.LinkResponse:
		return 22
	case *warden.RunRequest, *warden.RunResponse:
		return 23
	case *warden.StreamRequest, *warden.StreamResponse:
		return 24

	case *warden.NetInRequest, *warden.NetInResponse:
		return 31
	case *warden.NetOutRequest, *warden.NetOutResponse:
		return 32

	case *warden.CopyInRequest, *warden.CopyInResponse:
		return 41
	case *warden.CopyOutRequest, *warden.CopyOutResponse:
		return 42

	case *warden.LimitMemoryRequest, *warden.LimitMemoryResponse:
		return 51
	case *warden.LimitDiskRequest, *warden.LimitDiskResponse:
		return 52
	case *warden.LimitBandwidthRequest, *warden.LimitBandwidthResponse:
		return 53

	case *warden.PingRequest, *warden.PingResponse:
		return 91
	case *warden.ListRequest, *warden.ListResponse:
		return 92
	case *warden.EchoRequest, *warden.EchoResponse:
		return 93
	}

	panic("unknown message type")
}

type fakeConn struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
	WriteChan   chan string
	Closed      bool
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	return f.ReadBuffer.Read(b)
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	if f.WriteChan != nil {
		f.WriteChan <- string(b)
	}

	return f.WriteBuffer.Write(b)
}

func (f *fakeConn) Close() error {
	f.Closed = true
	return nil
}

func (f *fakeConn) SetDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:4222")
	return addr
}

func (f *fakeConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:65525")
	return addr
}

func waitReceive(from chan []byte, giveup time.Duration) []byte {
	select {
	case val := <-from:
		return val
	case <-time.After(giveup):
		return nil
	}
}
