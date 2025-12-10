package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync"
)

var maxSteps = 32
var fingerTableLen = 161 // fixed
var successorsLen = 0    // 1-32
var next = 0

// chord. range: (predecessor_id, id]
// next node range: (id, successor_id]
// node: {ipaddr, key id,}
// data: finger table, predecessor, successor, my id, key range?
type Chord struct {
	FingerTable []*Chord
	Predecessor *Chord
	Successor   *Chord

	// my node info
	Id     int64  // my id
	IpAddr string // my ip address
	// range
	Start int64 // exclusive
	End   int64 // inclusive

	// lock for Predecessor and finger table.
	mutex sync.Mutex
}

// create chord ring
func createChordRing(myId int64, myIpAddress string) *Chord {
	chord := &Chord{}
	chord.Id = myId
	chord.IpAddr = myIpAddress
	chord.Predecessor = nil
	chord.Successor = chord
	chord.FingerTable = make([]*Chord, fingerTableLen)
	for i := 1; i < fingerTableLen; i++ {
		chord.FingerTable[i] = chord
	}

	return chord
}

// join chord ring
func (c *Chord) joinChordRing(chord *Chord) {
	c.Predecessor = nil
	c.Successor = chord.find(c.Id, chord)
}

// local: closest_preceding_node
func (c *Chord) closestPrecedingNode(id int64) *Chord {
	for i := len(c.FingerTable) - 1; i >= 0; i-- {
		if c.FingerTable[i].Id > c.Id && c.FingerTable[i].Id < id {
			return c.FingerTable[i]
		}
	}
	return c.Successor
}

// local: hash function
func hash(str string) *big.Int {
	h := sha1.New()
	h.Write([]byte(str))
	id := new(big.Int).SetBytes(h.Sum(nil))

	two := big.NewInt(2)
	exponent := big.NewInt(int64(fingerTableLen) - 1)
	modSize := new(big.Int).Exp(two, exponent, nil)

	id.Mod(id, modSize)
	return id
}

// rpc call
func call(address string, method string, args interface{}, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", address)
	defer c.Close()
	if err != nil {
		fmt.Printf("rpc dial err: %s\n", err)
		return err
	}
	err = c.Call(method, args, reply)
	if err != nil {
		fmt.Printf("rpc call err: %s\n", err)
		return err
	}
	return nil
}

// rpc register
func (c *Chord) register() {
	err := rpc.Register(c)
	if err != nil {
		fmt.Printf("rpc register err: %s\n", err)
		return
	}
	rpc.HandleHTTP()
	listen, err := net.Listen("tcp", c.IpAddr)
	if err != nil {
		fmt.Printf("rpc listen err: %s\n", err)
		return
	}
	go http.Serve(listen, nil)
}

func main() {
	var isJoin = false

	myIp := flag.String("a", "", "chord client ip address")
	myPort := flag.Int("p", -1, "chord client port number")
	chordRingIp := flag.String("ja", "", "chord ring client ip address")
	chordRingPort := flag.Int("jp", -1, "chord ring client port number")
	stabilizePeriod := flag.Int("ts", -1, "stabilize periodical time in milliseconds")
	fixFingerPeriod := flag.Int("tff", -1, "fix finger periodical time in milliseconds")
	checkPredecessor := flag.Int("tcp", -1, "check predecessor periodical time in milliseconds")
	successorsNumber := flag.Int("r", -1, "successors number to be maintained")
	id := flag.String("i", "", "id. optional")

	flag.Parse()

	if net.ParseIP(*myIp) == nil {
		fmt.Printf("chord client ip address is invalid: %s\n", *myIp)
		return
	}

	if *myPort < 0 {
		fmt.Printf("chord client port number is invalid: %d\n", *myPort)
		return
	}

	if *chordRingIp != "" {
		if net.ParseIP(*chordRingIp) == nil {
			fmt.Printf("chord ring client ip address is invalid: %s\n", *chordRingIp)
			return
		}

		isJoin = true
	}

	if *chordRingPort != -1 {
		if !isJoin {
			fmt.Printf("chord ring client ip is required.\n")
			return
		}
		if *chordRingPort < 0 {
			fmt.Printf("chord ring client port number is invalid: %d\n", *chordRingPort)
			return
		}

	}

	if *stabilizePeriod < 1 || *stabilizePeriod > 60000 {
		fmt.Printf("stabilize periodical time must be between 1 and 60000: %d\n", *stabilizePeriod)
		return
	}

	if *fixFingerPeriod < 1 || *fixFingerPeriod > 60000 {
		fmt.Printf("fix finger periodical time must be between 1 and 60000: %d\n", *fixFingerPeriod)
		return
	}

	if *checkPredecessor < 1 || *checkPredecessor > 60000 {
		fmt.Printf("check predecessor periodical time must be between 1 and 60000: %d\n", *checkPredecessor)
		return
	}

	if *successorsNumber < 1 || *successorsNumber > 32 {
		fmt.Printf("successors number must be between 1 and 32: %d\n", *successorsNumber)
		return
	}

	// some ip from att or get local ip address.
	ipAddr := *myIp + strconv.Itoa(*myPort)
	var myId *big.Int
	if *id == "" {
		myId = hash(ipAddr)
	} else {
		idnumber, err := strconv.Atoi(*id)
		if err != nil {
			fmt.Printf("chord id number is invalid: %d\n", idnumber)
			return
		}
		myId = big.NewInt(int64(idnumber))
	}

	// if has --ja, --jp then join otherwise create a chord ring.
	var c *Chord
	if isJoin {
		// todo
		existingChordRing := &Chord{}
		needToAddChordNode := &Chord{}
		needToAddChordNode.joinChordRing(existingChordRing)
		c = needToAddChordNode
	} else {
		c = createChordRing(myId.Int64(), ipAddr)
	}

	c.register()

}
