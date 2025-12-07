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
)

// Local Node
type Node struct {
	ID          *big.Int
	IP          string    // IP:PORT Address
	Successor   *NodeInfo // next node in ring
	Predecessor *NodeInfo // previous node in ring
}

// NodeInfo = information about a remote node
type NodeInfo struct {
	ID *big.Int
	IP string // IP:PORT Address of remote Node
}

// create new node
func NewNode(ip string, port int) *Node {
	ipAddress := fmt.Sprintf("%s:%d", ip, port)

	node := &Node{
		ID:          Hash(ipAddress),
		IP:          ipAddress,
		Successor:   nil,
		Predecessor: nil,
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

	fmt.Printf("RPC server listening on %s\n", n.IP)

	// Start serving (in goroutine so it doesn't block)
	go http.Serve(listener, nil)

	return nil
}

// print complete state of this node
func (n *Node) PrintState() {
	fmt.Printf("\n===== NODE-STATE =====\n")

	// own node info
	fmt.Printf("Node ID: %s\n", IDToString(n.ID))
	fmt.Printf("Node IP: %s\n", n.IP)

	// successor info
	fmt.Println("\n----- SUCCESSOR NODE -------")
	if n.Successor != nil {
		fmt.Printf("Successor ID: %s\n", IDToString(n.Successor.ID))
		fmt.Printf("Successor IP: %s\n", n.Successor.IP)
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

	fmt.Printf("======================\n")
	fmt.Println()
}

// ping - rpc method to check if node is alive
func (n *Node) Ping(args *EmptyArgs, reply *PingReply) error {
	reply.NodeID = IDToString(n.ID)
	reply.NodeIP = n.IP
	return nil
}

// create a new chord ring (one alone node)
func (n *Node) Create() {
	// in a ring with only one node, we are our own successor
	n.Successor = &NodeInfo{
		ID: n.ID,
		IP: n.IP,
	}
	fmt.Println("NEW CHORD RING CREATED")
}

// FindSuccessor - RPC method to find the successor node of an ID
func (n *Node) FindSuccessor(args *FindSuccessorArgs, reply *FindSuccessorReply) error {
	id := args.ID

	if InRange(id, n.ID, n.Successor.ID) {
		reply.Node = n.Successor
		return nil
	}

	reply.Node = n.Successor
	return nil
}

// join an existing chord ring
func (n *Node) Join(bootstrapNode string) error {
	var reply FindSuccessorReply
	err := CallNode(bootstrapNode, "Node.FindSuccessor", &FindSuccessorArgs{ID: n.ID}, &reply)
	if err != nil {
		return fmt.Errorf("Failed to contact bootstrap node: %v", err)
	}

	n.Successor = reply.Node

	fmt.Printf("Joined Chord Ring by Node: %s\n", bootstrapNode)
	fmt.Printf("My Successor's Node IP is: %s\n", n.Successor.IP)

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
		if InRange(id, current.ID, reply.Node.ID) {
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

	// use iterativ search to find successor
	successor, err := n.findSuccessorIterative(id)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Key '%s' is stored at node %s (ID: %s)\n", key, IDToString(successor.ID), successor.IP)

	return successor, nil
}

// RPC method to get this nodes predecessor
func (n *Node) GetPredecessor(args *EmptyArgs, reply *GetPredecessorReply) error {
	reply.Predecessor = n.Predecessor
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
	candidate := args.Node // new node

	// Dont accept ourselves as predecessor
	if candidate.IP == n.IP {
		return nil
	}

	// if existing node have no predecessor, or new node is between my existing predecessor and I (existing node) = accept
	if n.Predecessor == nil || InRange(candidate.ID, n.Predecessor.ID, n.ID) {
		n.Predecessor = candidate // Node C sets New Node B as its predecessor
		fmt.Printf("Notify: Updated Predecessor Node IP to: %s\n", candidate.IP)
	}

	return nil
}

// stabilize - periodically verify and update successor and predecessor
func (n *Node) Stabilize() {
	// ask our successor: who is your predecessor?
	var reply GetPredecessorReply
	err := CallNode(n.Successor.IP, "Node.GetPredecessor", &EmptyArgs{}, &reply)
	if err != nil {
		fmt.Printf("Stabilize: Failed to call Successor Node %s\n", n.Successor.IP)
		return
	}

	replyFromPredecessor := reply.Predecessor

	// if successor has a predecessor, and its between us and successor, it should be our new successor
	if replyFromPredecessor != nil && replyFromPredecessor.IP != n.IP {
		// special case: if we point to ourselves, accept any predecessor
		if n.Successor.IP == n.IP {
			n.Successor = replyFromPredecessor
			fmt.Printf("Stabilize: Updated Successor to %s (was pointing to self)\n", replyFromPredecessor.IP)
		} else if InRange(replyFromPredecessor.ID, n.ID, n.Successor.ID) {
			n.Successor = replyFromPredecessor
			fmt.Printf("Stabilize: Updated Successor to %s\n", replyFromPredecessor.IP)
		}
	}
	// notify our successor that we might be its predecessor
	err = CallNode(n.Successor.IP, "Node.Notify", &NotifyArgs{Node: &NodeInfo{ID: n.ID, IP: n.IP}}, &EmptyReply{})
	if err != nil {
		fmt.Printf("Stabilize: Failed to notify Successor %s\n", n.Successor.IP)
	}
}

// interactive commandloop for user input
func (n *Node) CommandLoop() {
	scanner := bufio.NewScanner(os.Stdin)

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
