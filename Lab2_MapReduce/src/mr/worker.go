package mr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sort"
	"time"
)

var myWorkerAddress string
var workerID int

const (
	// Maximum number of map tasks to try when discovering intermediate files
	// Prevents infinite loop if file system behaves unexpectedly
	maxMapTasks = 1000
	// Sleep duration when waiting for tasks
	workerSleepDuration = 1 * time.Second

	// coordinator failure detection
	maxFailedRPCAttempts = 5
	rpcRetryDelay        = 2 * time.Second
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type WorkerRPCHandler struct{}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	// Your worker implementation here.

	// ONLY start RPC server in advanced distributed mode
	if myWorkerAddress != "" {
		startWorkerRPCServer(myWorkerAddress)
		fmt.Printf("Worker [starting]: Started RPC server at %s\n", myWorkerAddress)

		// register worker with coordinator
		args := RegisterWorkerArgs{
			WorkerAddress: myWorkerAddress,
		}
		reply := RegisterWorkerReply{}
		call("Coordinator.RegisterWorker", &args, &reply)
		workerID = reply.WorkerID
		fmt.Printf("Worker %d: Registered with coordinator\n", workerID)
	} else {
		fmt.Println("Worker [starting]: Running in basic mode, no RPC server started")
		workerID = 1
	}
	// worker loop - keep requesting tasks from coordinator until we are done
	// track number of failed RPC attempts to detect coordinator failure
	failedRPCAttempts := 0
	for {
		// ask coordinator for a task
		args := RequestTaskArgs{}
		reply := RequestTaskReply{}

		ok := call("Coordinator.RequestTask", &args, &reply)
		if !ok {
			failedRPCAttempts++
			fmt.Printf("Worker %d: RPC call to request task from coordinator failed (attempt: %d/%d)\n",
				workerID, failedRPCAttempts, maxFailedRPCAttempts)
			if failedRPCAttempts >= maxFailedRPCAttempts {
				// assume coordinator is down, exit worker
				fmt.Printf("Worker %d: COORDINATOR FAILURE DETECTION\n", workerID)
				fmt.Printf("Worker %d: Coordinator seems to be down, exiting\n", workerID)
				return
			}
			// wait before retrying to call coordinator
			fmt.Printf("Worker %d: Waiting %v before retrying...\n", workerID, rpcRetryDelay)
			time.Sleep(rpcRetryDelay)
			continue
		}
		// reset failed RPC attempts counter on successful call
		failedRPCAttempts = 0

		// check what we got of an task type
		if reply.TaskType == "Map" {
			// perform map task
			fmt.Printf("Worker %d: Received Map task %d for file: '%s'\n", workerID, reply.TaskID, reply.Filename)

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
			fmt.Printf("Worker %d: Map task %d produced %v key-value pairs from file: '%s'\n",
				workerID, reply.TaskID, len(kva), reply.Filename)

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

			// write intermediate output files
			for i := 0; i < reply.NReduce; i++ {
				// 1: create temp file for intermediate data
				tmpFile, err := os.CreateTemp("", "mr-tmp-*")
				if err != nil {
					log.Fatalf("cannot create temp file %v", tmpFile.Name())
				}
				tmpFileName := tmpFile.Name()

				// 2: write all data to temp file
				enc := json.NewEncoder(tmpFile)
				for _, kv := range buckets[i] {
					err := enc.Encode(&kv)
					if err != nil {
						// error handling: if anything went wrong, close and delete temp file
						tmpFile.Close()
						os.Remove(tmpFileName)
						log.Fatalf("cannot encode kv pair %v", kv)
					}
				}
				// close temp file before renaming
				tmpFile.Close()

				// 3: atomic rename
				finalFilename := fmt.Sprintf("mr-%d-%d", reply.TaskID, i)
				err = os.Rename(tmpFileName, finalFilename)
				if err != nil {
					log.Fatalf("cannot rename temp file to final file %v", finalFilename)
				}
			}

			fmt.Printf("Worker %d: Wrote %d intermediate files for map task %d\n",
				workerID, reply.NReduce, reply.TaskID)

			// report task completion to coordinator
			completeArgs := TaskCompleteArgs{
				TaskID:   reply.TaskID,
				TaskType: "Map",
				WorkerID: workerID,
			}
			completeReply := TaskCompleteReply{}
			ok = call("Coordinator.TaskComplete", &completeArgs, &completeReply)
			if !ok {
				fmt.Printf("Worker %d: Failed to report map task %d completion\n", workerID, reply.TaskID)
			}
		} else if reply.TaskType == "Reduce" {
			// perform reduce task
			fmt.Printf("Worker %d: Received Reduce task %d\n", workerID, reply.TaskID)

			// FIRST: gather intermediate key-value pairs for this reduce task
			// read all intermediate files thats named mr-*-<reply.TaskID>
			// (all mr-*-Y files where Y = reply.TaskID)
			// example: if TaskID=0, read mr-0-0, mr-1-0, mr-2-0

			intermediate := []KeyValue{}

			// we dont know how many map tasks there were, so we try reading until we cant find more files
			// BACKWARDS COMPATIBLE: try local file first, then RPC fetch from remote worker
			for mapTaskNum := 0; mapTaskNum < maxMapTasks; mapTaskNum++ {
				filename := fmt.Sprintf("mr-%d-%d", mapTaskNum, reply.TaskID)

				// TRY LOCAL FILE FIRST (basic mode)
				content, err := os.ReadFile(filename)

				if err != nil {
					// Local file not found
					// FALLBACK: Try RPC fetch from remote worker (distributed mode)
					if myWorkerAddress != "" {
						// Ask coordinator: which worker completed this map task?
						getWorkerArgs := GetMapWorkerArgs{MapTaskID: mapTaskNum}
						getWorkerReply := GetMapWorkerReply{}

						ok := call("Coordinator.GetMapWorker", &getWorkerArgs, &getWorkerReply)
						if !ok || !getWorkerReply.Found {
							// No worker has this map task, assume no more map tasks
							break
						}

						// RPC to that worker to fetch the intermediate file
						fetchArgs := FetchIntermediateFileArgs{
							MapTaskID:    mapTaskNum,
							ReduceTaskID: reply.TaskID,
						}
						fetchReply := FetchIntermediateFileReply{}

						ok = callWorker(getWorkerReply.WorkerAddr, "WorkerRPCHandler.FetchIntermediateFile", &fetchArgs, &fetchReply)
						if !ok || !fetchReply.Found {
							fmt.Printf("Worker %d: Failed to fetch intermediate file from worker at %s for map task %d\n",
								workerID, getWorkerReply.WorkerAddr, mapTaskNum)
							break
						}

						content = fetchReply.Content
						fmt.Printf("Worker %d: Fetched intermediate file for map task %d from worker at %s\n",
							workerID, mapTaskNum, getWorkerReply.WorkerAddr)
					} else {
						// Basic mode and file not found, assume no more files
						break
					}
				}

				// Decode key-value pairs from content (either local or remote)
				dec := json.NewDecoder(bytes.NewReader(content))
				for {
					var kv KeyValue
					if err := dec.Decode(&kv); err != nil {
						break
					}
					intermediate = append(intermediate, kv)
				}
			}

			// SECOND: sort intermediate key-value pairs by key
			// group by key
			// all same keys sorted adjacently
			// we need to group all values for same key
			// if sorted first then we can easily loop throught and collect values for same key
			sort.Slice(intermediate, func(i, j int) bool {
				return intermediate[i].Key < intermediate[j].Key
			})

			// THIRD: create temp output file for atomic writing
			outputFilename := fmt.Sprintf("mr-out-%d", reply.TaskID)

			// Create temp file
			tmpFile, err := os.CreateTemp("", "mr-out-tmp-*")
			if err != nil {
				log.Fatalf("cannot create temp file")
			}
			tmpFileName := tmpFile.Name()

			// FOURTH: loop through intermediate data och group by key.
			// i points to first occurrence of a key.
			// j finds the last occurrence of the same key.
			// then we collect all values from i to j-1.
			// and call reduce function on that key and its values.
			// then write output to temp file.
			// jumps to next key and we repeat until done.
			i := 0
			for i < len(intermediate) {
				j := i + 1

				// find all values for the same key
				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}

				// all values for key = intermediate[i].Key
				// from index i to j-1
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}

				// run reduce function
				output := reducef(intermediate[i].Key, values)

				// write to temp file
				fmt.Fprintf(tmpFile, "%v %v\n", intermediate[i].Key, output)

				// move to next key
				i = j
			}

			tmpFile.Close()

			// Atomic rename
			err = os.Rename(tmpFileName, outputFilename)
			if err != nil {
				log.Fatalf("cannot rename %v to %v", tmpFileName, outputFilename)
			}

			fmt.Printf("Worker %d: Reduce task %d wrote output file: '%s'\n",
				workerID, reply.TaskID, outputFilename)

			// FIFTH: report task completion to coordinator
			completeArgs := TaskCompleteArgs{
				TaskID:   reply.TaskID,
				TaskType: "Reduce",
				WorkerID: workerID,
			}
			completeReply := TaskCompleteReply{}
			ok = call("Coordinator.TaskComplete", &completeArgs, &completeReply)
			if !ok {
				fmt.Printf("Worker %d: Failed to report reduce task %d completion\n", workerID, reply.TaskID)
			}
		} else if reply.TaskType == "Wait" {
			// no task available, wait and try again
			time.Sleep(workerSleepDuration)
		} else if reply.TaskType == "Exit" {
			// all tasks are done, exit worker
			fmt.Printf("Worker %d: All tasks done, exiting\n", workerID)
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
	sockname := coordinatorSock()

	var c *rpc.Client
	var err error

	if myWorkerAddress != "" {
		// distributed mode - connect via TCP
		c, err = rpc.DialHTTP("tcp", sockname)
	} else {
		// basic mode - connect via Unix domain socket
		c, err = rpc.DialHTTP("unix", sockname)
	}
	if err != nil {
		fmt.Printf("Worker %d: Failed to connect to coordinator: %v\n", workerID, err)
		return false // retry
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

// send an RPC request to another worker, wait for the response.
// returns true on success, false on failure.
func callWorker(address string, rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		fmt.Printf("Worker: Failed to dial worker at %s: %v\n", address, err)
		return false
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Printf("Worker: RPC call to %s failed: %v\n", address, err)
	return false
}

func init() {
	myWorkerAddress = os.Getenv("WORKER_ADV")
}

func startWorkerRPCServer(address string) {
	rpc.Register(&WorkerRPCHandler{})
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Worker RPC server error:", err)
	}

	fmt.Printf("Worker: RPC server listening at %s\n", address)
	go http.Serve(l, nil)
}

func (w *WorkerRPCHandler) FetchIntermediateFile(args *FetchIntermediateFileArgs, reply *FetchIntermediateFileReply) error {
	filename := fmt.Sprintf("mr-%d-%d", args.MapTaskID, args.ReduceTaskID)

	content, err := os.ReadFile(filename)
	if err != nil {
		reply.Found = false
		return nil
	}

	reply.Content = content
	reply.Found = true
	return nil
}
