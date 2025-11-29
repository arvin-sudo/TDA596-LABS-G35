package mr

import (
	"fmt"
	"log"
)
import "net"
import "net/rpc"
import "net/http"

var coorfileServerPort int

// var coordinatorFilePath = "/Users/xinyi/Documents/old_from_mac_2017/daryl/cth/tda596_distributedsystem/lab_demo/lab2/src/main"
var coordinatorFilePath = "."

type CoordinatorNet struct {
	Coordinator

	// store like {0:"172.31.77.237:32212"}
	// the intermediate files would be in mr-0-*.
	workerMapTaskAddrMap map[int]string // key is map task number, value is the address.
}

// -----------------RPC handlers for the worker to call.------------------
func (c *CoordinatorNet) AssignTaskHTTP(args *AssignTaskArgs, reply *AssignTaskReplyHTTP) error {
	c.mutex.Lock()
	coordinatorPhase := c.coordinatorPhase
	c.mutex.Unlock()
	if coordinatorPhase == OngoingMap {
		basicTaskReply := &AssignTaskReply{}
		err := c.assignMapTask(args, basicTaskReply)
		if err != nil {
			return err
		}
		reply.AssignTaskReply = *basicTaskReply
		reply.Port = coorfileServerPort

		//// 1. iterate map tasks, find an available one or expired one and return
		//for _, mapTask := range c.mapTasks {
		//	if mapTask.taskStatus == Idle { // "Idle", "InProgress", "Done"
		//		mapTask.assignTime = time.Now()
		//		mapTask.taskStatus = InProgress
		//
		//		reply.Id = mapTask.id
		//		reply.TaskType = MapTask
		//		reply.Filename = mapTask.filename
		//		reply.Port = coorfileServerPort
		//		reply.NReduce = c.nReduce
		//		reply.NMap = len(c.mapTasks)
		//		return nil
		//	} else if mapTask.taskStatus == InProgress {
		//		// check expired ? consider reassign
		//		assignTime := mapTask.assignTime
		//		if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
		//			// reassign
		//			mapTask.assignTime = time.Now()
		//			mapTask.taskStatus = InProgress
		//
		//			reply.Id = mapTask.id
		//			reply.TaskType = MapTask
		//			reply.Filename = mapTask.filename
		//			reply.Port = coorfileServerPort
		//			reply.NReduce = c.nReduce
		//			reply.NMap = len(c.mapTasks)
		//			return nil
		//		}
		//	}
		//}
		//
		//time.Sleep(time.Second)
		//
		//c.mutex.Lock()
		//c.coordinatorPhase = WaitingMapFinish
		//c.mutex.Unlock()
	} else if coordinatorPhase == OngoingReduce {
		basicTaskReply := &AssignTaskReply{}
		err := c.assignReduceTask(args, basicTaskReply)
		if err != nil {
			return err
		}
		reply.AssignTaskReply = *basicTaskReply
		reply.Port = coorfileServerPort
		reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap

		//// 2. if all Map tasks done, then assgin a Reduce task.
		//for _, reduceTask := range c.reduceTasks {
		//	if reduceTask.taskStatus == Idle {
		//		reduceTask.assignTime = time.Now()
		//		reduceTask.taskStatus = InProgress
		//
		//		reply.Id = reduceTask.id
		//		reply.TaskType = ReduceTask
		//		reply.NReduce = c.nReduce
		//		reply.NMap = len(c.mapTasks)
		//		reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
		//		return nil
		//	}
		//}
		//
		//time.Sleep(time.Second)
		//
		//c.mutex.Lock()
		//c.coordinatorPhase = WaitingReduceFinish
		//c.mutex.Unlock()
	} else if coordinatorPhase == WaitingMapFinish {
		basicTaskReply := &AssignTaskReply{}
		err := c.waitingMapFinish(args, basicTaskReply)
		if err != nil {
			return err
		}
		reply.AssignTaskReply = *basicTaskReply
		reply.Port = coorfileServerPort
		reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap

		//needToWait := false
		//for _, mapTask := range c.mapTasks {
		//	if mapTask.taskStatus == InProgress {
		//		assignTime := mapTask.assignTime
		//		if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
		//			// reassign
		//			mapTask.assignTime = time.Now()
		//			mapTask.taskStatus = InProgress
		//
		//			reply.Id = mapTask.id
		//			reply.TaskType = MapTask
		//			reply.Filename = mapTask.filename
		//			reply.NReduce = c.nReduce
		//			reply.NMap = len(c.mapTasks)
		//			reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
		//			reply.Port = coorfileServerPort
		//
		//			return nil
		//		}
		//		needToWait = true
		//	}
		//}
		//
		//if needToWait {
		//	c.mutex.Lock()
		//	c.coordinatorPhase = WaitingReduceFinish
		//	c.mutex.Unlock()
		//} else {
		//	c.mutex.Lock()
		//	c.coordinatorPhase = OngoingReduce
		//	c.mutex.Unlock()
		//}
		//
		//reply.Id = 0
		//reply.TaskType = WaitTask
		//// send a wait type task
		//return nil
	} else if coordinatorPhase == WaitingReduceFinish {
		basicTaskReply := &AssignTaskReply{}
		err := c.waitingReduceFinish(args, basicTaskReply)
		if err != nil {
			return err
		}
		reply.AssignTaskReply = *basicTaskReply
		reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
		reply.Port = coorfileServerPort

		//needToWait := false
		//for _, reduceTask := range c.reduceTasks {
		//	if reduceTask.taskStatus == InProgress {
		//		assignTime := reduceTask.assignTime
		//		if time.Now().After(assignTime.Add(time.Duration(ExpireTimeSecond) * time.Second)) {
		//			// reassign
		//			reduceTask.assignTime = time.Now()
		//			reduceTask.taskStatus = InProgress
		//
		//			reply.Id = reduceTask.id
		//			reply.TaskType = ReduceTask
		//			reply.NReduce = c.nReduce
		//			reply.NMap = len(c.mapTasks)
		//			reply.WorkerMapTaskAddrMap = c.workerMapTaskAddrMap
		//			reply.Port = coorfileServerPort
		//
		//			return nil
		//		}
		//		needToWait = true
		//	}
		//}
		//
		//if needToWait {
		//	c.mutex.Lock()
		//	c.coordinatorPhase = WaitingReduceFinish
		//	c.mutex.Unlock()
		//} else {
		//	c.mutex.Lock()
		//	c.coordinatorPhase = AllDone
		//	c.mutex.Unlock()
		//}
		//
		//reply.Id = 0
		//reply.TaskType = WaitTask
		//return nil
	} else if coordinatorPhase == AllDone {
		reply.Id = 0
		reply.TaskType = ExitTask
		return nil
	}
	return nil
}

func (c *CoordinatorNet) TaskDoneHTTP(args *TaskDoneArgsHTTP, reply *TaskDoneReply) error {
	if args.TaskType == MapTask {
		c.mapTasks[args.Id].taskStatus = Done

		// key: task id  value: address(e.g "172.31.69.101/home/ubuntu/lab2")
		c.workerMapTaskAddrMap[args.Id] = args.MyAddress
	} else if args.TaskType == ReduceTask {
		c.reduceTasks[args.Id].taskStatus = Done
	}
	return nil
}

// -----------------RPC handlers ends.------------------

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
	c.coordinatorPhase = OngoingMap
	c.mapTasks = make([]*Task, len(files))
	c.reduceTasks = make([]*Task, nReduce)
	c.workerMapTaskAddrMap = make(map[int]string)

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

	c.serverByHttp()
	return &c
}
