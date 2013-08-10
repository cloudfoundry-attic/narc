package main

import (
	"flag"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/vito/narc"
	"log"
)

var configFile = flag.String("config", "", "path to config file")

func main() {
	flag.Parse()

	var config narc.Config

	if *configFile != "" {
		config = narc.LoadConfig(*configFile)
	} else {
		config = narc.DefaultConfig
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

	agentConfig := narc.AgentConfig{
		WardenSocketPath: config.WardenSocketPath,
	}

	agent, err := narc.NewAgent(agentConfig)
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

	server, err := narc.NewProxyServer(agent)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	err = server.Start(8081)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	select {}
}
