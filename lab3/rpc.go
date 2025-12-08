package lab3

import "fmt"

// find
func (c *Chord) find(id int64, start *Chord) *Chord {
	found, nextNode := false, start
	i := 0
	for !found && i < maxSteps {
		found, nextNode = nextNode.findSuccessor(id)
		i++
	}
	if found {
		return nextNode
	} else {
		fmt.Printf("Node %d not found\n", id)
		return nil
	}
}

// -------------------------- rpc --------------------------
// rpc: find_successor
func (c *Chord) findSuccessor(id int64) (bool, *Chord) {
	if id > c.Id && id <= c.Successor.Id {
		return true, c.Successor
	} else {
		return false, c.closestPrecedingNode(id)
	}
}

func (c *Chord) ping() error {
	return nil
}

// -------------------------- end rpc --------------------------
