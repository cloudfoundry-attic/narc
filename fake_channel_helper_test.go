package narc

import (
	"code.google.com/p/go.crypto/ssh"
	"io"
)

type FakeChannel struct {
	Acks chan bool

	requests []ssh.ChannelRequest

	channelType string
	extraData   []byte

	writePipe *io.PipeWriter
	readPipe  *io.PipeReader
}

func NewFakeChannel(requests []ssh.ChannelRequest) *FakeChannel {
	read, write := io.Pipe()

	return &FakeChannel{
		Acks: make(chan bool),

		requests:  requests,
		writePipe: write,
		readPipe:  read,
	}
}

func (*FakeChannel) Accept() error {
	return nil
}

func (*FakeChannel) Reject(reason ssh.RejectionReason, message string) error {
	return nil
}

func (f *FakeChannel) Read(data []byte) (int, error) {
	if len(f.requests) == 0 {
		return 0, io.EOF
	}

	req := f.requests[0]
	f.requests = f.requests[1:]

	return 0, req
}

func (f *FakeChannel) Write(data []byte) (int, error) {
	return f.writePipe.Write(data)
}

func (f *FakeChannel) Close() error {
	return nil
}

func (f *FakeChannel) Stderr() io.Writer {
	return nil
}

func (f *FakeChannel) AckRequest(ok bool) error {
	select {
	case f.Acks <- ok:
	default:
	}

	return nil
}

func (f *FakeChannel) ChannelType() string {
	return f.channelType
}

func (f *FakeChannel) ExtraData() []byte {
	return f.extraData
}
