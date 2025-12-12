// Node-Peer

package main

import (
	"bufio"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"sync"
)

type Node struct {
	mu             sync.RWMutex
	ID             *big.Int
	IP             string            // IP:PORT Address
	Successor      []*NodeInfo       // list of nodes in ring
	Predecessor    *NodeInfo         // previous node in ring
	Bucket         map[string]string // Data-storage
	FingerTable    []*NodeInfo       // make Chord Lookup faster from O(N) to O(Log n)
	SuccessorCount int               // number of successors to keep track of
}

// NodeInfo = information about a remote node
type NodeInfo struct {
	ID *big.Int
	IP string // IP:PORT Address of remote Node
}

// create new node
func NewNode(ip string, port int, successorCount int) *Node {
	ipAddress := fmt.Sprintf("%s:%d", ip, port)

	node := &Node{
		ID:             Hash(ipAddress),
		IP:             ipAddress,
		Successor:      make([]*NodeInfo, 0, successorCount),
		Predecessor:    nil,
		Bucket:         make(map[string]string),
		FingerTable:    make([]*NodeInfo, KeySize+1),
		SuccessorCount: successorCount,
	}

	return node
}

// Start RPC server
func (n *Node) StartRPCServer() error {
	// Register this node for RPC
	rpc.Register(n)
	rpc.HandleHTTP()

	// Listen on our address
	listener, err := net.Listen("tcp", n.IP)
	if err != nil {
		return err
	}

	// Start serving in goroutine
	go func() {
		if err := http.Serve(listener, nil); err != nil {
			fmt.Printf("RPC Server error: %v\n", err)
		}
	}()

	// Server is now listening (net.Listen completed successfully)
	fmt.Printf("RPC Server listening on IP: %s\n", n.IP)

	return nil
}

// print complete state of this node
func (n *Node) PrintState() {
	n.mu.RLock()
	defer n.mu.RUnlock()

	fmt.Printf("\n======= NODE-STATE =========\n")

	// own node info
	fmt.Printf("Node ID: %s\n", IDToString(n.ID))
	fmt.Printf("Node IP: %s\n", n.IP)

	// successor info
	fmt.Println("\n----- SUCCESSOR NODE -------")
	if len(n.Successor) > 0 {
		fmt.Printf("Successor ID: %s\n", IDToString(n.Successor[0].ID))
		fmt.Printf("Successor IP: %s\n", n.Successor[0].IP)
	} else {
		fmt.Printf("Successor Node: None\n")
	}

	// predecessor info
	fmt.Println("\n----- PREDECESSOR NODE -----")
	if n.Predecessor != nil {
		fmt.Printf("Predecessor ID: %s\n", IDToString(n.Predecessor.ID))
		fmt.Printf("Predecessor IP: %s\n", n.Predecessor.IP)
	} else {
		fmt.Printf("Predecessor Node: None\n")
	}

	// fingerTable info
	fmt.Println("\n------- FINGER TABLE -------")
	prevIP := ""
	for i := 1; i <= KeySize; i++ {
		if n.FingerTable[i] != nil {
			currentIP := n.FingerTable[i].IP
			// only print when IP changes (skip duplicates)
			if currentIP != prevIP {
				fmt.Printf("Finger[%d]: %s (ID: %s)\n", i, currentIP, IDToString(n.FingerTable[i].ID))
				prevIP = currentIP
			}
		}
	}
	if prevIP == "" {
		fmt.Println("Finger Table: None")
	}

	// data storage info
	fmt.Println("\n------- DATA BUCKET --------")
	if len(n.Bucket) <= 0 {
		fmt.Printf("Data Stored: None\n")
	} else {
		fmt.Printf("Data Stored: %d\n", len(n.Bucket))
		for key := range n.Bucket {
			fmt.Printf("%s\n", key)
		}
	}

	fmt.Printf("============================\n")
	fmt.Println()
}

// ping - rpc method to check if node is alive
func (n *Node) Ping(args *EmptyArgs, reply *PingReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	reply.NodeID = IDToString(n.ID)
	reply.NodeIP = n.IP
	return nil
}

// PUT - rpc method to store data
func (n *Node) Put(args *PutArgs, reply *PutReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Bucket[args.Key] = args.Value
	fmt.Printf("PUT: Stored [%s]\n", args.Key)
	return nil
}

// GET - rpc method to fetch data content
func (n *Node) Get(args *GetArgs, reply *GetReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	value, exists := n.Bucket[args.Key]

	if !exists {
		reply.Value = ""
		reply.Found = false
		fmt.Printf("GET: Not found [%s]\n", args.Key)
	} else {
		reply.Value = value
		reply.Found = true
		fmt.Printf("GET: Found [%s]\n", args.Key)
	}

	return nil
}

// create a new chord ring (one alone node)
func (n *Node) Create() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// when creating a new ring (alone), all r successors point to ourselves
	self := &NodeInfo{
		ID: n.ID,
		IP: n.IP,
	}

	n.Successor = make([]*NodeInfo, n.SuccessorCount)
	for i := 0; i < n.SuccessorCount; i++ {
		n.Successor[i] = self
	}

	// initialize finger table - all fingers point to ourselves since we are the only node
	for i := 1; i <= KeySize; i++ {
		n.FingerTable[i] = self
	}

	fmt.Println("NEW CHORD RING CREATED")
}

// closestPrecedingNode - find the closest node in finger table that precedes ID
// loop backwards in finger table (160 down to 1) - check if finger is between me and target
func (n *Node) closestPrecedingNode(id *big.Int) *NodeInfo {
	// search finger table from furthest to closest
	for i := KeySize; i >= 1; i-- {
		finger := n.FingerTable[i]

		// skip if finger is nil or points to this node (n.IP)
		if finger == nil || finger.IP == n.IP {
			continue
		}

		// check if this finger is between this node and target id: n.ID < finger.ID < id (not inclusive)
		if InBetween(finger.ID, n.ID, id, false) {
			return finger
		}
	}

	// no better node found in finger table, return this nodes successor
	return n.Successor[0]
}

// FindSuccessor - RPC method to find the successor node of an ID
func (n *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	id := args.ID

	// if ID is between itself and its successor, its successor is the answer
	if InBetween(id, n.ID, n.Successor[0].ID, true) {
		reply.Node = n.Successor[0]
		return nil
	}

	// otherwise, find the closest preceding node to forward the query to
	reply.Node = n.closestPrecedingNode(id)
	return nil
}

// join an existing chord ring
func (n *Node) Join(bootstrapNode string) error {
	// step 1: find my successor by asking bootstrap node (without lock)
	var findReply FindSuccessorReply
	err := CallNode(bootstrapNode, "Node.FindSuccessor", &FindSuccessorArgs{ID: n.ID}, &findReply)
	if err != nil {
		return fmt.Errorf("Failed to contact bootstrap node: %v", err)
	}

	firstSuccessor := findReply.Node

	// step 2: get successors successor list to build my successor list
	var listReply GetSuccessorListReply
	err = CallNode(firstSuccessor.IP, "Node.GetSuccessorList", &EmptyArgs{}, &listReply)
	if err != nil {
		// if we fail to get successor list, just use the first successor repeatedly
		fmt.Printf("Warning: Could not get Successor list from %s: %v\n", firstSuccessor.IP, err)
		listReply.Successors = []*NodeInfo{firstSuccessor}
	}

	// step 3: build my successor list - [firstSuccessor, firstSuccessors successors...]
	n.mu.Lock()
	n.Successor = make([]*NodeInfo, 0, n.SuccessorCount)
	n.Successor = append(n.Successor, firstSuccessor)

	// add successors from listReply until up to r total
	for i := 0; i < len(listReply.Successors) && len(n.Successor) < n.SuccessorCount; i++ {
		// avoid add ourselve to own successor list
		if listReply.Successors[i].IP != n.IP {
			n.Successor = append(n.Successor, listReply.Successors[i])
		}
	}

	// initialize finger table with first successor
	for i := 1; i <= KeySize; i++ {
		n.FingerTable[i] = firstSuccessor
	}
	n.mu.Unlock()

	fmt.Printf("Joined Chord Ring by Node: %s\n", bootstrapNode)
	fmt.Printf("My Successor's Node IP is: %s\n", firstSuccessor.IP)
	fmt.Printf("Successor List Size: %d\n", len(n.Successor))

	return nil
}

// findSuccessorIterative - local helper func to find successor iterative
func (n *Node) findSuccessorIterative(id *big.Int) (*NodeInfo, error) {
	current := &NodeInfo{ID: n.ID, IP: n.IP}

	// keep asking nodes until we find the right successor
	for {
		// ask current node: who is successor of this ID?
		var reply FindSuccessorReply
		err := CallNode(current.IP, "Node.FindSuccessor", &FindSuccessorArgs{ID: id}, &reply)
		if err != nil {
			return nil, fmt.Errorf("Failed to contact Node %s: %v", current.IP, err)
		}

		// if the reply is between current node and reply.successor
		if InBetween(id, current.ID, reply.Node.ID, true) {
			return reply.Node, nil
		}

		// if reply.Node is same as current Node (only 1 node in ring)
		if reply.Node.IP == current.IP {
			return reply.Node, nil
		}

		// else, move to the next node and retry
		current = reply.Node
	}
}

// find which node is responsible for a given key
func (n *Node) Lookup(key string) (*NodeInfo, error) {
	// hash the key to get ID
	id := Hash(key)

	fmt.Printf("Lookup key '%s' (ID: %s)\n", key, IDToString(id))

	// use iterative search to find successor
	successor, err := n.findSuccessorIterative(id)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Key '%s' is stored at Node: %s (ID: %s)\n", key, successor.IP, IDToString(successor.ID))

	return successor, nil
}

// RPC method to get this nodes predecessor
func (n *Node) GetPredecessor(args *EmptyArgs, reply *GetPredecessorReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	reply.Predecessor = n.Predecessor
	return nil
}

// RPC method to get this nodes successor list
func (n *Node) GetSuccessorList(args *EmptyArgs, reply *GetSuccessorListReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	reply.Successors = n.Successor
	return nil
}

// RPC method - new node calls Notify() on existing node to check if its predecessor
// ring example:
// Node A (ID:100) -> Node C (ID:500)
// New Node B (ID:300) joins
// Node B calls Notify() on Node C to check if its predecessor
//
// Node C then checks if it has no a predecessor (It does, Node A)
// or is Node B between my predecessor Node A and me Node C? (Yes, ID: 100 < 300 < 500)
// Result: Node B is Node C new predecessor: Node C.Predecessor = Node B
func (n *Node) Notify(args *NotifyArgs, reply *EmptyReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	candidate := args.Node // new node

	// Dont accept ourselves as predecessor
	if candidate.IP == n.IP {
		return nil
	}

	// if existing node have no predecessor, or new node is between my existing predecessor and I (existing node) = accept
	if n.Predecessor == nil || InBetween(candidate.ID, n.Predecessor.ID, n.ID, true) {
		n.Predecessor = candidate // Node C sets New Node B as its predecessor
		fmt.Printf("Notify: Updated Predecessor Node IP to: %s\n", candidate.IP)
	}

	return nil
}

// stabilize - periodically verify and update successor and predecessor
func (n *Node) Stabilize() {
	// step 1: read current state (read lock)
	n.mu.RLock()
	if len(n.Successor) == 0 {
		n.mu.RUnlock()
		return
	}

	successorIP := n.Successor[0].IP
	myID := n.ID
	myIP := n.IP
	n.mu.RUnlock()

	// step 2: make rpc calls (without holding lock to avoid deadlock)
	// ask our successor: who is your predecessor?
	var predReply GetPredecessorReply
	err := CallNode(successorIP, "Node.GetPredecessor", &EmptyArgs{}, &predReply)
	if err != nil {
		fmt.Printf("Stabilize: Failed to call Successor Node %s: %v\n", successorIP, err)

		// successor is dead - try to use next successor in list
		n.mu.Lock()
		if len(n.Successor) > 1 {
			// promote second successor to first
			fmt.Printf("Stabilize: Promoting Successor[1] (%s) to Successor[0]\n", n.Successor[1].IP)
			n.Successor = n.Successor[1:]
		} else {
			// no backup successor - revert to single-node ring
			self := &NodeInfo{ID: myID, IP: myIP}
			n.Successor = make([]*NodeInfo, n.SuccessorCount)
			for i := 0; i < n.SuccessorCount; i++ {
				n.Successor[i] = self
			}
			fmt.Printf("Stabilize: No Backup Successor - Reverting to single-node ring\n")
		}

		n.mu.Unlock()
		return
	}

	// step 3: update state based on reply with lock
	// if successor has a predecessor, and its between us and successor, it should be our new successor
	n.mu.Lock()
	if len(n.Successor) == 0 || n.Successor[0].IP != successorIP {
		n.mu.Unlock()
		return
	}

	replyFromPredecessor := predReply.Predecessor

	if replyFromPredecessor != nil && replyFromPredecessor.IP != myIP {
		// special case: if we point to ourselves, accept any predecessor
		if successorIP == myIP {
			n.Successor[0] = replyFromPredecessor
			fmt.Printf("Stabilize: Updated Successor to %s (was pointing to self)\n", replyFromPredecessor.IP)
		} else if InBetween(replyFromPredecessor.ID, myID, n.Successor[0].ID, true) {
			n.Successor[0] = replyFromPredecessor
			fmt.Printf("Stabilize: Updated Successor to %s\n", replyFromPredecessor.IP)
		}
	}

	// read successor again in case it was updated
	successorIP = n.Successor[0].IP
	n.mu.Unlock()

	// step 4: reconcile successor list with our successors list
	var listReply GetSuccessorListReply

	err = CallNode(successorIP, "Node.GetSuccessorList", &EmptyArgs{}, &listReply)
	if err == nil && len(listReply.Successors) > 0 {
		// build new successor list
		n.mu.Lock()
		newList := make([]*NodeInfo, 0, n.SuccessorCount)
		newList = append(newList, n.Successor[0]) // first successor

		// add from successors list up to r total
		for i := 0; i < len(listReply.Successors) && len(newList) < n.SuccessorCount; i++ {
			// avoid add ourselve to own successor list
			if listReply.Successors[i].IP != n.IP {
				newList = append(newList, listReply.Successors[i])
			}
		}

		n.Successor = newList
		n.mu.Unlock()
	}

	// step 5: notify our successor that we might be its predecessor (without lock)
	err = CallNode(successorIP, "Node.Notify", &NotifyArgs{Node: &NodeInfo{ID: myID, IP: myIP}}, &EmptyReply{})
	if err != nil {
		fmt.Printf("Stabilize: Failed to notify Successor %s\n", successorIP)
	}
}

// check predecessor if its still alive
func (n *Node) CheckPredecessor() {
	// read current predecessor
	n.mu.RLock()
	currentPredecessor := n.Predecessor
	n.mu.RUnlock()

	if currentPredecessor == nil {
		return
	}

	// ping predecessor without lock to avoid deadlock
	var reply PingReply
	err := CallNode(currentPredecessor.IP, "Node.Ping", &EmptyArgs{}, &reply)
	if err != nil {
		n.mu.Lock()
		n.Predecessor = nil
		n.mu.Unlock()
		fmt.Printf("CheckPredecessor: Predecessor %s Failed, removed\n", currentPredecessor.IP)
	}
}

// FixFingers - periodically update one entry in the finger table
func (n *Node) FixFingers(next int) int {
	// calculate target ID for this finger entry: (n + 2^(next-1)) mod 2^m
	targetID := Jump(n.ID, next)

	// find successor that target ID
	successor, err := n.findSuccessorIterative(targetID)
	if err != nil {
		fmt.Printf("FixFingers: Failed to find successor for finger[%d]: %v\n", next, err)
		// dont update this finger, try again next time
	} else {
		// update the finger table entry
		n.mu.Lock()
		n.FingerTable[next] = successor
		n.mu.Unlock()
	}

	// move to next finger (circular: 1 -> 2 -> ... -> 160 -> 1 -> 2)
	next++
	if next > KeySize {
		next = 1
	}

	return next
}

// interactive commandloop for user input
func (n *Node) CommandLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Available Commands:")
	fmt.Println(">	Lookup <key>")
	fmt.Println(">	PrintState")
	fmt.Println(">	Help")
	fmt.Println(">	Exit")
	fmt.Println()

	for {
		fmt.Print("> ")

		// read input from user
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		parts := strings.Fields(input) // split input on whitespaces

		if len(parts) == 0 {
			continue
		} // reset loop if input empty

		cmd := strings.ToLower(parts[0]) // lowercase input word

		switch cmd {
		case "lookup":
			// handle lookup cmd
			if len(parts) < 2 {
				fmt.Println("Correct Usage: Lookup <key>")
				continue
			}

			key := parts[1]
			_, err := n.Lookup(key)
			if err != nil {
				fmt.Printf("Lookup Failed: %v\n", err)
			}

		case "printstate":
			// handle printstate cmd
			n.PrintState()

		case "exit":
			// handle exit cmd
			fmt.Println("Exiting...")
			os.Exit(0)

		case "help":
			// print all cmds
			fmt.Println("Available Commands:")
			fmt.Println("> Lookup <key>")
			fmt.Println("> PrintState")
			fmt.Println("> Help")
			fmt.Println("> Exit")
			fmt.Println()

		default:
			// handle unknown cmd
			fmt.Printf("Unknown Command: %s\n", cmd)
		}
	}
}
