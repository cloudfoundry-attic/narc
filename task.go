package narc

type Task struct {
	Container   Container
	Limits      TaskLimits
	SecureToken string
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}
