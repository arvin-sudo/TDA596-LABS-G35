// Utility - Helper - Hashing

package main

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"net/rpc"
)

/*
 In Chord each node as a unique ID (160-bit number from SHA-1).
 Create this ID by hashing the nodes IP:PORT
*/

// Hash a string to a big.Int (160-bit identifier)
func Hash(key string) *big.Int {
	hash := sha1.Sum([]byte(key))

	// convert byte array to big.Int
	id := big.NewInt(0)
	id.SetBytes(hash[:])

	return id
}

// helper: convert big.Int to hex string for printing
func IDToString(id *big.Int) string {
	return fmt.Sprintf("%x", id)
}

// Call RPC method on another node to verify they communicate
func CallNode(ip string, method string, args interface{}, reply interface{}) error {
	client, err := rpc.DialHTTP("tcp", ip)
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.Call(method, args, reply)
	return err
}

// check if ID is in range (start, end] on the chord ring
// if start == end, the range is the full ring
func InRange(id, start, end *big.Int) bool {
	// Case 1: Same node (alone ring)
	// special case: if start == end, only true if ID == start
	if start.Cmp(end) == 0 {
		return id.Cmp(start) == 0
	}

	// Case 2:
	// normal case: start < end (no wraparound)
	if start.Cmp(end) < 0 {
		// ID should be: start < ID AND ID <= end
		return start.Cmp(id) < 0 && id.Cmp(end) <= 0
	}

	// Case 3:
	// wraparound case: start > end (wraps around 0)
	// ID is in range if: ID > start OR ID <= end
	return id.Cmp(start) > 0 || id.Cmp(end) <= 0
}
