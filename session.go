package sshark

import (
	"fmt"
)

type Session struct {
	Container Container
	Port      MappedPort
	Limits    SessionLimits
}

type SessionLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

func (s *Session) LoadPublicKey(publicKey string) error {
	command := fmt.Sprintf("mkdir ~/.ssh; echo '%s' >> ~/.ssh/authorized_keys", publicKey)
	_, err := s.Container.Run(command)
	return err
}

func (s *Session) StartSSHServer() error {
	command := fmt.Sprintf(
		`mkdir -p ~/.ssh && ssh-keygen -t rsa -f .ssh/host_key -N "" && /usr/sbin/sshd -h $PWD/.ssh/host_key -o UsePrivilegeSeparation=no -p %d`,
		s.Port,
	)

	_, err := s.Container.Run(command)
	return err
}
