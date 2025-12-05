// RPC - Remote Procedure Call (Args and Reply structs)

package main

import (
	"math/big"
)

// RPC argument/reply structs will go here later
// (e.g., FindSuccessorArgs, NotifyArgs, etc.)

// Empty args (used when no input needed)
type EmptyArgs struct{}

// Empty reply (used when only error/success matters)
type EmptyReply struct{}

// Ping reply
type PingReply struct {
	NodeID string
	NodeIP string
}

// FindSuccessor args and reply
type FindSuccessorArgs struct {
	ID *big.Int
}

type FindSuccessorReply struct {
	Node *NodeInfo // the successor node
}
