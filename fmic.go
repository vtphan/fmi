/*
   Copyright 2015 Vinhthuy Phan
	Compressed FM index.
	Todo:
	 - identify genomic region containing the match of a search.
*/
package fmi

import (
	"fmt"
	"sort"
	"math"
	"bufio"
	"encoding/binary"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------

type IndexC struct {
	BWT []byte
	SA  []int64          // suffix array
	C   map[byte]int64   // count table
	OCC map[byte][]int64 // occurence table

	END_POS int64          // position of "$" in the text
	SYMBOLS []int          // sorted symbols
	EP      map[byte]int64 // ending row/position of each symbol

	LEN  int64
	OCC_SIZE int64
	Freq map[byte]int64 // Frequency of each symbol
	M int             // Compression ratio
	input_file string
}

//-----------------------------------------------------------------------------
// Build FM index given the file storing the text.
func CompressedIndex(file string, compression_ratio int) *IndexC {
	I := new(IndexC)
	I.M = compression_ratio
	ReadFasta(file)
	I.build_suffix_array()
	I.build_bwt_fmindex()
	I.input_file = file
	return I
}

//-----------------------------------------------------------------------------
// BWT is saved into a separate file
func (I *IndexC) build_suffix_array() {
	I.LEN = int64(len(SEQ))
	I.OCC_SIZE = int64(math.Ceil(float64(I.LEN/int64(I.M))))+1
	I.SA = make([]int64, I.LEN)
	SA := make([]int, I.LEN)
	ws := &WorkSpace{}
	ws.ComputeSuffixArray(SEQ, SA)
	for i := range SA {
		I.SA[i] = int64(SA[i])
	}
}

//-----------------------------------------------------------------------------
func (I *IndexC) build_bwt_fmindex() {
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
			I.END_POS = i
		}
	}

	I.C = make(map[byte]int64)
	I.OCC = make(map[byte][]int64)
	for c := range I.Freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]int64, I.OCC_SIZE)
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	I.EP = make(map[byte]int64)
	count := make(map[byte]int64)

	for j := 1; j < len(I.SYMBOLS); j++ {
		curr_c, prev_c := byte(I.SYMBOLS[j]), byte(I.SYMBOLS[j-1])
		I.C[curr_c] = I.C[prev_c] + I.Freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.Freq[curr_c] - 1
		count[curr_c] = 0
	}

	for j := 0; j < len(I.BWT); j++ {
		count[I.BWT[j]] += 1
		if j % I.M == 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][int(j/I.M)] = count[symbol]
			}
		}
	}
}

//-----------------------------------------------------------------------------
func (I *IndexC) Occurence(c byte, pos int64) int64 {
	i := int64(pos/int64(I.M))
	count := I.OCC[c][i]
	for j:=i*int64(I.M)+1; j<=pos; j++ {
		if I.BWT[j]==c {
			count += 1
		}
	}
	return count
}

//-----------------------------------------------------------------------------
// Returns starting, ending positions (sp, ep) and last-matched position (i)
func (I *IndexC) Search(pattern []byte) (int, int, int) {
	var offset int64
	var i int
	start_pos := len(pattern) - 1
	c := pattern[start_pos]
	sp := I.C[c]
	ep := I.EP[c]
	for i = int(start_pos - 1); sp <= ep && i >= 0; i-- {
		c = pattern[i]
		offset = I.C[c]
		sp = offset + I.Occurence(c,sp-1)
		ep = offset + I.Occurence(c,ep) - 1
	}
	return int(sp), int(ep), i + 1
}

//-----------------------------------------------------------------------------
func (I *IndexC) Show() {
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
func (I *IndexC) Check() {
	for i:=0; i<len(I.SYMBOLS); i++ {
		c := byte(I.SYMBOLS[i])
		fmt.Printf("%c%6d %6d  [", c, I.Freq[c], I.C[c])
		for j:=0; j<int(I.LEN); j++ {
			fmt.Printf("%d ", I.Occurence(c,int64(j)))
		}
		fmt.Printf("]\n")
	}
	a, b, c := I.Search(SEQ[0 : len(SEQ)-1])
	fmt.Println("Search for SEQ returns", a, b, c)
}
//-----------------------------------------------------------------------------
// Save the index to directory.
func (I *IndexC) SaveCompressedIndex() {

	_save_slice := func(s []int64, filename string) {
		f, err := os.Create(filename)
		check_for_error(err)
		defer f.Close()
		w := bufio.NewWriter(f)
		err = binary.Write(w, binary.LittleEndian, s)
		check_for_error(err)
		w.Flush()
	}

	dir := I.input_file + ".fmi"
	os.Mkdir(dir, 0777)

	var wg sync.WaitGroup
	wg.Add(len(I.SYMBOLS) + 2)

	go func() {
		defer wg.Done()
		_save_slice(I.SA, path.Join(dir, "sa"))
	}()

	go func() {
		defer wg.Done()
		err := ioutil.WriteFile(path.Join(dir, "bwt"), I.BWT, 0666)
		check_for_error(err)
	}()

	for symb := range I.OCC {
		go func(symb byte) {
			defer wg.Done()
			_save_slice(I.OCC[symb], path.Join(dir, "occ."+string(symb)))
		}(symb)
	}

	f, err := os.Create(path.Join(dir, "others"))
	check_for_error(err)
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "%d %d %d %d\n", I.LEN, I.OCC_SIZE, I.END_POS, I.M)
	for i := 0; i < len(I.SYMBOLS); i++ {
		symb := byte(I.SYMBOLS[i])
		fmt.Fprintf(w, "%s %d %d %d\n", string(symb), I.Freq[symb], I.C[symb], I.EP[symb])
	}
	w.Flush()
	wg.Wait()
}
//-----------------------------------------------------------------------------
// Load FM index. Usage:  idx := Load(index_file)
func LoadCompressedIndex(dir string) *IndexC {

	_load_slice := func(filename string, length int64) []int64 {
		f, err := os.Open(filename)
		check_for_error(err)
		defer f.Close()

		v := make([]int64, length)
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanBytes)
		for i := 0; scanner.Scan(); i++ {
			// convert 8 consecutive bytes to a int64 number
			v[i] = int64(scanner.Bytes()[0])
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 8
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 16
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 24
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 32
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 40
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 48
			scanner.Scan()
			v[i] += int64(scanner.Bytes()[0]) << 56
		}
		// r := bufio.NewReader(f)
		// binary.Read(r, binary.LittleEndian, v)
		return v
	}

	I := new(IndexC)

	// First, load "others"
	f, err := os.Open(path.Join(dir, "others"))
	check_for_error(err)
	defer f.Close()

	var symb byte
	var freq, c, ep int64
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	fmt.Sscanf(scanner.Text(), "%d%d%d%d\n", &I.LEN, &I.OCC_SIZE, &I.END_POS, &I.M)

	I.Freq = make(map[byte]int64)
	I.C = make(map[byte]int64)
	I.EP = make(map[byte]int64)
	for scanner.Scan() {
		fmt.Sscanf(scanner.Text(), "%c%d%d%d", &symb, &freq, &c, &ep)
		I.SYMBOLS = append(I.SYMBOLS, int(symb))
		I.Freq[symb], I.C[symb], I.EP[symb] = freq, c, ep
	}

	// Second, load Suffix array, BWT and OCC
	I.OCC = make(map[byte][]int64)
	var wg sync.WaitGroup
	wg.Add(len(I.SYMBOLS) + 2)
	go func() {
		defer wg.Done()
		I.SA = _load_slice(path.Join(dir, "sa"), I.LEN)
	}()

	go func() {
		defer wg.Done()
		I.BWT, err = ioutil.ReadFile(path.Join(dir, "bwt"))
		check_for_error(err)
	}()

	Symb_OCC_chan := make(chan Symb_OCC)
	for _, symb := range I.SYMBOLS {
		go func(symb int) {
			defer wg.Done()
			Symb_OCC_chan <- Symb_OCC{symb, _load_slice(path.Join(dir, "occ."+string(symb)), I.OCC_SIZE)}
		}(symb)
	}
	go func() {
		wg.Wait()
		close(Symb_OCC_chan)
	}()

	for symb_occ := range Symb_OCC_chan {
		I.OCC[byte(symb_occ.Symb)] = symb_occ.OCC
	}
	return I
}

//-----------------------------------------------------------------------------
