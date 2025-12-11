package main

import (
	"math/big"
)

// entity
type FindArgs struct {
	Id *big.Int
}

type ChordDTO struct {
	Id     *big.Int
	IpAddr string

	//Predecessor *ChordDTO
	//Successor   *ChordDTO
}
type FindReply struct {
	C     *ChordDTO
	Found bool
}

type GetSuccessorListArgs struct {
}
type GetSuccessorListReply struct {
	Successors []*ChordDTO
}
type NotifyArgs struct {
	ChordDTO *ChordDTO
}
type NotifyReply struct{}

type PingArgs struct{}
type PingReply struct{}
type GetIdArgs struct{}
type GetIdReply struct {
	Id *big.Int
}

// find successor of id
func (c *Chord) find(id *big.Int, start *ChordDTO) *ChordDTO {
	found, nextNode := false, start
	i := 0
	for !found && i < maxSteps {
		args := &FindArgs{Id: id}
		reply := &FindReply{}

		ok := call(nextNode.IpAddr, "Chord.FindSuccessor", &args, &reply)
		if !ok {
			i++
			continue
		}
		found = reply.Found
		nextNode = reply.C
		//spew.Dump(reply)
		i++
	}
	if found {
		return nextNode
	} else {
		//fmt.Printf("Node %d not found\n", id)
		return nil
	}
}

// -------------------------- rpc --------------------------
// rpc: find_successor
func (c *Chord) FindSuccessor(args *FindArgs, reply *FindReply) error {
	if InBetween(args.Id, c.Id, c.Successors[0].Id) {
		chordDTO := &ChordDTO{
			Id:     c.Successors[0].Id,
			IpAddr: c.Successors[0].IpAddr,
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
	if end.Cmp(start) == 0 {
		return true
	}
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

func (c *Chord) GetId(args *GetIdArgs, reply *GetIdReply) error {
	reply.Id = c.Id
	return nil
}

func (c *Chord) GetSuccessorList(args *GetSuccessorListArgs, reply *GetSuccessorListReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	reply.Successors = make([]*ChordDTO, 0)
	for i := range c.Successors {
		reply.Successors = append(reply.Successors, &ChordDTO{
			Id:     c.Successors[i].Id,
			IpAddr: c.Successors[i].IpAddr,
		})
	}
	return nil
}

type GetPredecessorArgs struct{}
type GetPredecessorReply struct {
	Predecessor *ChordDTO
}

func (c *Chord) GetPredecessor(args *GetPredecessorArgs, reply *GetPredecessorReply) error {
	if c.Predecessor != nil {
		reply.Predecessor = &ChordDTO{
			Id:     c.Predecessor.Id,
			IpAddr: c.Predecessor.IpAddr,
		}
	}
	return nil
}
func (c *Chord) Notify(args *NotifyArgs, reply *NotifyReply) error {
	dto := args.ChordDTO
	if c.Predecessor == nil || (InBetween(dto.Id, c.Predecessor.Id, c.Id)) {
		c.mutex.Lock()
		c.Predecessor = &Chord{Id: dto.Id, IpAddr: dto.IpAddr}
		c.mutex.Unlock()
	}
	return nil
}

// -------------------------- end rpc --------------------------
