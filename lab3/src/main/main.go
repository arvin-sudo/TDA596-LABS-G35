package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var maxSteps = 32
var fingerTableLen = 161 // fixed
var fixFingerTableDone = false
var successorsLen = 0 // 1-32
var next = 0

// chord. range: (predecessor_id, id]
// next node range: (id, successor_id]
// node: {ipaddr, key id,}
// data: finger table, predecessor, successor, my id, key range?
type Chord struct {
	// internal use
	FingerTable []*Chord

	Predecessor *Chord
	Successors  []*Chord

	// my node info
	Id     *big.Int // my id
	IpAddr string   // my ip address
	// range, maybe useless
	Start int64 // exclusive
	End   int64 // inclusive

	// lock for Predecessor and finger table.
	mutex sync.Mutex
}

// create chord ring
func createChordRing(myId *big.Int, myIpAddress string, successorsNumber int) *Chord {
	chord := &Chord{}
	chord.Id = myId
	chord.IpAddr = myIpAddress
	chord.Predecessor = nil
	chord.Successors = make([]*Chord, successorsNumber)
	for i := 0; i < successorsNumber; i++ {
		chord.Successors[i] = chord
	}
	chord.FingerTable = make([]*Chord, fingerTableLen)
	for i := 1; i < fingerTableLen; i++ {
		chord.FingerTable[i] = chord
	}
	//spew.Dump(chord)
	return chord
}

// join chord ring
func (c *Chord) joinChordRing(chord *Chord, successorsNumber int) {
	// initialize successor to itself firstly, then change to existing chord ring successors.
	c.Successors = make([]*Chord, successorsNumber)
	for i := 0; i < successorsNumber; i++ {
		c.Successors[i] = c
	}
	c.FingerTable = make([]*Chord, fingerTableLen)
	for i := 1; i < fingerTableLen; i++ {
		c.FingerTable[i] = c
	}

	c.Predecessor = nil
	chordDTO := &ChordDTO{
		Id:     chord.Id,
		IpAddr: chord.IpAddr,
	}
	find := chord.find(c.Id, chordDTO)
	if find != nil {
		c.Successors[0] = &Chord{
			Id:     find.Id,
			IpAddr: find.IpAddr,
		}
	}

	getSuccessorListReply := &GetSuccessorListReply{}
	ok := call(c.Successors[0].IpAddr, "Chord.GetSuccessorList", &GetSuccessorListArgs{}, getSuccessorListReply)
	if ok {
		for i := 1; i < successorsNumber; i++ {
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

// local: closest_preceding_node
func (c *Chord) closestPrecedingNode(id *big.Int) *Chord {
	if fixFingerTableDone {
		for i := len(c.FingerTable) - 1; i > 0; i-- {
			if InBetween(c.FingerTable[i].Id, c.Id, id) {
				return c.FingerTable[i]
			}
		}
	}
	return c.Successors[0]
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
func call(address string, method string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", address)

	if err != nil {
		fmt.Printf("rpc dial err: %s, method: %s\n", err, method)
		return false
	}
	err = c.Call(method, args, reply)
	if err != nil {
		fmt.Printf("rpc call err: %s, method: %s\n", err, method)
		return false
	}
	c.Close()
	return true
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
	ipAddr := *myIp + ":" + strconv.Itoa(*myPort)
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
		existingIpAddr := *chordRingIp + ":" + strconv.Itoa(*chordRingPort)
		reply := &GetIdReply{}
		ok := call(existingIpAddr, "Chord.GetId", &GetIdArgs{}, reply)
		if !ok {
			fmt.Printf("chord get id failed from: %s\n", existingIpAddr)
			return
		}
		fmt.Printf("getid: %d\n", reply.Id)
		existingChordRing := &Chord{
			Id:     reply.Id,
			IpAddr: existingIpAddr,
		}
		needToAddChordNode := &Chord{
			Id:     myId,
			IpAddr: ipAddr,
		}
		needToAddChordNode.joinChordRing(existingChordRing, *successorsNumber)
		c = needToAddChordNode
	} else {
		fmt.Printf("myid: %s\n", myId.String())
		c = createChordRing(myId, ipAddr, *successorsNumber)
	}

	c.register()
	c.runBackground(c.checkPredecessor, time.Duration(*checkPredecessor)*time.Millisecond)
	c.runBackground(c.fixFingerTable, time.Duration(*fixFingerPeriod)*time.Millisecond)
	c.runBackground(c.stabilize, time.Duration(*stabilizePeriod)*time.Millisecond)
	// monitor cmd from user
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := parts[0]
		switch cmd {
		case "Lookup":
		case "StoreFile":
		case "PrintState":
			c.Print()
		default:
			fmt.Printf("unknown command: %s\n", cmd)
		}
	}
	select {}
}

func (c *Chord) Print() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	fmt.Println("------------------ Chord Info ------------------")
	fmt.Printf("Self id: %v, ipAddr: %s \n", c.Id, c.IpAddr)
	for i, v := range c.Successors {
		fmt.Printf("Successors[%d] id: %v, ipAddr: %s\n", i, v.Id, v.IpAddr)
	}
	if c.Predecessor != nil {
		fmt.Printf("Predecessor id: %v, ipAddr: %s \n", c.Predecessor.Id, c.Predecessor.IpAddr)
	} else {
		fmt.Printf("Predecessor is  nil \n")
	}
	for i := 1; i < len(c.FingerTable); i++ {
		fmt.Printf("Fingers[%d] id: %v, ipAddr: %s\n", i, c.FingerTable[i].Id, c.FingerTable[i].IpAddr)
	}
}
