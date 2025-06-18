package config

type LockFile []Lock

type Lock struct {
	Id      int // unique identifier for the lock
	Key     string // unique identifier for the lock
	Value Monitor
}
