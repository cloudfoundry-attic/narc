package sshark

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"strconv"
	"time"
)

type Config struct {
	MessageBus        MessageBusConfig
	Capacity          CapacityConfig
	AdvertiseInterval time.Duration
	WardenSocketPath  string
	StateFilePath     string
}

type MessageBusConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type CapacityConfig struct {
	MemoryInBytes uint64
	DiskInBytes   uint64
}

var kilobyte = uint64(1024)
var megabyte = kilobyte * 1024
var gigabyte = megabyte * 1024

var DefaultConfig = Config{
	MessageBus: MessageBusConfig{
		Host: "127.0.0.1",
		Port: 4222,
	},

	Capacity: CapacityConfig{
		MemoryInBytes: 1 * gigabyte,
		DiskInBytes:   1 * gigabyte,
	},

	WardenSocketPath: "/tmp/warden.sock",
	StateFilePath:    "/tmp/sshark.json",

	AdvertiseInterval: 10 * time.Second,
}

func LoadConfig(configFilePath string) Config {
	file := yaml.ConfigFile(configFilePath)

	mbusHost := file.Require("message_bus.host")
	mbusPort, err := strconv.Atoi(file.Require("message_bus.port"))
	if err != nil {
		panic("non-numeric message bus port")
	}

	mbusUsername, _ := file.Get("message_bus.username")
	mbusPassword, _ := file.Get("message_bus.password")

	wardenSocketPath := file.Require("warden_socket")
	stateFilePath, _ := file.Get("state_file")

	capacityMemory, err := strconv.Atoi(file.Require("capacity.memory"))
	if err != nil {
		panic("non-numeric memory capacity")
	}

	capacityDisk, err := strconv.Atoi(file.Require("capacity.disk"))
	if err != nil {
		panic("non-numeric disk capacity")
	}

	advertiseInterval, err := strconv.Atoi(file.Require("advertise_interval"))
	if err != nil {
		panic("non-numeric advertise interval")
	}

	return Config{
		MessageBus: MessageBusConfig{
			Host:     mbusHost,
			Port:     mbusPort,
			Username: mbusUsername,
			Password: mbusPassword,
		},

		Capacity: CapacityConfig{
			MemoryInBytes: uint64(capacityMemory) * 1024 * 1024,
			DiskInBytes:   uint64(capacityDisk) * 1024 * 1024,
		},

		AdvertiseInterval: time.Duration(advertiseInterval) * time.Second,

		WardenSocketPath: wardenSocketPath,
		StateFilePath:    stateFilePath,
	}
}
