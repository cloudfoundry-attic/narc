package narc

type MappedPort uint32

type JobInfo struct {
	ExitStatus uint32
}

type ContainerInfo struct {
	MemoryLimitInBytes uint64
}

type Container interface {
	ID() string
	Destroy() error
	Run(command string) (*JobInfo, error)
}
