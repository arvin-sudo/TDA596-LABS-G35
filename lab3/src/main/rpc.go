package main

import (
	"fmt"
	"math/big"
)

// entity
type FindArgs struct {
	Id big.Int
}

type ChordDTO struct {
	Id     big.Int
	IpAddr string

	//Predecessor *ChordDTO
	//Successor   *ChordDTO
}
type FindReply struct {
	C     *ChordDTO
	Found bool
}

type PingArgs struct{}
type PingReply struct{}
type GetIdArgs struct{}
type GetIdReply struct {
	Id big.Int
}

// find
func (c *Chord) find(id big.Int, start *ChordDTO) *ChordDTO {
	found, nextNode := false, start
	i := 0
	for !found && i < maxSteps {
		args := &FindArgs{Id: id}
		reply := &FindReply{}

		ok := call(nextNode.IpAddr, "Chord.FindSuccessor", &args, &reply)
		if !ok {
			continue
		}
		found = reply.Found
		nextNode = reply.C
		i++
	}
	if found {
		return nextNode
	} else {
		fmt.Printf("Node %d not found\n", id)
		return nil
	}
}

// -------------------------- rpc --------------------------
// rpc: find_successor
func (c *Chord) FindSuccessor(args *FindArgs, reply *FindReply) error {
	InBetween(args.Id, c.Id, c.Successor.Id)
	if args.Id > c.Id && args.Id <= c.Successor.Id {
		chordDTO := &ChordDTO{
			Id:     c.Successor.Id,
			IpAddr: c.Successor.IpAddr,
		}
		reply.C = chordDTO
		reply.Found = true
	} else {
		chordNode := c.closestPrecedingNode(args.Id)
		chordDTO := &ChordDTO{
			Id:     chordNode.Id,
			IpAddr: chordNode.IpAddr,
		}
		reply.C = chordDTO
		reply.Found = false
	}
	return nil
}

// helper functions
func InBetween(id, start, end *big.Int) bool {
	// normal case
	if end.Cmp(start) == 1 {
		return id.Cmp(start) == 1 && id.Cmp(end) <= 0
	}

	// over 0
	return id.Cmp(start) == 1 || id.Cmp(end) <= 0
}

func (c *Chord) Ping(args *PingArgs, reply *PingReply) error {
	return nil
}

func (c *Chord) GetId(arg *GetIdArgs, reply *GetIdReply) error {
	reply.Id = c.Id
	return nil
}

// -------------------------- end rpc --------------------------
