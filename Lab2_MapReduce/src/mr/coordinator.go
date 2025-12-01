package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const (
	// Task timeout - after this duration, assume worker has died and reassign task
	taskTimeout = 10 * time.Second
)

type Coordinator struct {
	// Your definitions here.
	mutex        sync.Mutex
	nReduce      int
	mapTasks     []MapTask
	reduceTasks  []ReduceTask
	workers      map[int]string // workerID -> workerAddress
	nextWorkerID int
}

type TaskStatus int

const (
	Idle TaskStatus = iota
	InProgress
	Completed
)

type MapTask struct {
	ID            int
	Filename      string
	Status        TaskStatus
	StartTime     time.Time
	WorkerID      int
	WorkerAddress string
}

type ReduceTask struct {
	ID        int
	Status    TaskStatus
	StartTime time.Time
}

// Your code here -- RPC handlers for the worker to call.

// RequestTask - worker requests a task from the coordinator
func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	// lock mutex before accessing mapTasks and unlock when done
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// CHECK: if all reduce tasks are completed, tell workers to exit
	allReduceCompleted := true
	for i := range c.reduceTasks {
		if c.reduceTasks[i].Status != Completed {
			allReduceCompleted = false
			break
		}
	}
	if allReduceCompleted {
		reply.TaskType = "Exit"
		fmt.Println("Coordinator: All tasks completed, signaling workers to exit")
		return nil
	}

	// FIRST: check if all map tasks are completed
	allMapCompleted := true
	for i := range c.mapTasks {
		if c.mapTasks[i].Status != Completed {
			allMapCompleted = false
			break
		}
	}

	// !FIRST: if maps not completed, assign idle map tasks
	if !allMapCompleted {
		// First, check for timed-out tasks and reset them
		for i := range c.mapTasks {
			if c.mapTasks[i].Status == InProgress {
				elapsed := time.Since(c.mapTasks[i].StartTime)
				if elapsed > taskTimeout {
					fmt.Printf("Coordinator: Map task %d timed out after %v, resetting to Idle\n",
						c.mapTasks[i].ID, elapsed)
					c.mapTasks[i].Status = Idle
				}
			}
		}

		// find idle map task
		for i := range c.mapTasks {
			if c.mapTasks[i].Status == Idle {
				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].StartTime = time.Now()
				reply.TaskType = "Map"
				reply.TaskID = c.mapTasks[i].ID
				reply.Filename = c.mapTasks[i].Filename
				reply.NReduce = c.nReduce
				fmt.Printf("Coordinator: Assigned map task %d (file: %s) to worker\n",
					c.mapTasks[i].ID, c.mapTasks[i].Filename)
				return nil
			}
		}
	}

	// SECOND: maps tasks done -> assign reduce tasks
	if allMapCompleted {
		// First, check for timed-out reduce tasks and reset them
		for i := range c.reduceTasks {
			if c.reduceTasks[i].Status == InProgress {
				elapsed := time.Since(c.reduceTasks[i].StartTime)
				if elapsed > taskTimeout {
					fmt.Printf("Coordinator: Reduce task %d timed out after %v, resetting to Idle\n",
						c.reduceTasks[i].ID, elapsed)
					c.reduceTasks[i].Status = Idle
				}
			}
		}

		for i := range c.reduceTasks {
			if c.reduceTasks[i].Status == Idle {
				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].StartTime = time.Now()
				reply.TaskType = "Reduce"
				reply.TaskID = c.reduceTasks[i].ID
				reply.NReduce = c.nReduce
				fmt.Printf("Coordinator: Assigned reduce task %d to worker\n",
					c.reduceTasks[i].ID)
				return nil
			}
		}
	}

	// THIRD: no idle tasks available -> tell worker to wait
	reply.TaskType = "Wait"
	reply.TaskID = 0

	return nil
}

// task complete - worker notifies coordinator of task completion
func (c *Coordinator) TaskComplete(args *TaskCompleteArgs, reply *TaskCompleteReply) error {
	// lock mutex before accessing mapTasks and unlock when done
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if args.TaskType == "Map" {
		if args.TaskID >= 0 && args.TaskID < len(c.mapTasks) {
			c.mapTasks[args.TaskID].Status = Completed
			// Track which worker completed this map task (for distributed mode)
			if args.WorkerID != 0 {
				c.mapTasks[args.TaskID].WorkerID = args.WorkerID
				c.mapTasks[args.TaskID].WorkerAddress = c.workers[args.WorkerID]
			}
			reply.Success = true
			fmt.Printf("Coordinator: Map task %d completed by worker %d\n", args.TaskID, args.WorkerID)
		}
	}

	if args.TaskType == "Reduce" {
		if args.TaskID >= 0 && args.TaskID < len(c.reduceTasks) {
			c.reduceTasks[args.TaskID].Status = Completed
			reply.Success = true
			fmt.Printf("Coordinator: Reduce task %d completed by worker %d\n", args.TaskID, args.WorkerID)
		}
	}

	return nil
}

// register worker
func (c *Coordinator) RegisterWorker(args *RegisterWorkerArgs, reply *RegisterWorkerReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	reply.WorkerID = c.nextWorkerID
	c.workers[c.nextWorkerID] = args.WorkerAddress
	c.nextWorkerID++

	fmt.Printf("Coordinator: Registered worker %d at address %s\n",
		reply.WorkerID, args.WorkerAddress)
	return nil
}

// GetMapWorker - returns the address of the worker that completed a map task
func (c *Coordinator) GetMapWorker(args *GetMapWorkerArgs, reply *GetMapWorkerReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if args.MapTaskID >= 0 && args.MapTaskID < len(c.mapTasks) {
		task := &c.mapTasks[args.MapTaskID]
		if task.Status == Completed {
			reply.WorkerAddr = task.WorkerAddress
			reply.Found = true
		}
	}

	return nil
}

// ReportTaskFailure - worker reports that a task failed
func (c *Coordinator) ReportTaskFailure(args *ReportTaskFailureArgs, reply *ReportTaskFailureReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	fmt.Printf("Coordinator: Received task failure report: %s task %d, reason: %s\n",
		args.TaskType, args.TaskID, args.Reason)

	if args.TaskType == "Reduce" && args.FailedMapID >= 0 {
		// Reduce task failed because it couldnt fetch intermediate file from a map task
		// Optimization: Reset ALL map tasks from the dead worker (batch reassignment) PLEASE WORK
		if args.FailedMapID < len(c.mapTasks) {
			deadWorkerAddr := c.mapTasks[args.FailedMapID].WorkerAddress
			deadWorkerID := c.mapTasks[args.FailedMapID].WorkerID

			if deadWorkerAddr != "" {
				// Reset ALL map tasks from this dead worker
				resetCount := 0
				for i := range c.mapTasks {
					if c.mapTasks[i].WorkerAddress == deadWorkerAddr && c.mapTasks[i].Status == Completed {
						c.mapTasks[i].Status = Idle
						c.mapTasks[i].WorkerID = 0
						c.mapTasks[i].WorkerAddress = ""
						resetCount++
					}
				}
				fmt.Printf("Coordinator: Worker %d (at %s) appears dead, reset %d map tasks for batch reassignment\n",
					deadWorkerID, deadWorkerAddr, resetCount)
			}
		}

		// Also reset the reduce task to Idle so it can be retried after map tasks complete
		if args.TaskID >= 0 && args.TaskID < len(c.reduceTasks) {
			reduceTask := &c.reduceTasks[args.TaskID]
			if reduceTask.Status == InProgress {
				fmt.Printf("Coordinator: Resetting reduce task %d to Idle for retry\n", args.TaskID)
				reduceTask.Status = Idle
			}
		}
	}

	reply.Acknowledged = true
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()

	sockname := coordinatorSock()

	var l net.Listener
	var e error

	// check if distributed mode advanced (COOR_ADV is set)
	if os.Getenv("COOR_ADV") != "" {
		// distributed mode - use TCP socket
		l, e = net.Listen("tcp", sockname)
		fmt.Printf("Coordinator: Listening on TCP %s\n", sockname)
	} else {
		// basic mode - use UNIX domain socket
		os.Remove(sockname)
		l, e = net.Listen("unix", sockname)
		fmt.Printf("Coordinator: Listening on UNIX socket %s\n", sockname)
	}

	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// check if all reduce tasks are completed
	for i := range c.reduceTasks {
		if c.reduceTasks[i].Status != Completed {
			return false
		}
	}

	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Validate input parameters
	if nReduce <= 0 {
		log.Fatalf("MakeCoordinator: nReduce must be > 0, got %d", nReduce)
	}
	if len(files) == 0 {
		log.Fatalf("MakeCoordinator: must provide at least one input file")
	}

	c := Coordinator{}

	// Your code here.
	c.nReduce = nReduce
	c.workers = make(map[int]string)
	c.nextWorkerID = 1

	// create a map task for each input file
	c.mapTasks = make([]MapTask, len(files))
	for i, filename := range files {
		c.mapTasks[i] = MapTask{
			ID:       i,
			Filename: filename,
			Status:   Idle,
		}
		fmt.Printf("Coordinator: Created map task %d for file %s\n", i, filename)
	}

	c.reduceTasks = make([]ReduceTask, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = ReduceTask{ID: i, Status: Idle}
	}

	c.server()
	return &c
}
