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

	mbus, err := cfmessagebus.NewMessageBus("NATS")
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
		WardenSocketPath:  config.WardenSocketPath,
		StateFilePath:     config.StateFilePath,
		AdvertiseInterval: config.AdvertiseInterval,
	}

	agent, err := sshark.NewAgent(agentConfig)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

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

	go agent.AdvertisePeriodically(mbus)

	select {}
}
