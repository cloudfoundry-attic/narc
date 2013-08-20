package main

import (
	"flag"
	"log"

	"github.com/cloudfoundry/gibson"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/narc"
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

	containerProvider := narc.WardenTaskBackend{
		WardenSocketPath:     config.WardenSocketPath,
		WardenContainersPath: config.WardenContainersPath,
	}

	routerClient := gibson.NewCFRouterClient(config.Host, mbus)
	routerClient.Greet()

	proxyServerPort := 8081

	agent, err := narc.NewAgent(containerProvider, routerClient, proxyServerPort)
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

	server, err := narc.NewProxyServer(agent.Registry)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	err = server.Start(proxyServerPort)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	select {}
}
