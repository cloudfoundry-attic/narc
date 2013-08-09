package narc

import (
	"os"
)

type Task struct {
	Container   Container
	Limits      TaskLimits
	SecureToken string

	pty *os.File
}

type TaskLimits struct {
	MemoryLimitInBytes uint64
	DiskLimitInBytes   uint64
}
