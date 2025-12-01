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

var coordinatorAddress string

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
	WorkerID int    // ID of worker completing the task (0 in basic mode)
}

// Task reply complete - coordinator acknowledges
type TaskCompleteReply struct {
	Success bool
}

type RegisterWorkerArgs struct {
	WorkerAddress string
}

type RegisterWorkerReply struct {
	WorkerID int
}

// GetMapWorker - used by reduce workers to find which worker completed a map task
type GetMapWorkerArgs struct {
	MapTaskID int
}

type GetMapWorkerReply struct {
	WorkerAddr string
	Found      bool
}

// FetchIntermediateFile - used by reduce workers to fetch intermediate files from map workers
type FetchIntermediateFileArgs struct {
	MapTaskID    int
	ReduceTaskID int
}

type FetchIntermediateFileReply struct {
	Content []byte
	Found   bool
}

// ReportTaskFailure - worker reports that a task failed (e.g., can't fetch intermediate files)
type ReportTaskFailureArgs struct {
	TaskID      int
	TaskType    string // "Map" or "Reduce"
	Reason      string // Description of failure
	FailedMapID int    // For reduce tasks: which map task's file couldn't be fetched (-1 if not applicable)
}

type ReportTaskFailureReply struct {
	Acknowledged bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	// advanced mode: TCP address
	if coordinatorAddress != "" {
		return coordinatorAddress
	}

	// basic mode: UNIX-domain socket (for test-mr.sh)
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}

func init() {
	coordinatorAddress = os.Getenv("COOR_ADV")
}
