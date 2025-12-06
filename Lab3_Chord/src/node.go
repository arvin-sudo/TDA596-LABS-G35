// Node-Peer

package main

import (
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
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

// print node info data
func (n *Node) PrintInfo() {
	fmt.Println(" ")
	fmt.Printf("=== NODE-INFO ===\n")
	fmt.Printf("ID: %s\n", IDToString(n.ID))
	fmt.Printf("IP-Address: %s\n", n.IP)

	if n.Successor != nil {
		fmt.Printf("Successor: %s (ID: %s)\n", n.Successor.IP, IDToString(n.Successor.ID))
	} else {
		fmt.Printf("Successor: none\n")
	}

	if n.Predecessor != nil {
		fmt.Printf("Predecessor: %s (ID: %s)\n", n.Predecessor.IP, IDToString(n.Predecessor.ID))
	} else {
		fmt.Printf("Predecessor: none\n")
	}

	fmt.Printf("=== END-INFO ===\n")
	fmt.Println(" ")
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
	fmt.Println("Created new Chord Ring")
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
	fmt.Printf("My successor's IP-Adress is: %s\n", n.Successor.IP)

	return nil
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
		fmt.Printf("Updated Predecessor to: %s\n", candidate.IP)
	}

	return nil
}

// stabilize - periodically verify and update successor and predecessor
func (n *Node) Stabilize() {
	// ask our successor: who is your predecessor?
	var reply GetPredecessorReply
	err := CallNode(n.Successor.IP, "Node.GetPredecessor", &EmptyArgs{}, &reply)
	if err != nil {
		fmt.Printf("Stabilize: Failed to call successor %s\n", n.Successor.IP)
		return
	}

	replyFromPredecessor := reply.Predecessor

	// if successor has a predecessor, and its between us and successor, it should be our new successor
	if replyFromPredecessor != nil && replyFromPredecessor.IP != n.IP && InRange(replyFromPredecessor.ID, n.ID, n.Successor.ID) {
		n.Successor = replyFromPredecessor
		fmt.Printf("Stabilize: Updated Successor to %s\n", replyFromPredecessor.IP)
	}

	// notify our successor that we might be its predecessor
	err = CallNode(n.Successor.IP, "Node.Notify", &NotifyArgs{Node: &NodeInfo{ID: n.ID, IP: n.IP}}, &EmptyReply{})
	if err != nil {
		fmt.Printf("Stabilize: Failed to notify Successor %s\n", n.Successor.IP)
	}
}
