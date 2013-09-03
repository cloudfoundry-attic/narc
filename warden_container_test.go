package narc

import (
	"errors"
	. "launchpad.net/gocheck"
)

type WCSuite struct {
	fakeCmdWithJson FakeCmdWithJson
}

type FakeCmdWithJson struct {
	request  *CreateContainerMessage
	response *CreateContainerResponse
	cmd      string

	stubErr    error
	stubHandle string
}

func (f *FakeCmdWithJson) Run(request *CreateContainerMessage, response *CreateContainerResponse, cmd string) error {
	f.request = request
	f.response = response
	f.cmd = cmd

	response.Handle = f.stubHandle

	if f.stubErr != nil {
		return f.stubErr
	}

	return nil
}

func init() {
	Suite(&WCSuite{})
}

func (s *WCSuite) TestNewWardenSendCorrectMessage(c *C) {
	NewWardenContainer("a_socket", TaskLimits{MemoryLimitInBytes: 10, DiskLimitInBytes: 20}, &s.fakeCmdWithJson)

	c.Assert(s.fakeCmdWithJson.cmd, Equals, "create_warden_container.sh")
	c.Assert(s.fakeCmdWithJson.request, DeepEquals, &CreateContainerMessage{
		WardenSocketPath: "a_socket",
		DiskLimit:        20,
		MemoryLimit:      10,
		Network:          true,
	})
}

func (s *WCSuite) TestNewWardenHandlesErrorsInResponse(c *C) {
	expectedError := errors.New("adad")
	s.fakeCmdWithJson.stubErr = expectedError
	_, err := NewWardenContainer("a_socket", TaskLimits{MemoryLimitInBytes: 10, DiskLimitInBytes: 20}, &s.fakeCmdWithJson)
	c.Assert(err, Equals, expectedError)
}

func (s *WCSuite) TestNewWardenHandlesNoErrorsInResponse(c *C) {
	s.fakeCmdWithJson.stubHandle = "abc"
	s.fakeCmdWithJson.stubErr = nil
	container, err := NewWardenContainer("a_socket",
		TaskLimits{MemoryLimitInBytes: 10, DiskLimitInBytes: 20},
		&s.fakeCmdWithJson)
	c.Assert(err, IsNil)
	c.Assert(container.ID(), Equals, "abc")
}

// Run
//fileInfor, _ := os.Stat("/opt/warden/containers/" + wardenContainer.ID())
// _, err = wardenContainer.Run("ls")

// Destroy
