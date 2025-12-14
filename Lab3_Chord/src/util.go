// Utility - Helper - Hashing

package main

import (
	"bufio"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"math/big"
	"net/http"
	"net/rpc"
)

// fingerTable
const (
	KeySize = sha1.Size * 8 // 160bits
)

// Global variable to track if TLS is enabled for this process
// Set by main() and used by CallNode() to connect to remote nodes
var GlobalUseTLS bool = false

/*
 In Chord each node has a unique ID (160-bit number from SHA-1).
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

// Call RPC method on another node - auto uses TLS if GlobalUseTLS is true
func CallNode(ip string, method string, args interface{}, reply interface{}) error {
	var client *rpc.Client
	var err error

	if GlobalUseTLS {
		// TLS Mode: connect with TLS and skip certificate verification
		// (we use self-signed certs, so cant verify them against a CA)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // accept self-signed certificates
		}

		conn, err := tls.Dial("tcp", ip, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %v", err)
		}

		// manually do HTTP CONNECT handshake for RPC over TLS
		io := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		io.WriteString("CONNECT " + rpc.DefaultRPCPath + " HTTP/1.0\n\n")
		io.Flush()

		// read response
		resp, err := http.ReadResponse(io.Reader, &http.Request{Method: "CONNECT"})
		if err == nil && resp.Status == "200 "+rpc.DefaultRPCPath {
			client = rpc.NewClient(conn)
		} else {
			conn.Close()
			return fmt.Errorf("unexpected HTTP response: %v", resp.Status)
		}
	} else {
		// Plain TCP Mode: Standard HTTP RPC
		client, err = rpc.DialHTTP("tcp", ip)
		if err != nil {
			return err
		}
	}
	defer client.Close()

	err = client.Call(method, args, reply)
	return err
}

// check if ID is between (start, end) on the chord ring
// if inclusive is true, checks (start, end], otherwise (start, end)
// start == end, the range is the full ring
func InBetween(id, start, end *big.Int, inclusive bool) bool {
	// Case 1: Same node (single node or full ring), start == end
	if start.Cmp(end) == 0 {
		if inclusive {
			return id.Cmp(end) == 0
		}
		return false
	}

	// Case 2: normal case (no wraparound): start < end
	if start.Cmp(end) < 0 {
		if inclusive {
			// (start, end] ID should be: start < ID AND ID <= end
			return start.Cmp(id) < 0 && id.Cmp(end) <= 0
		} else {
			// (start, end) = start < ID AND ID < end
			return start.Cmp(id) < 0 && id.Cmp(end) < 0
		}
	}

	// Case 3: wraparound case: start > end (wrapsaround 0)
	if inclusive {
		// (start, end] wrapping = ID is in range if: ID > start OR ID <= end
		return id.Cmp(start) > 0 || id.Cmp(end) <= 0
	} else {
		// (start, end) wrapping = id > start OR id < end
		return id.Cmp(start) > 0 || id.Cmp(end) < 0
	}
}

// jump - calculate the target ID for finger table entry i
// returns: (nodeID + 2^(i-1)) mod 2^m where m = 160
func Jump(nodeID *big.Int, fingerIndex int) *big.Int {
	// calculate 2^(fingerIndex - 1)
	power := big.NewInt(int64(fingerIndex - 1))
	offset := new(big.Int).Exp(big.NewInt(2), power, nil)

	// add offset to nodeID
	target := new(big.Int).Add(nodeID, offset)

	// mod 2^160 to wrap around the ring
	hashMod := new(big.Int).Exp(big.NewInt(2), big.NewInt(KeySize), nil)
	result := new(big.Int).Mod(target, hashMod)

	return result
}
