package mr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	if id == nil {
		n := rand.IntN(100)
		id = &n
	}

	for {
		taskReply := AssignTaskRequest()
		if taskReply == nil {
			fmt.Printf("[%d] taskReply nil.\n", *id)
			continue
		}

		if taskReply.TaskType == MapTask {
			err := DoMapTask(mapf, taskReply)
			if err != nil {
				return
			}
			// report map task done
			TaskDoneRequest(taskReply)
		} else if taskReply.TaskType == ReduceTask {
			err := DoReduceTask(reducef, taskReply)
			if err != nil {
				fmt.Printf("[%d] DoReduceTask failed: %s\n", *id, err)
				return
			}
			// report map task done
			TaskDoneRequest(taskReply)
		} else if taskReply.TaskType == WaitTask {
			time.Sleep(1 * time.Second)
		} else if taskReply.TaskType == ExitTask {
			// exit this worker.
			return
		}

	}
}
func DoMapTask(mapf func(string, string) []KeyValue, taskReply *AssignTaskReply) error {
	filename := taskReply.Filename
	// deal with file.
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("[%d] open file error: %v\n", *id, err)
		return err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Printf("[%d] read file error: %v\n", *id, err)
			return err
		}
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
		os.Rename(files[i].Name(), fmt.Sprintf("mr-%d-%d", taskReply.Id, i))
		files[i].Close()
	}
	return nil
}
func DoReduceTask(reducef func(string, []string) string, taskReply *AssignTaskReply) error {
	nMap := taskReply.NMap

	// fetch all into a big array and sort.
	intermediate := []KeyValue{}
	for i := 0; i < nMap; i++ {

		// if reduce task number is 0, then: mr-0-0, mr-1-0, mr-2-0.....mr-7-0
		file, err := os.Open(fmt.Sprintf("mr-%d-%d", i, taskReply.Id))
		if err != nil {
			fmt.Printf("[%d] open file error: %v\n", *id, err)
		}
		// decode from json
		decoder := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := decoder.Decode(&kv); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Printf("[%d] decoder error: %s\n", *id, err)
				return err
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
	defer ofile.Close()
	return nil
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

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
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
