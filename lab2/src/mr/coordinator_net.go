package mr

import (
	"fmt"
	"log"
	"time"
)
import "net"
import "net/rpc"
import "net/http"

var coorfileServerPort int
var coordinatorFilePath = "/Users/xinyi/Documents/old_from_mac_2017/daryl/cth/tda596_distributedsystem/lab_demo/lab2/src/main"

type CoordinatorNet struct {
	Coordinator

	// store like {0:"172.31.77.237/intermediate"}
	// the intermediate files would be in mr-0-* under intermediate folder.
	workerMapTaskAddrMap map[int]string // key is map task number, value is the address.
}

func (c *CoordinatorNet) AssignTaskHTTP(args *AssignTaskArgs, reply *AssignTaskReplyHTTP) error {
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
				reply.Port = coorfileServerPort
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
					reply.Port = coorfileServerPort
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
				reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
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
					reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
					reply.Port = coorfileServerPort

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
					reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
					reply.Port = coorfileServerPort

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

func (c *CoordinatorNet) TaskDoneHTTP(args *TaskDoneArgsHTTP, reply *TaskDoneReply) error {
	if args.TaskType == "Map" {
		c.mapTasks[args.Id].taskStatus = "Done"

		// key: task id  value: address(e.g "172.31.69.101/home/ubuntu/lab2")
		c.workerMapTaskAddrMap[args.Id] = args.MyAddress
	} else if args.TaskType == "Reduce" {
		c.reduceTasks[args.Id].taskStatus = "Done"
	}
	return nil
}

// the RPC argument and reply types are defined in rpc.go.
// start a thread that listens for RPCs from worker.go
func (c *CoordinatorNet) serverByHttp() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	//sockname := coordinatorSock()
	//os.Remove(sockname)
	//l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinatorNet(files []string, nReduce int) *CoordinatorNet {
	// file server
	go func() {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			fmt.Printf("http server listen err: %s \n", err)
		}
		coorfileServerPort = listener.Addr().(*net.TCPAddr).Port
		fmt.Printf("coordinator port: %d\n", coorfileServerPort)
		http.Serve(listener, http.FileServer(http.Dir(coordinatorFilePath)))
		defer listener.Close()

	}()

	c := CoordinatorNet{}
	c.nReduce = nReduce
	c.taskPhase = "Map"
	c.mapTasks = make([]*Task, len(files))
	c.reduceTasks = make([]*Task, nReduce)
	c.workerMapTaskAddrMap = make(map[int]string)

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

	c.serverByHttp()
	return &c
}
