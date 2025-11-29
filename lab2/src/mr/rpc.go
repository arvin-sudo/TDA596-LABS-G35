package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

// RPC definitions
type AssignTaskArgs struct {
	// empty because no useful info need to be replied.
}

type AssignTaskReply struct {
	Id       int
	TaskType TaskType
	Filename string
	NReduce  int
	NMap     int
}

type TaskDoneArgs struct {
	Id       int
	TaskType TaskType
}

type TaskDoneReply struct {
	// empty because no useful info need to be replied.
}

// --------- used by http -----------
type TaskDoneArgsHTTP struct {
	TaskDoneArgs
	MyAddress string
}

type AssignTaskReplyHTTP struct {
	AssignTaskReply
	// only for map task to fetch raw file.
	Port int

	// only for reduce task to get intermediate files.
	WorkerMapTaskAddrMap map[int]string
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
