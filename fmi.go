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
	"path"
	"runtime"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

/*
type List struct {
	data []int
}
SA = new(List)
OCC = make([]List, len(SYMBOLS))
type BWT struct {
	data []byte
}
*/
type Index struct{
	SA []int 						// suffix array
	C map[byte]int  				// count table
	OCC map[byte][]int 			// occurence table

	END_POS int 					// position of "$" in the text
	SYMBOLS []int  				// sorted symbols
	EP map[byte]int 				// ending row/position of each symbol

	LEN int
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
	I.LEN = len(SEQ)
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
func _save(thing interface{}, filename string, error_message string) {
	var out bytes.Buffer
	enc := gob.NewEncoder(&out)
	err := enc.Encode(thing)
	if err != nil {
		log.Fatal(error_message)
	}
	fmt.Println("save", filename)
	ioutil.WriteFile(filename, out.Bytes(), 0600)
}

//-----------------------------------------------------------------------------
func (I *Index) Save(file string) {
	// save(I, file+".fm", "Fail to save fm index")
	dir := file + ".index"
	os.Mkdir(dir, 0777)

	for symb := range I.OCC {
		_save(I.OCC[symb], path.Join(dir, "occ." + string(symb)),"Fail to save to occ."+string(symb))
	}
	_save(I.SA, path.Join(dir,"sa"), "Fail to save suffix array")
	_save(I.C, path.Join(dir,"c"), "Fail to save count")
	_save(I.END_POS, path.Join(dir,"end_pos"), "Fail to save end_pos")
	_save(I.SYMBOLS, path.Join(dir,"symbols"), "Fail to save symbols")
	_save(I.EP, path.Join(dir,"ep"), "Fail to save ep")
	_save(I.LEN, path.Join(dir,"len"), "Fail to save len")
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
func _load(thing interface{}, filename string) {
	fin,err := os.Open(filename)
	decOCC := gob.NewDecoder(fin)
	err = decOCC.Decode(thing)
	if err != nil {
		log.Fatal("Load error:", filename, err)
	}
}

//-----------------------------------------------------------------------------
func _load_occ(filename string, Len int) []int {
	thing := make([]int, Len)
	fin,err := os.Open(filename)
	decOCC := gob.NewDecoder(fin)
	err = decOCC.Decode(&thing)
	if err != nil {
		log.Fatal("Error loading occ table:", filename, err)
	}
	return thing
	// fmt.Println(thing[key], key)
}

//-----------------------------------------------------------------------------
// Load FM index
// Usage:  idx := Load(index_file)
func Load (dir string) *Index {
	I := new(Index)
	_load(&I.C, path.Join(dir, "c"))
	_load(&I.SA, path.Join(dir, "sa"))
	_load(&I.END_POS, path.Join(dir, "end_pos"))
	_load(&I.SYMBOLS, path.Join(dir, "symbols"))
	_load(&I.EP, path.Join(dir, "ep"))
	_load(&I.LEN, path.Join(dir, "len"))

	I.OCC = make(map[byte][]int)
	for _,symb := range I.SYMBOLS {
		I.OCC[byte(symb)] = _load_occ(path.Join(dir, "occ."+string(symb)), I.LEN)
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
		idx.Save(*build_file)
		// idx.show()
		// print_byte_array(SEQ)
		// print_byte_array(BWT)
	} else if *index_file!="" && *queries_file!="" {
		result := make(chan []int, 100000)
		runtime.GOMAXPROCS(*workers)
		idx := Load(*index_file)

		// fmt.Print(idx)
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