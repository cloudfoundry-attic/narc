package sshark

import (
	"fmt"
)

type Session struct {
	Container Container
	Port      MappedPort
}

func (s *Session) LoadPublicKey(publicKey string) error {
	command := fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", publicKey)
	_, err := s.Container.Run(command)
	return err
}

func (s *Session) StartSSHServer() error {
	command := fmt.Sprintf(
		"dropbearkey -t rsa -f .koala; dropbear -F -E -r .koala -p :%d",
		s.Port,
	)
	_, err := s.Container.Run(command)
	return err
}
