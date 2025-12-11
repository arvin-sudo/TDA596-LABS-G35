package main

import (
	"math"
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
	x := c.Successor.Predecessor
	if x.Id > c.Id && x.Id < c.Successor.Id {
		c.Successor = x
	}
	c.Successor.notify(c)
}

func (c *Chord) notify(chord *Chord) {
	if c.Predecessor == nil || (chord.Id > chord.Predecessor.Id && chord.Id < c.Id) {
		c.mutex.Lock()
		c.Predecessor = chord
		c.mutex.Unlock()
	}
}

func (c *Chord) fixFingerTable() {
	next = next + 1
	if next >= fingerTableLen {
		next = 1
	}

	chordDTO := &ChordDTO{
		Id:     c.Id + int64(math.Pow(2, float64(next-1))),
		IpAddr: c.IpAddr,
	}

	find := c.find(c.Id+int64(math.Pow(2, float64(next-1))), chordDTO)
	if find != nil {
		c.FingerTable[next] = &Chord{Id: find.Id, IpAddr: find.IpAddr}
	}
}
func (c *Chord) checkPredecessor() {
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
