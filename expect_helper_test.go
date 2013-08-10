package narc

import (
	ex "bitbucket.org/teythoon/expect"
	"io"
	. "launchpad.net/gocheck"
)

func expect(c *C, reader *ex.Reader, output string) {
	_, err := reader.Expect(output)
	if err != nil {
		c.Error(err.Error())
	}
}

// TODO: remove once expect just needs an io.Reader
type bogusWriteCloser struct {
	io.Reader
}

func (b bogusWriteCloser) Write([]byte) (int, error) {
	return 0, nil
}

func (b bogusWriteCloser) Close() error {
	return nil
}
