package main

import (
	"flag"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/sshark"
	"log"
)

var configFile = flag.String("config", "", "path to config file")

func main() {
	flag.Parse()

	var config sshark.Config

	if *configFile != "" {
		config = sshark.LoadConfig(*configFile)
	} else {
		config = sshark.DefaultConfig
	}

	mbus, err := go_cfmessagebus.NewCFMessageBus("NATS")
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	mbus.Configure(
		config.MessageBus.Host,
		config.MessageBus.Port,
		config.MessageBus.Username,
		config.MessageBus.Password,
	)

	err = mbus.Connect()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	agentConfig := sshark.AgentConfig{
		WardenSocketPath: config.WardenSocketPath,
		StateFilePath:    config.StateFilePath,
	}

	agent, err := sshark.NewAgent(agentConfig)
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
