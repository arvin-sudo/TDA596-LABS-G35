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
