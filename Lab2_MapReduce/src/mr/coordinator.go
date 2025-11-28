package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
)

type Coordinator struct {
	// Your definitions here.
	mutex       sync.Mutex
	nReduce     int
	mapTasks    []MapTask
	reduceTasks []ReduceTask
}

type TaskStatus int

const (
	Idle TaskStatus = iota
	InProgress
	Completed
)

type MapTask struct {
	ID       int
	Filename string
	Status   TaskStatus
}

type ReduceTask struct {
	ID     int
	Status TaskStatus
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
		// find idle map task
		for i := range c.mapTasks {
			if c.mapTasks[i].Status == Idle {
				c.mapTasks[i].Status = InProgress
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
		for i := range c.reduceTasks {
			if c.reduceTasks[i].Status == Idle {
				c.reduceTasks[i].Status = InProgress
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
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// task complete - worker notifies coordinator of task completion
func (c *Coordinator) TaskComplete(args *TaskCompleteArgs, reply *TaskCompleteReply) error {
	// lock mutex before accessing mapTasks and unlock when done
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if args.TaskType == "Map" {
		if args.TaskID >= 0 && args.TaskID < len(c.mapTasks) {
			c.mapTasks[args.TaskID].Status = Completed
			reply.Success = true
			fmt.Printf("Coordinator: Map task %d completed by worker\n", args.TaskID)
		}
	}

	if args.TaskType == "Reduce" {
		if args.TaskID >= 0 && args.TaskID < len(c.reduceTasks) {
			c.reduceTasks[args.TaskID].Status = Completed
			reply.Success = true
			fmt.Printf("Coordinator: Reduce task %d completed by worker\n", args.TaskID)
		}
	}

	return nil
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
	c := Coordinator{}

	// Your code here.
	c.nReduce = nReduce

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
