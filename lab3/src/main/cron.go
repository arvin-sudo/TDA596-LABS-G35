package main

import (
	"math/big"
	"time"
)

func (c *Chord) runBackground(task func(), interval time.Duration) {
	go func() {
		task()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				task()
			}
		}
	}()
}

// periodic call stabilize(--ts), fix fingers(--tff) and check predecessor(--tcp)
func (c *Chord) stabilize() {
	//fmt.Println("Stabilizing period...")
	//x := c.Successors[0].Predecessor
	//spew.Dump(x)
	//spew.Dump(c)
	//spew.Dump(c.Successor)
	GetPredecessorRep := &GetPredecessorReply{}
	var x *Chord = nil
	ok := call(c.Successors[0].IpAddr, "Chord.GetPredecessor", &GetPredecessorArgs{}, GetPredecessorRep)
	if ok {
		if GetPredecessorRep.Predecessor != nil {
			x = &Chord{
				Id:     GetPredecessorRep.Predecessor.Id,
				IpAddr: GetPredecessorRep.Predecessor.IpAddr,
			}
		}
	}
	if x != nil && InBetween(x.Id, c.Id, c.Successors[0].Id) {
		c.Successors[0] = x
	}
	call(c.Successors[0].IpAddr, "Chord.Notify", &NotifyArgs{ChordDTO: &ChordDTO{Id: c.Id, IpAddr: c.IpAddr}}, &NotifyReply{})

	// update successor list afterwards
	getSuccessorListReply := &GetSuccessorListReply{}
	ok = call(c.Successors[0].IpAddr, "Chord.GetSuccessorList", &GetSuccessorListArgs{}, getSuccessorListReply)
	if ok {
		for i := 1; i < len(c.Successors); i++ {
			if i-1 < len(getSuccessorListReply.Successors) {
				dto := getSuccessorListReply.Successors[i-1]
				c.Successors[i] = &Chord{
					Id:     dto.Id,
					IpAddr: dto.IpAddr,
				}

			}
		}
	}
}

func (c *Chord) fixFingerTable() {
	//fmt.Println("fixFingerTable period")
	next = next + 1
	if next >= fingerTableLen {
		fixFingerTableDone = true
		next = 1
	}

	id := new(big.Int)
	base := big.NewInt(2)
	exponent := big.NewInt(int64(next - 1))
	id.Exp(base, exponent, nil)
	id.Add(id, c.Id)
	chordDTO := &ChordDTO{
		Id:     id,
		IpAddr: c.IpAddr,
		//IpAddr: c.Successors[0].IpAddr,
	}

	find := c.find(id, chordDTO)
	if find != nil {
		c.FingerTable[next] = &Chord{Id: find.Id, IpAddr: find.IpAddr}
	}
}
func (c *Chord) checkPredecessor() {
	//fmt.Println("Check Predecessor period.")
	if c.Predecessor == nil {
		return
	}

	ok := call(c.Predecessor.IpAddr, "Chord.Ping", &PingArgs{}, &PingReply{})
	if !ok {
		c.mutex.Lock()
		c.Predecessor = nil
		c.mutex.Unlock()
	}
}
