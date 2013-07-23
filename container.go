package sshark

type JobId uint32
type MappedPort uint32

type StreamOutput struct {
	Name string
	Data string

	Finished   bool
	ExitStatus uint32
}

type JobInfo struct {
	ExitStatus uint32
}

type Container interface {
	Destroy() error
	Spawn(command string) (JobId, error)
	Stream(job JobId) (chan *StreamOutput, error)
	Run(command string) (*JobInfo, error)
	NetIn() (MappedPort, error)
	CopyIn(src, dst string) error
}
