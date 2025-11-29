package mr

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)
import "log"
import "net/rpc"

var fileServerPort int

var filePath = "."
var myIPAddress *string

var coordinatorIP = "172.31.69.101"

// main/mrworker.go calls this function.
func WorkerHTTP(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	if id == nil {
		n := rand.IntN(100)
		id = &n
	}
	go func() {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			fmt.Printf("http server listen err: %s \n", err)
		}
		fileServerPort = listener.Addr().(*net.TCPAddr).Port
		fmt.Printf("port: %d\n", fileServerPort)
		http.Serve(listener, http.FileServer(http.Dir(filePath)))
		defer listener.Close()
	}()

	for {
		taskReply := AssignTaskRequestHTTP()
		if taskReply == nil {
			fmt.Printf("[%d] taskReply nil.\n", *id)
			continue
		}

		// slow down a bit, convenient to test crash some tasks.
		time.Sleep(5 * time.Second)

		if taskReply.TaskType == MapTask {
			filename := taskReply.Filename
			port := taskReply.Port

			resp, err := http.Get("http://" + coordinatorIP + ":" + strconv.Itoa(port) + "/" + filename)
			if err != nil {
				fmt.Printf("[%d] http get err: %s \n", *id, err)
				continue
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("[%d] http read body err: %s \n", *id, err)
			}

			kva := mapf(filename, string(body))
			reduce := taskReply.NReduce

			// create nReduce tmp files to store intermediate result.
			files := make([]*os.File, reduce)
			jsonEncoders := make([]*json.Encoder, reduce)
			for i := 0; i < reduce; i++ {
				f, _ := ioutil.TempFile(filePath, "mr-tmp-*")
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
			TaskDoneRequestHTTP(taskReply)
		} else if taskReply.TaskType == ReduceTask {
			nMap := taskReply.NMap

			// fetch all into a big array and sort.
			intermediate := []KeyValue{}
			for i := 0; i < nMap; i++ {

				// fetch from different address
				addrMap := taskReply.WorkerMapTaskAddrMap
				addr := addrMap[i]
				resp, err := http.Get(fmt.Sprintf("%s/mr-%d-%d", addr, i, taskReply.Id))
				if err != nil {
					fmt.Printf("[%d] http get err: %s \n", *id, err)
					continue
				}

				decoder := json.NewDecoder(resp.Body)
				defer resp.Body.Close()

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
			TaskDoneRequestHTTP(taskReply)
		} else if taskReply.TaskType == WaitTask {
			time.Sleep(1 * time.Second)
		} else if taskReply.TaskType == ExitTask {
			// exit this worker.
			return
		}

	}
}

func AssignTaskRequestHTTP() *AssignTaskReplyHTTP {
	args := AssignTaskArgs{}
	reply := AssignTaskReplyHTTP{}
	ok := callHTTP("CoordinatorNet.AssignTaskHTTP", &args, &reply)
	if ok {
		fmt.Printf("[%d] AssignTaskRequestHTTP reply:%v\n", *id, reply)
		return &reply
	} else {
		fmt.Printf("[%d] Detect Coordinator failure. AssignTaskRequestHTTP failed. \n", *id)
		return nil
	}
}

func TaskDoneRequestHTTP(task *AssignTaskReplyHTTP) {
	args := TaskDoneArgsHTTP{}
	args.Id = task.Id
	args.TaskType = task.TaskType
	myIPAddress_ := getMyIpAddress()
	fileAddress := fmt.Sprintf("http://%s:%d", myIPAddress_, fileServerPort)
	fmt.Print(fileAddress)
	args.MyAddress = fileAddress
	reply := TaskDoneReply{}
	ok := callHTTP("CoordinatorNet.TaskDoneHTTP", &args, &reply)
	if ok {
		fmt.Printf("[%d] TaskDoneRequestHTTP reply:%v\n", *id, reply)
	} else {
		fmt.Printf("[%d] TaskDoneRequestHTTP failed\n", *id)
	}
	return
}

func getMyIpAddress() string {
	if myIPAddress != nil {
		return *myIPAddress
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, interf := range interfaces {
		if !strings.Contains(interf.Name, "docker") {
			addrs, _ := interf.Addrs()
			if len(addrs) > 0 {
				//fmt.Println(addrs)
				if ipnet, ok := addrs[0].(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ip := ipnet.IP.String()
						myIPAddress = &ip
						return ip
					}
				}
			}
		}
	}
	return "127.0.0.1"
}

func callHTTP(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", coordinatorIP+":1234")
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
