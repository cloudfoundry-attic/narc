package narc

import (
	"encoding/json"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type WCSuite struct{}

func init() {
	Suite(&WCSuite{})
}

func (w *WCSuite) TestNewWardenContainerSuccess(c *C) {
	wardenContainer, err := NewWardenContainer("/tmp/warden.sock",
		TaskLimits{MemoryLimitInBytes: 4, DiskLimitInBytes: 5})

	c.Assert(err, IsNil)
	c.Assert(wardenContainer.ID(), NotNil)
	fileInfor, _ := os.Stat("/opt/warden/containers/" + wardenContainer.ID())
	c.Assert(fileInfor, NotNil)
	_, err = wardenContainer.Run("ls")
	c.Assert(err, IsNil)
	err = wardenContainer.Destroy()
	c.Assert(err, IsNil)
}

func (w *WCSuite) TestNewWardenContainerReturningError(c *C) {
	wardenContainer, err := NewWardenContainer(
		"/me/dont/exist",
		TaskLimits{MemoryLimitInBytes: 65, DiskLimitInBytes: 400})
	c.Assert(err, NotNil)
	c.Assert(wardenContainer, IsNil)
}

func (w *WCSuite) TestNewWardenContainerLimits(c *C) {
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	defer os.Remove("test.out")

	tmpDir := os.TempDir()
	newFile, err := os.Create(tmpDir + "/create_warden_container.sh")
	if err != nil {
		panic(err)
	}
	newFile.WriteString(`#!/usr/bin/env ruby
require "json"
contents = STDIN.read
File.open("test.out", "w") do |f|
f.write(contents)
f.flush
end

puts({"handle" => "abc"}.to_json)
`)
	newFile.Close()
	err = os.Chmod(newFile.Name(), 0777)
	if err != nil {
		panic(err)
	}

	os.Setenv("PATH", tmpDir+":"+oldPath)

	wardenContainer, err := NewWardenContainer("/tmp/warden.sock",
		TaskLimits{MemoryLimitInBytes: 40, DiskLimitInBytes: 6})

	c.Assert(err, IsNil)
	c.Assert(wardenContainer.ID(), NotNil)
	file, err := ioutil.ReadFile("test.out")
	if err != nil {
		panic(err)
	}
	var createRequest CreateContainerMessage
	json.Unmarshal(file, &createRequest)
	c.Assert(createRequest.MemoryLimit, Equals, uint64(40))
	c.Assert(createRequest.DiskLimit, Equals, uint64(6))
	c.Assert(createRequest.Network, Equals, true)
}
