package main

import (
	"fmt"
	"github.com/cloudfoundry/sshark"
	"github.com/vito/gordon"
)

func main() {
	client := warden.NewClient(
		&warden.ConnectionInfo{
			SocketPath: "/tmp/warden.sock",
		},
	)

	err := client.Connect()
	if err != nil {
		fmt.Println("could not connect to Warden:", err)
		return
	}

	container, err := sshark.NewWardenContainer(client)
	if err != nil {
		fmt.Println("failed to create container:", err)
		return
	}

	ran := make(chan bool)

	for i := 0; i < 20; i += 1 {
		go func(i int) {
			res, err := container.Run(fmt.Sprintf("sleep 0.%d; exit %d", i, i))
			if err != nil {
				fmt.Println("could not run command:", err)
				return
			}

			fmt.Println("Ran:", res)

			ran <- true
		}(i)
	}

	for i := 0; i < 20; i += 1 {
		<-ran
	}
}
