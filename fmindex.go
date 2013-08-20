// Copyright 2013 Vinhthuy Phan
// Package implements suffix array, uncompressed FM index
//
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

//-----------------------------------------------------------------------------
// Global variables:
//		sequence (SEQ),
//		suffix array (SA),
//		BWT,
//		FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte
var SA []int
var BWT []byte
var C = make(map[byte]int)
var OCC = make(map[byte][]int)
var END_POS int // position of $ in BWT
var SYMBOLS []int

//-----------------------------------------------------------------------------
type BySuffix []int

func (s BySuffix) Len() int { return len(s) }
func (s BySuffix) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySuffix) Less(i, j int) bool { return (bytes.Compare(SEQ[s[i]:], SEQ[s[j]:]) == -1) }

//-----------------------------------------------------------------------------
func build_bwt() {
	BWT = make([]byte, len(SEQ))
	for i := 0; i < len(SA); i++ {
		BWT[i] = SEQ[(len(SEQ)+SA[i]-1)%len(SEQ)]
		if BWT[i] == '$' {
			END_POS = i
		}
	}
}

//-----------------------------------------------------------------------------
// Return a length-l-substring of the text ending at position SA[r]-1
// Terminate if reaches beyond the first index.
//-----------------------------------------------------------------------------
func r_substr(r int, l int) []byte {
	var s = make([]byte, l)
	var i int
	for i = l - 1; (i >= 0) && (r != END_POS); i-- {
		s[i] = BWT[r]
		r = (C[BWT[r]] + OCC[BWT[r]][r]) - 1 // substract 1 because index starts from 0
	}
	if i < 0 {
		return s
	}
	return s[i+1:]
}

//-----------------------------------------------------------------------------

func build_fm_index(byte_array []byte) {
	SEQ = byte_array

	var count = make(map[byte]int)

	SA = make([]int, len(SEQ))
	for i := 0; i < len(SEQ); i++ {
		SA[i] = i
		count[SEQ[i]]++
	}
	for c := range count {
		OCC[c] = make([]int, len(SEQ))
		SYMBOLS = append(SYMBOLS, int(c))
		C[c] = 0
	}
	sort.Ints(SYMBOLS)

	sort.Sort(BySuffix(SA))
	build_bwt()

	// Build FM index (OCC and C)
	for i := 1; i < len(SYMBOLS); i++ {
		curr_c, prev_c := byte(SYMBOLS[i]), byte(SYMBOLS[i-1])
		C[curr_c] = C[prev_c] + count[prev_c]
	}

	for i := 0; i < len(BWT); i++ {
		OCC[BWT[i]][i] = 1
		if i > 0 {
			for symbol := range OCC {
				OCC[symbol][i] += OCC[symbol][i-1]
			}
		}
	}
}

//-----------------------------------------------------------------------------

func print_fm_index() {
	fmt.Printf(" %8s  OCC\n", "C")
	for i := 0; i < len(SYMBOLS); i++ {
		c := byte(SYMBOLS[i])
		fmt.Printf("%c%8d  %d\n", c, C[c], OCC[c])
	}
}

func print_byte_array(a []byte) {
	for i := 0; i < len(a); i++ {
		fmt.Printf("%c", a[i])
	}
	fmt.Println()
}

//-----------------------------------------------------------------------------
func main() {
	fmt.Println(os.Args)
	if len(os.Args) != 2 {
		panic("Must provide input file")
	}

	byte_array, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	build_fm_index(byte_array)
	print_fm_index()
	print_byte_array(SEQ)
	print_byte_array(BWT)
	fmt.Println(SA)

	p, l := 9, 10
	s := r_substr(p, l)
	fmt.Printf("SA[%d] = %d, len=%d\n", p, SA[p], l)
	for i := 0; i < l; i++ {
		if SA[p]-(i+1) >= 0 {
			fmt.Printf("%c", SEQ[SA[p]-(i+1)])
		}
	}
	fmt.Printf("\n")
	for i := 0; i < len(s); i++ {
		fmt.Printf("%c", s[i])
	}
	fmt.Println()
}
