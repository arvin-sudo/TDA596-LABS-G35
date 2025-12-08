package lab3

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
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

	// some ip from att or get local ip address.
	ipAddr := ""
	myId := hash(ipAddr)

	// if no --ja, --jp then create
	c := createChordRing(myId.Int64(), ipAddr)
	// else join

	c.register()

}
