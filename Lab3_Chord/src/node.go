// Node-Peer

package main

import (
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
)

type Node struct {
	ID *big.Int
	IP string // IP:PORT Address
}

// create new node
func NewNode(ip string, port int) *Node {
	ipAddress := fmt.Sprintf("%s:%d", ip, port)

	node := &Node{
		ID: Hash(ipAddress),
		IP: ipAddress,
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
	fmt.Printf("== Node Info ==\n")
	fmt.Printf("ID: %s\n", IDToString(n.ID))
	fmt.Printf("IP-Address: %s\n", n.IP)
	fmt.Printf("== END ==\n")
}
