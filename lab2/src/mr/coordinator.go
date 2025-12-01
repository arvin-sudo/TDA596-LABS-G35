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

const ExpireTimeSecond = 10

type CoordinatorPhase string

const (
	OngoingMap          CoordinatorPhase = "Map"
	WaitingMapFinish    CoordinatorPhase = "WaitMap"
	OngoingReduce       CoordinatorPhase = "Reduce"
	WaitingReduceFinish CoordinatorPhase = "WaitReduce"
	AllDone             CoordinatorPhase = "Done"
)

// keep record of thoses tasks' info and manage which phase should be.
type Coordinator struct {
	nReduce          int
	coordinatorPhase CoordinatorPhase // state machine, e.g "Map", "WaitMap", "Reduce", "WaitReduce", "Done"
	mapTasks         []*Task
	reduceTasks      []*Task

	// for locking coordinatorPhase
	mutex sync.Mutex
}

type TaskType string

const (
	MapTask    TaskType = "Map"
	WaitTask   TaskType = "Wait"
	ReduceTask TaskType = "Reduce"
	ExitTask   TaskType = "Exit"
)

type TaskStatus string

const (
	Idle       TaskStatus = "Idle"
	InProgress TaskStatus = "InProgress"
	Done       TaskStatus = "Done"
)

// task pass to worker, let worker do something depends.
type Task struct {
	id         int        // task number
	taskStatus TaskStatus // e.g "Idle", "InProgress", "Done"
	assignTime time.Time
	taskType   TaskType // e.g "Map", "Wait", "Reduce", "Exit"

	filename string // only for Map task
}

// -----------------RPC handlers for the worker to call.------------------
func (c *Coordinator) AssignTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	coordinatorPhase := c.coordinatorPhase
	if coordinatorPhase == OngoingMap {
		err := c.assignMapTask(args, reply)
		if err != nil {
			return err
		}
	} else if coordinatorPhase == OngoingReduce {
		err := c.assignReduceTask(args, reply)
		if err != nil {
			return err
		}
	} else if coordinatorPhase == WaitingMapFinish {
		err := c.waitingMapFinish(args, reply)
		if err != nil {
			return err
		}
	} else if coordinatorPhase == WaitingReduceFinish {
		err := c.waitingReduceFinish(args, reply)
		if err != nil {
			return err
		}
	} else if coordinatorPhase == AllDone {
		reply.Id = 0
		reply.TaskType = ExitTask
		return nil
	}
	return nil
}

func (c *Coordinator) TaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	if args.TaskType == MapTask {
		c.mapTasks[args.Id].taskStatus = Done
	} else if args.TaskType == ReduceTask {
		c.reduceTasks[args.Id].taskStatus = Done
	}
	return nil
}

// -----------------RPC handlers ends.------------------

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	coordinatorPhase := c.coordinatorPhase
	if coordinatorPhase == AllDone {
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
	c.coordinatorPhase = OngoingMap
	c.mapTasks = make([]*Task, len(files))
	c.reduceTasks = make([]*Task, nReduce)

	// initialise Map tasks
	for i := 0; i < len(files); i++ {
		task := &Task{
			id:         i,
			taskStatus: Idle,
			taskType:   MapTask,
			filename:   files[i],
		}
		c.mapTasks[i] = task
	}

	// initialise Reduce tasks
	for i := 0; i < nReduce; i++ {
		task := &Task{
			id:         i,
			taskStatus: Idle,
			taskType:   ReduceTask,
		}
		c.reduceTasks[i] = task
	}

	c.server()
	return &c
}

func (c *Coordinator) assignMapTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	// 1. iterate map tasks, find an available one or expired one and return
	for _, mapTask := range c.mapTasks {
		if mapTask.taskStatus == Idle { // "Idle", "InProgress", "Done"
			mapTask.assignTime = time.Now()
			mapTask.taskStatus = InProgress

			reply.Id = mapTask.id
			reply.TaskType = MapTask
			reply.Filename = mapTask.filename
			reply.NReduce = c.nReduce
			reply.NMap = len(c.mapTasks)
			return nil
		} else if mapTask.taskStatus == InProgress {
			// check expired ? consider reassign
			assignTime := mapTask.assignTime
			if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
				// reassign
				mapTask.assignTime = time.Now()
				mapTask.taskStatus = InProgress

				reply.Id = mapTask.id
				reply.TaskType = MapTask
				reply.Filename = mapTask.filename
				reply.NReduce = c.nReduce
				reply.NMap = len(c.mapTasks)
				return nil
			}
		}
	}

	// if not found, then send a Wait task
	reply.Id = 0
	reply.TaskType = WaitTask

	c.coordinatorPhase = WaitingMapFinish
	return nil
}

func (c *Coordinator) assignReduceTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	// 1. if all Map tasks done, then assgin a Reduce task.
	for _, reduceTask := range c.reduceTasks {
		if reduceTask.taskStatus == Idle {
			reduceTask.assignTime = time.Now()
			reduceTask.taskStatus = InProgress

			reply.Id = reduceTask.id
			reply.TaskType = ReduceTask
			reply.NReduce = c.nReduce
			reply.NMap = len(c.mapTasks)
			return nil
		}
	}

	// 2. if all Reduce tasks were assgined, then send a Wait task
	reply.Id = 0
	reply.TaskType = WaitTask

	c.coordinatorPhase = WaitingReduceFinish
	return nil
}

func (c *Coordinator) waitingMapFinish(args *AssignTaskArgs, reply *AssignTaskReply) error {
	needToWait := false
	for _, mapTask := range c.mapTasks {
		if mapTask.taskStatus == InProgress {
			assignTime := mapTask.assignTime
			if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
				// reassign
				mapTask.assignTime = time.Now()
				mapTask.taskStatus = InProgress

				reply.Id = mapTask.id
				reply.TaskType = MapTask
				reply.Filename = mapTask.filename
				reply.NReduce = c.nReduce
				reply.NMap = len(c.mapTasks)
				return nil
			}
			needToWait = true
		}
	}

	if needToWait {
		c.coordinatorPhase = WaitingMapFinish
	} else {
		c.coordinatorPhase = OngoingReduce
	}

	reply.Id = 0
	reply.TaskType = WaitTask
	// send a wait type task
	return nil
}

func (c *Coordinator) waitingReduceFinish(args *AssignTaskArgs, reply *AssignTaskReply) error {
	needToWait := false
	for _, reduceTask := range c.reduceTasks {
		if reduceTask.taskStatus == InProgress {
			assignTime := reduceTask.assignTime
			if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
				// reassign
				reduceTask.assignTime = time.Now()
				reduceTask.taskStatus = InProgress

				reply.Id = reduceTask.id
				reply.TaskType = ReduceTask
				reply.NReduce = c.nReduce
				reply.NMap = len(c.mapTasks)
				return nil
			}
			needToWait = true
		}
	}

	if needToWait {
		c.coordinatorPhase = WaitingReduceFinish
		reply.Id = 0
		reply.TaskType = WaitTask
	} else {
		c.coordinatorPhase = AllDone
		reply.Id = 0
		reply.TaskType = ExitTask
	}

	return nil
}
