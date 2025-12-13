package main

import (
	"fmt"
	"math/big"
	"testing"
)

func TestInBetween(t *testing.T) {
	id := big.NewInt(4)
	start := big.NewInt(6)
	end := big.NewInt(5)

	fmt.Printf("id: %v, start: %v, end: %v, %t\n", id, start, end, InBetween(id, start, end))

}

func TestSliceArray(t *testing.T) {
	Successors := []int{1, 2, 3}

	Successors = Successors[1:]
	fmt.Printf("Successors: %v\n", Successors)
}
