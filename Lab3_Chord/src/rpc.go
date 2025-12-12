// RPC - Remote Procedure Call (Args and Reply structs)

package main

import (
	"math/big"
)

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

// Predecessor - ask a node who its predecessor is
type GetPredecessorReply struct {
	Predecessor *NodeInfo
}

// Notify args - tell a node it might be our predecessor
type NotifyArgs struct {
	Node *NodeInfo
}

// PUT - store data - store a key-value pair
type PutArgs struct {
	Key   string // data id
	Value string // data content
}

type PutReply struct{}

// GET - fetch data - retrieve a value for a key
type GetArgs struct {
	Key string // data id
}

type GetReply struct {
	Value string // data content
	Found bool   // data exists?
}

// GetSuccessorList - ask a node for its successor list
type GetSuccessorListReply struct {
	Successors []*NodeInfo
}
