/*
   Copyright 2015 Vinhthuy Phan
	Compressed FM index.
	Todo:
	- compress
*/
package fmi

import (
	"fmt"
	"sort"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

type Index struct {
	BWT []byte
	SA  []int64          // suffix array
	C   map[byte]int64   // count table
	OCC map[byte][]int64 // occurence table

	END_POS int64          // position of "$" in the text
	SYMBOLS []int          // sorted symbols
	EP      map[byte]int64 // ending row/position of each symbol

	LEN  int64
	Freq map[byte]int64 // Frequency of each symbol
}

//

//-----------------------------------------------------------------------------

func check_for_error(e error) {
	if e != nil {
		panic(e)
	}
}

//-----------------------------------------------------------------------------
// Build FM index given the file storing the text.
func New(file string) *Index {
	I := new(Index)
	ReadFasta(file)
	I.build_suffix_array()
	I.build_bwt_fmindex()
	return I
}

//-----------------------------------------------------------------------------

type Symb_OCC struct {
	Symb int
	OCC  []int64
}

//-----------------------------------------------------------------------------
// BWT is saved into a separate file
func (I *Index) build_suffix_array() {
	I.LEN = int64(len(SEQ))
	I.SA = make([]int64, I.LEN)
	SA := make([]int, I.LEN)
	ws := &WorkSpace{}
	ws.ComputeSuffixArray(SEQ, SA)
	for i := range SA {
		I.SA[i] = int64(SA[i])
	}
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
	I.Freq = make(map[byte]int64)
	I.BWT = make([]byte, I.LEN)
	var i int64
	for i = 0; i < I.LEN; i++ {
		I.Freq[SEQ[i]]++
		if I.SA[i] == 0 {
			I.BWT[i] = SEQ[I.LEN-1]
		} else {
			I.BWT[i] = SEQ[I.SA[i]-1]
		}
		if I.BWT[i] == '$' {
			I.END_POS = i // this is no longer correct due to existence of many $'s
		}
	}

	I.C = make(map[byte]int64)
	I.OCC = make(map[byte][]int64)
	for c := range I.Freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]int64, I.LEN)
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	I.EP = make(map[byte]int64)
	for j := 1; j < len(I.SYMBOLS); j++ {
		curr_c, prev_c := byte(I.SYMBOLS[j]), byte(I.SYMBOLS[j-1])
		I.C[curr_c] = I.C[prev_c] + I.Freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.Freq[curr_c] - 1
	}

	for j := 0; j < len(I.BWT); j++ {
		I.OCC[I.BWT[j]][j] = 1
		if j > 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][j] += I.OCC[symbol][j-1]
			}
		}
	}
	// I.SYMBOLS = I.SYMBOLS[1:] // Remove $, which is the first symbol
	// delete(I.OCC, '$')
	// delete(I.C, '$')
	fmt.Println("Sequence", string(SEQ))

}

//-----------------------------------------------------------------------------
func (I *Index) Check() {
	a, b, c := I.Search(SEQ[0 : len(SEQ)-1])
	fmt.Println("Search for SEQ returns", a, b, c)
}

//-----------------------------------------------------------------------------
// Returns starting, ending positions (sp, ep) and last-matched position (i)
func (I *Index) Search(pattern []byte) (int, int, int) {
	var offset int64
	var i int
	start_pos := len(pattern) - 1
	c := pattern[start_pos]
	sp, ok := I.C[c]
	if !ok {
		return 0, -1, -1
	}
	ep := I.EP[c]
	for i = int(start_pos - 1); sp <= ep && i >= 0; i-- {
		c = pattern[i]
		offset, ok = I.C[c]
		if ok {
			sp = offset + I.OCC[c][sp-1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			return 0, -1, -1
		}
	}
	return int(sp), int(ep), i + 1
}

//-----------------------------------------------------------------------------
func (I *Index) Show() {
	fmt.Printf(" %6s %6s  OCC\n", "Freq", "C")
	for i := 0; i < len(I.SYMBOLS); i++ {
		c := byte(I.SYMBOLS[i])
		fmt.Printf("%c%6d %6d  %d\n", c, I.Freq[c], I.C[c], I.OCC[c])
	}
	fmt.Printf("SA ")
	for i := 0; i < len(I.SA); i++ {
		fmt.Print(I.SA[i], " ")
	}
	fmt.Printf("\nBWT ")
	for i := 0; i < len(I.BWT); i++ {
		fmt.Print(string(I.BWT[i]))
	}
	fmt.Println()
	fmt.Println("SEQ", string(SEQ))
}

//-----------------------------------------------------------------------------
