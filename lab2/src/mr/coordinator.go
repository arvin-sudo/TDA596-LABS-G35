package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

const LIMIT_TIME = 10

type Coordinator struct {
	nReduce     int
	taskPhase   string // "Map", "WaitMap", "Reduce", "WaitReduce", "Done"
	mapTasks    []*Task
	reduceTasks []*Task

	// for locking taskPhase
	mutex sync.Mutex
}

type Task struct {
	id         int    // task number
	taskStatus string // "Idle", "InProgress", "Done"
	assignTime time.Time
	taskType   string // "Map", "Wait", "Reduce", "Exit"

	filename string // only for Map task
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) AssignTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	c.mutex.Lock()
	taskPhase := c.taskPhase
	c.mutex.Unlock()
	if taskPhase == "Map" {
		// 1. iterate map tasks, find an available one or expired one and return
		for _, mapTask := range c.mapTasks {
			if mapTask.taskStatus == "Idle" { // "Idle", "InProgress", "Done"
				mapTask.assignTime = time.Now()
				mapTask.taskStatus = "InProgress"

				reply.Id = mapTask.id
				reply.TaskType = "Map"
				reply.Filename = mapTask.filename
				reply.NReduce = c.nReduce
				reply.NMap = len(c.mapTasks)
				return nil
			} else if mapTask.taskStatus == "InProgress" {
				// check expired ? consider reassign
				assignTime := mapTask.assignTime
				if time.Now().After(assignTime.Add(time.Duration(LIMIT_TIME) * time.Second)) {
					// reassign
					mapTask.assignTime = time.Now()
					mapTask.taskStatus = "InProgress"

					reply.Id = mapTask.id
					reply.TaskType = "Map"
					reply.Filename = mapTask.filename
					reply.NReduce = c.nReduce
					reply.NMap = len(c.mapTasks)
					return nil
				}
			}
		}

		time.Sleep(time.Second)

		c.mutex.Lock()
		c.taskPhase = "WaitMap"
		c.mutex.Unlock()
		//needToWait := false
		//for _, mapTask := range c.mapTasks {
		//	if mapTask.taskStatus == "InProgress" {
		//		needToWait = true
		//	}
		//}
		//
		//if needToWait {
		//	c.taskPhase = "Wait"
		//} else {
		//	c.taskPhase = "Reduce"
		//}

	} else if taskPhase == "Reduce" {
		// 2. if all Map tasks done, then assgin a Reduce task.
		for _, reduceTask := range c.reduceTasks {
			if reduceTask.taskStatus == "Idle" {
				reduceTask.assignTime = time.Now()
				reduceTask.taskStatus = "InProgress"

				reply.Id = reduceTask.id
				reply.TaskType = "Reduce"
				reply.NReduce = c.nReduce
				reply.NMap = len(c.mapTasks)
				return nil
			}
		}

		time.Sleep(time.Second)

		c.mutex.Lock()
		c.taskPhase = "WaitReduce"
		c.mutex.Unlock()
	} else if taskPhase == "WaitMap" {
		needToWait := false
		for _, mapTask := range c.mapTasks {
			if mapTask.taskStatus == "InProgress" {
				assignTime := mapTask.assignTime
				if time.Now().After(assignTime.Add(time.Duration(LIMIT_TIME) * time.Second)) {
					// reassign
					mapTask.assignTime = time.Now()
					mapTask.taskStatus = "InProgress"

					reply.Id = mapTask.id
					reply.TaskType = "Map"
					reply.Filename = mapTask.filename
					reply.NReduce = c.nReduce
					reply.NMap = len(c.mapTasks)
					return nil
				}
				needToWait = true
			}
		}

		if needToWait {
			c.mutex.Lock()
			c.taskPhase = "WaitMap"
			c.mutex.Unlock()
		} else {
			c.mutex.Lock()
			c.taskPhase = "Reduce"
			c.mutex.Unlock()
		}

		reply.Id = 0
		reply.TaskType = "Wait"
		// send a wait type task
		return nil
	} else if taskPhase == "WaitReduce" {
		needToWait := false
		for _, reduceTask := range c.reduceTasks {
			if reduceTask.taskStatus == "InProgress" {
				assignTime := reduceTask.assignTime
				if time.Now().After(assignTime.Add(time.Duration(LIMIT_TIME) * time.Second)) {
					// reassign
					reduceTask.assignTime = time.Now()
					reduceTask.taskStatus = "InProgress"

					reply.Id = reduceTask.id
					reply.TaskType = "Reduce"
					reply.NReduce = c.nReduce
					reply.NMap = len(c.mapTasks)
					return nil
				}
				needToWait = true
			}
		}

		if needToWait {
			c.mutex.Lock()
			c.taskPhase = "WaitReduce"
			c.mutex.Unlock()
		} else {
			c.mutex.Lock()
			c.taskPhase = "Done"
			c.mutex.Unlock()
		}

		reply.Id = 0
		reply.TaskType = "Wait"
		// send a task with type WaitReduce
		return nil
	} else if taskPhase == "Done" {
		reply.Id = 0
		reply.TaskType = "Done"
		// send a task with type Done
		return nil
	}
	return nil
}

func (c *Coordinator) TaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	if args.TaskType == "Map" {
		c.mapTasks[args.Id].taskStatus = "Done"
	} else if args.TaskType == "Reduce" {
		c.reduceTasks[args.Id].taskStatus = "Done"
	}
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

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	c.mutex.Lock()
	taskPhase := c.taskPhase
	c.mutex.Unlock()
	if taskPhase == "Done" {
		return true
	}

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	c.nReduce = nReduce
	c.taskPhase = "Map"
	c.mapTasks = make([]*Task, len(files))
	c.reduceTasks = make([]*Task, nReduce)

	// initialise Map tasks
	for i := 0; i < len(files); i++ {
		task := &Task{
			id:         i,
			taskStatus: "Idle",
			taskType:   "Map",
			filename:   files[i],
		}
		c.mapTasks[i] = task
	}

	// initialise Reduce tasks
	for i := 0; i < nReduce; i++ {
		task := &Task{
			id:         i,
			taskStatus: "Idle",
			taskType:   "Reduce",
		}
		c.reduceTasks[i] = task
	}

	c.server()
	return &c
}
