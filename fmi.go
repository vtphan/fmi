// Copyright 2013 Vinhthuy Phan
// Package implements uncompressed FM index
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
	"bufio"
	"runtime"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

type Index struct{
	SA []int 						// suffix array
	C map[byte]int  				// count table
	OCC map[byte][]int 			// occurence table
	END_POS int 					// position of "$" in the text
	SYMBOLS []int  				// sorted symbols
	EP map[byte]int 				// ending row/position of each symbol

	// un-exported variables
	bwt []byte
	freq map[byte]int  // Frequency of each symbol
}
//
//-----------------------------------------------------------------------------
type BySuffix []int

func (s BySuffix) Len() int { return len(s) }
func (s BySuffix) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySuffix) Less(i, j int) bool { return (bytes.Compare(SEQ[s[i]:], SEQ[s[j]:]) == -1) }

//-----------------------------------------------------------------------------
// BWT is saved into a separate file
func (I *Index) BuildSA_BWT(file string) {
	I.freq = make(map[byte]int)
	I.SA = make([]int, len(SEQ))
	for i := 0; i < len(SEQ); i++ {
		I.SA[i] = i
		I.freq[SEQ[i]]++
	}
	sort.Sort(BySuffix(I.SA))

	I.bwt = make([]byte, len(SEQ))
	for i := 0; i < len(I.SA); i++ {
		I.bwt[i] = SEQ[(len(SEQ)+I.SA[i]-1)%len(SEQ)]
		if I.bwt[i] == '$' {
			I.END_POS = i
		}
	}
	ioutil.WriteFile(file, I.bwt, 0644)
	fmt.Println("Save I.bwt to", file)
}

//-----------------------------------------------------------------------------
func (I *Index) BuildIndex() {
	I.C = make(map[byte]int)
	I.OCC = make(map[byte][]int)
	I.EP = make(map[byte]int)

	for c := range I.freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]int, len(SEQ))
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	for i := 1; i < len(I.SYMBOLS); i++ {
		curr_c, prev_c := byte(I.SYMBOLS[i]), byte(I.SYMBOLS[i-1])
		I.C[curr_c] = I.C[prev_c] + I.freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.freq[curr_c] - 1
	}

	for i := 0; i < len(I.bwt); i++ {
		I.OCC[I.bwt[i]][i] = 1
		if i > 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][i] += I.OCC[symbol][i-1]
			}
		}
	}
	I.SYMBOLS = I.SYMBOLS[1:]
	delete(I.OCC, '$')
	delete(I.C, '$')
}

//-----------------------------------------------------------------------------
func (I *Index) Save(file string) {
	// Encode the struct Index y using Gob, then store it into "CompressIndex.dat"
	var fout bytes.Buffer
	enc := gob.NewEncoder(&fout)
	err := enc.Encode(I)
	if err != nil {
	  log.Fatal("Save index; encode error:", err)
	}
	ioutil.WriteFile(file, fout.Bytes(), 0600)
	fmt.Println("Save index to", file)
}

//-----------------------------------------------------------------------------
func (I *Index) Search(pattern []byte, result chan []int) {
	var sp, ep, offset int
	var ok bool

	c := pattern[len(pattern) - 1]
	sp, ok = I.C[c]
	if ! ok {
		result <- make([]int, 0)
		return
	}
	ep = I.EP[c]
	// if Debug { fmt.Println("pattern: ", string(pattern), "\n\t", string(c), sp, ep) }
	for i:= len(pattern)-2; sp <= ep && i >= 0; i-- {
  		c = pattern[i]
  		offset, ok = I.C[c]
  		if ok {
			sp = offset + I.OCC[c][sp - 1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			result <- make([]int, 0)
			return
		}
  		// if Debug { fmt.Println("\t", string(c), sp, ep) }
	}
	res := make([]int, ep-sp+1)
	for i:=sp; i<=ep; i++ {
		res[i-sp] = I.SA[i]
	}
 	result <- res
}

//-----------------------------------------------------------------------------
// return a length-l-substring of the text ending at position SA[r]-1
// terminate if reaches beyond the first index.
//-----------------------------------------------------------------------------
func (I *Index) r_substr(r int, l int) []byte {
	var s = make([]byte, l)
	var i int
	for i = l - 1; (i >= 0) && (r != I.END_POS); i-- {
		s[i] = I.bwt[r]
		r = (I.C[I.bwt[r]] + I.OCC[I.bwt[r]][r]) - 1 // substract 1 because index starts from 0
	}
	if i < 0 {
		return s
	}
	return s[i+1:]
}

//-----------------------------------------------------------------------------
func (I *Index) show() {
	fmt.Printf(" %8s  OCC\n", "C")
	for i := 0; i < len(I.SYMBOLS); i++ {
		c := byte(I.SYMBOLS[i])
		fmt.Printf("%c%8d  %d\n", c, I.C[c], I.OCC[c])
	}
	fmt.Println(I.SYMBOLS)
}


//-----------------------------------------------------------------------------
// Build FM index given the file storing the text.
// Usage:	idx := Build(text_file)
func Build (file string) *Index {
	I := new(Index)

	byte_array, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	SEQ = append(byte_array, byte('$'))
	I.BuildSA_BWT(file+".bwt")
	I.BuildIndex()
	return I
}

//-----------------------------------------------------------------------------
// Load FM index
// Usage:  idx := Load(index_file)
func Load (file string) *Index {
	I := new(Index)

	finOCC,errOCC := os.Open(file)
	decOCC := gob.NewDecoder(finOCC)
	errOCC = decOCC.Decode(I)
	if errOCC != nil {
		log.Fatal("Load FM index; decode error:", errOCC)
	}
	return I
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
	var index_file = flag.String("i", "", "index file")
	var queries_file = flag.String("q", "", "queries file")
	var workers = flag.Int("w", 1, "number of workers")
	flag.BoolVar(&Debug, "debug", false, "Turn on debug mode.")
	flag.Parse()

	if *build_file != "" {
		idx := Build(*build_file)
		idx.Save(*build_file + ".fm")
		// idx.show()
		// print_byte_array(SEQ)
		// print_byte_array(BWT)
	} else if *index_file!="" && *queries_file!="" {
		result := make(chan []int, 100000)
		runtime.GOMAXPROCS(*workers)
		idx := Load(*index_file)

		f, err := os.Open(*queries_file)
		if err != nil { panic("error opening file " + *queries_file) }
		r := bufio.NewReader(f)
		count := 0
		for {
			line, err := r.ReadBytes('\n')
			if err != nil { break }
			if len(line) > 1 {
				line = line[0:len(line)-1]
				go idx.Search(line, result)
				count++
			}
		}
		for i:=0; i<count; i++ {
			fmt.Printf("%d\t%d\n", i+1, <- result)
		}
	}

}