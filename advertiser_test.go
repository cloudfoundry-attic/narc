package sshark

import (
	"fmt"
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
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
		AdvertiseInterval: 100 * time.Millisecond,
	}

	advertisements := make(chan []byte)

	mbus.Subscribe("ssh.advertise", func(msg []byte) {
		advertisements <- msg
	})

	agent, err := NewAgent(config)
	c.Assert(err, IsNil)

	go agent.AdvertisePeriodically(mbus)

	ad := waitReceive(advertisements, 1*time.Second)
	c.Assert(string(ad), Equals, fmt.Sprintf(`{"id":"%s"}`, agent.ID))
}

func (a *ASuite) TestAdvertiserAdvertisesPeriodically(c *C) {
	mbus := mock_cfmessagebus.NewMockMessageBus()

	config := AgentConfig{
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

func (a *ASuite) TestAdvertismentIncludesAvailableMemory(c *C) {

}
