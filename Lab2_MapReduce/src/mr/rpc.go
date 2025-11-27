package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Task request - worker asks coordinator for a task
type RequestTaskArgs struct {
	WorkerID int
}

// Task reply - coordinator responds with a task
type RequestTaskReply struct {
	TaskType string // "Map", "Reduce", or "Wait" or "Exit"
	TaskID   int
	Filename string // for Map tasks: input-file to read
	NReduce  int    // number of reduce-buckets
}

// Task request complete - worker notifies coordinator of task completion
type TaskCompleteArgs struct {
	TaskID   int
	TaskType string // "Map" or "Reduce"
}

// Task reply complete - coordinator acknowledges
type TaskCompleteReply struct {
	Success bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
