package sshark

type MappedPort uint32

type JobInfo struct {
	ExitStatus uint32
}

type Container interface {
	ID() string
	Destroy() error
	Run(command string) (*JobInfo, error)
	NetIn() (MappedPort, error)
}
