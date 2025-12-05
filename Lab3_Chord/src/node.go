// Node-Peer

package main

import (
	"fmt"
	"math/big"
)

type Node struct {
	ID      *big.Int
	Address string // IP:PORT
}

// create new node
func NewNode(ip string, port int) *Node {
	address := fmt.Sprintf("%s:%d", ip, port)

	node := &Node{
		ID:      Hash(address),
		Address: address,
	}

	return node
}

// print node info data
func (n *Node) PrintInfo() {
	fmt.Printf("== Node Info ==\n")
	fmt.Printf("ID: %s\n", IDToString(n.ID))
	fmt.Printf("IP-Address: %s\n", n.Address)
	fmt.Printf("== END ==")
}
