package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand/v2"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

var id *int = nil

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

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
	if id == nil {
		n := rand.IntN(100)
		id = &n
	}
	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	//CallExample()
	for {
		taskReply := AssignTaskRequest()
		if taskReply == nil {
			fmt.Printf("[%d] taskReply nil.\n", *id)
			continue
		}

		if taskReply.TaskType == "Map" {
			filename := taskReply.Filename
			// deal with file.
			file, err := os.Open(filename)
			if err != nil {
				fmt.Printf("[%d] open file error: %v\n", *id, err)
			}
			content, err := ioutil.ReadAll(file)
			if err != nil {
				fmt.Printf("[%d] read file error: %v\n", *id, err)
			}
			file.Close()
			kva := mapf(filename, string(content))
			reduce := taskReply.NReduce

			// create nReduce tmp files to store intermediate result.
			files := make([]*os.File, reduce)
			jsonEncoders := make([]*json.Encoder, reduce)
			for i := 0; i < reduce; i++ {
				f, _ := ioutil.TempFile(".", "mr-tmp-*")
				jsonEncoders[i] = json.NewEncoder(f)
				files[i] = f
			}

			for _, kv := range kva {
				// get index
				i := ihash(kv.Key) % reduce
				// write to the corresponding file in json format.
				jsonEncoders[i].Encode(&kv)
			}

			for i := 0; i < reduce; i++ {
				files[i].Close()
				os.Rename(files[i].Name(), fmt.Sprintf("mr-%d-%d", taskReply.Id, i))
			}

			// report map task done
			TaskDoneRequest(taskReply)
		} else if taskReply.TaskType == "Reduce" {
			//
			nMap := taskReply.NMap

			// fetch all into a big array and sort.
			intermediate := []KeyValue{}
			for i := 0; i < nMap; i++ {

				// if reduce task number is 0, then: mr-0-0, mr-1-0, mr-2-0.....mr-7-0
				file, _ := os.Open(fmt.Sprintf("mr-%d-%d", i, taskReply.Id))
				// decode from json
				decoder := json.NewDecoder(file)
				for {
					var kv KeyValue
					if err := decoder.Decode(&kv); err != nil {
						break
					}
					intermediate = append(intermediate, kv)
				}
			}
			sort.Sort(ByKey(intermediate))
			oname := fmt.Sprintf("mr-out-%d", taskReply.Id)
			ofile, _ := os.Create(oname)
			i := 0
			for i < len(intermediate) {
				j := i + 1
				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}
				output := reducef(intermediate[i].Key, values)

				// this is the correct format for each line of Reduce output.
				fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

				i = j
			}
			ofile.Close()
			//time.Sleep(150 * time.Millisecond)
			TaskDoneRequest(taskReply)
		} else if taskReply.TaskType == "Wait" {
			time.Sleep(1 * time.Second)
		} else if taskReply.TaskType == "Done" {
			// exit this worker.
			return
		}

	}
}
func AssignTaskRequest() *AssignTaskReply {
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}
	ok := call("Coordinator.AssignTask", &args, &reply)
	if ok {
		fmt.Printf("[%d] AssignTaskRequest reply:%v\n", *id, reply)
		return &reply
	} else {
		fmt.Printf("[%d] AssignTaskRequest failed\n", *id)
		return nil
	}

}

func TaskDoneRequest(task *AssignTaskReply) {
	args := TaskDoneArgs{}
	args.Id = task.Id
	args.TaskType = task.TaskType
	reply := TaskDoneReply{}
	ok := call("Coordinator.TaskDone", &args, &reply)
	if ok {
		fmt.Printf("[%d] TaskDoneRequest reply:%v\n", *id, reply)
	} else {
		fmt.Printf("[%d] TaskDoneRequest failed\n", *id)
	}
	return
}
func DoMapTask() {

}

func DoReduceTask() {

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
