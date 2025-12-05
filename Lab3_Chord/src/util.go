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
func IDToString(id * big.Int) string {
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