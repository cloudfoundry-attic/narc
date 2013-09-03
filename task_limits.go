package narc

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}

func (limits *TaskLimits) IsValid() bool {
	return limits.MemoryLimitInBytes > 0 && limits.DiskLimitInBytes > 0
}
