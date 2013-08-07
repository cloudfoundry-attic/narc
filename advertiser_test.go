package sshark

import (
	"fmt"
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	"github.com/vito/gordon"
	. "launchpad.net/gocheck"
	"time"
)

type AdvertiserSuite struct{}

func init() {
	Suite(&AdvertiserSuite{})
}

func (a *AdvertiserSuite) TestAdvertiserAdvertisesID(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	config := AgentConfig{
		WardenSocketPath:  "/tmp/warden.sock",
		AdvertiseInterval: 100 * time.Millisecond,
	}

	advertisements := make(chan []byte)

	mbus.Subscribe("ssh.advertise", func(msg []byte) {
		advertisements <- msg
	})

	agent, err := NewAgent(config)
	c.Assert(err, IsNil)

	go agent.AdvertisePeriodically(mbus)

	ad := waitReceive(advertisements, 5*time.Second)
	c.Assert(string(ad), Matches, fmt.Sprintf(`.*"id":"%s".*`, agent.ID))
}

func (a *ASuite) TestAdvertiserAdvertisesPeriodically(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	config := AgentConfig{
		WardenSocketPath:  "/tmp/warden.sock",
		AdvertiseInterval: 100 * time.Millisecond,
	}

	advertisements := make(chan []byte)

	mbus.Subscribe("ssh.advertise", func(msg []byte) {
		advertisements <- msg
	})

	agent, err := NewAgent(config)
	c.Assert(err, IsNil)

	go agent.AdvertisePeriodically(mbus)

	msg1 := waitReceive(advertisements, 1*time.Second)
	c.Assert(msg1, NotNil)

	time1 := time.Now()

	msg2 := waitReceive(advertisements, 1*time.Second)
	c.Assert(msg2, NotNil)

	time2 := time.Now()

	c.Assert(time2.Sub(time1) >= 100*time.Millisecond, Equals, true)
}

func (a *ASuite) TestAdvertiserAdvertisesAvailableMemory(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	config := AgentConfig{
		WardenSocketPath:  "/tmp/warden.sock",
		AdvertiseInterval: 100 * time.Millisecond,
		Capacity: CapacityConfig{
			MemoryInBytes: 1024 * gigabyte,
			DiskInBytes:   1024 * gigabyte,
		},
	}

	advertisements := make(chan []byte)

	mbus.Subscribe("ssh.advertise", func(msg []byte) {
		advertisements <- msg
	})

	agent, err := NewAgent(config)
	c.Assert(err, IsNil)

	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: a.Config.WardenSocketPath,
		},
	)

	err = client.Connect()
	c.Assert(err, IsNil)

	handles, err := client.List()
	c.Assert(err, IsNil)

	var reservedMemory uint64
	for _, handle := range handles.GetHandles() {
		memoryLimit, err := client.GetMemoryLimit(handle)
		if err != nil {
			c.Assert(err, IsNil)
		}

		reservedMemory += memoryLimit
	}

	reservedMemoryInMegabytes := reservedMemory / 1024 / 1024

	agent.StartSession(
		"some-session-guid",
		SessionLimits{MemoryLimitInBytes: uint64(1 * megabyte)},
	)

	go agent.AdvertisePeriodically(mbus)

	ad := waitReceive(advertisements, 1*time.Second)
	c.Assert(
		string(ad),
		Matches,
		fmt.Sprintf(`.*"available_memory":%d.*`, (1024*1024)-reservedMemoryInMegabytes-1),
	)
}

func (a *ASuite) TestAdvertiserAdvertisesAvailableDisk(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	config := AgentConfig{
		WardenSocketPath:  "/tmp/warden.sock",
		AdvertiseInterval: 100 * time.Millisecond,
		Capacity: CapacityConfig{
			MemoryInBytes: 1024 * gigabyte,
			DiskInBytes:   1024 * gigabyte,
		},
	}

	advertisements := make(chan []byte)

	mbus.Subscribe("ssh.advertise", func(msg []byte) {
		advertisements <- msg
	})

	agent, err := NewAgent(config)
	c.Assert(err, IsNil)

	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: a.Config.WardenSocketPath,
		},
	)

	err = client.Connect()
	c.Assert(err, IsNil)

	handles, err := client.List()
	c.Assert(err, IsNil)

	var reservedDisk uint64
	for _, handle := range handles.GetHandles() {
		diskLimit, err := client.GetDiskLimit(handle)

		if err != nil {
			c.Assert(err, IsNil)
		}

		reservedDisk += diskLimit
	}

	reservedDiskInMegabytes := reservedDisk / 1024 / 1024

	agent.StartSession(
		"some-session-guid",
		SessionLimits{DiskLimitInBytes: uint64(1 * megabyte)},
	)

	go agent.AdvertisePeriodically(mbus)

	ad := waitReceive(advertisements, 1*time.Second)
	c.Assert(
		string(ad),
		Matches,
		fmt.Sprintf(`.*"available_disk":%d.*`, (1024*1024)-reservedDiskInMegabytes-1),
	)
}
