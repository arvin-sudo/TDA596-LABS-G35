package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// worker loop - keep requesting tasks from coordinator until we are done
	for {
		// ask coordinator for a task
		args := RequestTaskArgs{}
		reply := RequestTaskReply{}

		ok := call("Coordinator.RequestTask", &args, &reply)
		if !ok {
			fmt.Println("Worker: RPC call failed")
			return
		}

		// check what we got of an task type
		if reply.TaskType == "Map" {
			// perform map task
			fmt.Printf("Worker: Received Map task %d for file: '%s'\n", reply.TaskID, reply.Filename)

			// read input file
			file, err := os.Open(reply.Filename)
			if err != nil {
				log.Fatalf("cannot open %v", reply.Filename)
			}
			content, err := io.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", reply.Filename)
			}
			file.Close()

			// run map function
			kva := mapf(reply.Filename, string(content))
			fmt.Printf("Worker: Map task %d produced %v key-value pairs from file: '%s'\n",
				reply.TaskID, len(kva), reply.Filename)

			// create buckets for each reduce task
			buckets := make([][]KeyValue, reply.NReduce)
			for i := range buckets {
				buckets[i] = []KeyValue{}
			}

			// partition key-value pairs into buckets
			for _, kv := range kva {
				bucket := ihash(kv.Key) % reply.NReduce
				buckets[bucket] = append(buckets[bucket], kv)
			}

			// write intermediate files
			for i := 0; i < reply.NReduce; i++ {
				filename := fmt.Sprintf("mr-%d-%d", reply.TaskID, i)
				file, err := os.Create(filename)
				if err != nil {
					log.Fatalf("cannot create %v", filename)
				}

				enc := json.NewEncoder(file)
				for _, kv := range buckets[i] {
					err := enc.Encode(&kv)
					if err != nil {
						log.Fatalf("cannot encode kv pair %v", kv)
					}
				}
				file.Close()
			}

			fmt.Printf("Worker: Wrote %d intermediate files for task %d\n",
				reply.NReduce, reply.TaskID)

			// report task completion to coordinator
			completeArgs := TaskCompleteArgs{
				TaskID:   reply.TaskID,
				TaskType: "Map",
			}
			completeReply := TaskCompleteReply{}
			ok = call("Coordinator.TaskComplete", &completeArgs, &completeReply)
			if !ok {
				fmt.Printf("Worker: Failed to report task %d completion\n", reply.TaskID)
			}

		} else if reply.TaskType == "Wait" {
			// no task available, wait and try again
			time.Sleep(time.Second)
		} else if reply.TaskType == "Exit" {
			// all tasks are done, exit worker
			fmt.Println("Worker: All tasks done, exiting")
			return
		}

	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
