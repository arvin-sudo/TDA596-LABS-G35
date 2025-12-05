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
	ID *big.Int
	IP string // IP:PORT Address
	Successor *NodeInfo // next node in ring
}

// NodeInfo = information about a remote node
type NodeInfo struct{
	ID *big.Int
	IP string // IP:PORT Address of remote Node
}

// create new node
func NewNode(ip string, port int) *Node {
	ipAddress := fmt.Sprintf("%s:%d", ip, port)

	node := &Node{
		ID: Hash(ipAddress),
		IP: ipAddress,
		Successor: nil,
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