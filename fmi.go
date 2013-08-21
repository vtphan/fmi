// Copyright 2013 Vinhthuy Phan
// Package implements suffix array, uncompressed FM index
//
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"encoding/gob"
	"os"
	"sort"
	"flag"
	"log"
)

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte
var SA []int
var BWT []byte
var Freq map[byte]int  // Frequency of each symbol

type FMindex struct{
	C map[byte]int  // count table
	OCC map[byte][]int // occurence table
	END_POS int // position of "$" in the text
	SYMBOLS []int  // sorted symbols
	EP map[byte]int // ending row/position of each symbol
}
//
//-----------------------------------------------------------------------------
type BySuffix []int

func (s BySuffix) Len() int { return len(s) }
func (s BySuffix) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySuffix) Less(i, j int) bool { return (bytes.Compare(SEQ[s[i]:], SEQ[s[j]:]) == -1) }

//-----------------------------------------------------------------------------
func build_bwt(file string) int {
	var end_pos int
	Freq = make(map[byte]int)
	SA = make([]int, len(SEQ))
	for i := 0; i < len(SEQ); i++ {
		SA[i] = i
		Freq[SEQ[i]]++
	}
	sort.Sort(BySuffix(SA))

	BWT = make([]byte, len(SEQ))
	for i := 0; i < len(SA); i++ {
		BWT[i] = SEQ[(len(SEQ)+SA[i]-1)%len(SEQ)]
		if BWT[i] == '$' {
			end_pos = i
		}
	}
	ioutil.WriteFile(file, BWT, 0644)
	fmt.Println("Save BWT to", file)
	return end_pos
}

//-----------------------------------------------------------------------------
func (I *FMindex) BuildIndex() {
	I.C = make(map[byte]int)
	I.OCC = make(map[byte][]int)
	I.EP = make(map[byte]int)

	for c := range Freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]int, len(SEQ))
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	for i := 1; i < len(I.SYMBOLS); i++ {
		curr_c, prev_c := byte(I.SYMBOLS[i]), byte(I.SYMBOLS[i-1])
		I.C[curr_c] = I.C[prev_c] + Freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + Freq[curr_c] - 1
	}

	for i := 0; i < len(BWT); i++ {
		I.OCC[BWT[i]][i] = 1
		if i > 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][i] += I.OCC[symbol][i-1]
			}
		}
	}
	delete(I.OCC, '$')
}

//-----------------------------------------------------------------------------

func (I *FMindex) Build (file string) {
	byte_array, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	SEQ = byte_array
	I.END_POS = build_bwt(file+".bwt")
	I.BuildIndex()

	// Encode the struct FMindex y using Gob, then store it into "CompressFMindex.dat"
	var fout bytes.Buffer
	enc := gob.NewEncoder(&fout)
	err = enc.Encode(I)
	if err != nil {
	  log.Fatal("Build FM index; encode error:", err)
	}
	ioutil.WriteFile(file+".fm", fout.Bytes(), 0600)
	fmt.Println("Save index to", file+".fm")
}

//-----------------------------------------------------------------------------
func (I *FMindex) Load (file string) {
	finOCC,errOCC := os.Open(file)
	decOCC := gob.NewDecoder(finOCC)
	errOCC = decOCC.Decode(I)
	if errOCC != nil {
		log.Fatal("Load FM index; decode error:", errOCC)
	}
}
//-----------------------------------------------------------------------------
func (I *FMindex) Search(pattern []byte) {
	p := len(pattern)
	i := p - 1
	c := pattern[p - 1]
	sp := I.C[byte(c)]
	ep := I.EP[byte(c)]
	fmt.Println("pattern: ", string(pattern))
	fmt.Println("\t", string(c), sp, ep)
	for i > 0 && sp <= ep {
	  	c = pattern[i - 1]
	  	if c == '$' {
	      if sp - 2 < I.END_POS {
	          I.OCC[byte(c)][sp - 1 - 1] = 0
	      } else {
	          I.OCC[byte(c)][sp - 1 - 1] = 1
	      }
	      if ep - 1 < I.END_POS {
	          I.OCC[byte(c)][ep - 1] = 0
	      } else {
	          I.OCC[byte(c)][ep - 1] = 1
	      }
	  }
	  sp = I.C[byte(c)] + I.OCC[byte(c)][sp - 1]
	  ep = I.C[byte(c)] + I.OCC[byte(c)][ep] - 1
	  i--
	  fmt.Println("\t", string(c), sp, ep)
	}
	if ep < sp || (ep==0 && sp==0){
	  fmt.Print("0 occurence\n")
	} else {
	  fmt.Print(ep - sp + 1, " occurrrences\n")
	}
}

//-----------------------------------------------------------------------------
// return a length-l-substring of the text ending at position SA[r]-1
// terminate if reaches beyond the first index.
//-----------------------------------------------------------------------------
func (I *FMindex) r_substr(r int, l int) []byte {
	var s = make([]byte, l)
	var i int
	for i = l - 1; (i >= 0) && (r != I.END_POS); i-- {
		s[i] = BWT[r]
		r = (I.C[BWT[r]] + I.OCC[BWT[r]][r]) - 1 // substract 1 because index starts from 0
	}
	if i < 0 {
		return s
	}
	return s[i+1:]
}

//-----------------------------------------------------------------------------
func (I *FMindex) show() {
	fmt.Printf(" %8s  OCC\n", "C")
	for i := 0; i < len(I.SYMBOLS); i++ {
		c := byte(I.SYMBOLS[i])
		fmt.Printf("%c%8d  %d\n", c, I.C[c], I.OCC[c])
	}
}

//-----------------------------------------------------------------------------
func print_byte_array(a []byte) {
	for i := 0; i < len(a); i++ {
		fmt.Printf("%c", a[i])
	}
	fmt.Println()
}

//-----------------------------------------------------------------------------
func main() {
	var build_file = flag.String("build", "", "Specify a file, from which to build FM index.")
	var load_file = flag.String("load", "", "Specify a file (saved FM index), from which to load.")
	flag.Parse()

	if *build_file != "" {
		var idx FMindex
		idx.Build(*build_file)
		idx.show()
		print_byte_array(SEQ)
		print_byte_array(BWT)
		fmt.Println(SA)
	} else if *load_file != "" {
		var idx FMindex
		idx.Load(*load_file)

		// Search the substring pattern using FM index
		// to do: use strings.count to automate tests
		pattern := []string{"q", "ad", "a", "b", "c", "r", "ab", "abra", "abr", "ra", "dabra"}
		for i:=0; i<len(pattern); i++ {
		  fmt.Print(pattern[i],":\t")
		  idx.Search([]byte(pattern[i]))
		}
	}

}