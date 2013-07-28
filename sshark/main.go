package main

import (
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/sshark"
	"log"
)

func main() {
	// TODO: rename package to just cfmessagebus
	mbus, err := go_cfmessagebus.NewCFMessageBus("NATS")
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	mbus.Configure("127.0.0.1", 4222, "", "")
	err = mbus.Connect()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	config := sshark.AgentConfig{
		WardenSocketPath: "/tmp/warden.sock",
	}

	agent, err := sshark.NewAgent(config)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Printf("agent ID: %s\n", agent.ID)

	err = agent.HandleStarts(mbus)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	err = agent.HandleStops(mbus)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	select {}
}
