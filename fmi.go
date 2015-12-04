/*
   Copyright 2015 Vinhthuy Phan
	Compressed FM index.
	Todo:
	- compress
*/
package fmi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"sync"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

// var SA []uint64

type Index struct {
	BWT []byte
	SA  []uint64          // suffix array
	C   map[byte]uint64   // count table
	OCC map[byte][]uint64 // occurence table

	END_POS uint64          // position of "$" in the text
	SYMBOLS []int           // sorted symbols
	EP      map[byte]uint64 // ending row/position of each symbol

	LEN  uint64
	Freq map[byte]uint64 // Frequency of each symbol
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
	OCC  []uint64
}

//-----------------------------------------------------------------------------
// Load FM index. Usage:  idx := Load(index_file)
func Load(dir string) *Index {

	_load_slice := func(filename string, length uint64) []uint64 {
		f, err := os.Open(filename)
		check_for_error(err)
		defer f.Close()

		v := make([]uint64, length)
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanBytes)
		for i := 0; scanner.Scan(); i++ {
			// convert 8 consecutive bytes to a uint64 number
			v[i] = uint64(scanner.Bytes()[0])
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 8
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 16
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 24
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 32
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 40
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 48
			scanner.Scan()
			v[i] += uint64(scanner.Bytes()[0]) << 56
		}
		// r := bufio.NewReader(f)
		// binary.Read(r, binary.LittleEndian, v)
		return v
	}

	I := new(Index)

	// First, load "others"
	f, err := os.Open(path.Join(dir, "others"))
	check_for_error(err)
	defer f.Close()

	var symb byte
	var freq, c, ep uint64
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	fmt.Sscanf(scanner.Text(), "%d%d\n", &I.LEN, &I.END_POS)

	I.Freq = make(map[byte]uint64)
	I.C = make(map[byte]uint64)
	I.EP = make(map[byte]uint64)
	for scanner.Scan() {
		fmt.Sscanf(scanner.Text(), "%c%d%d%d", &symb, &freq, &c, &ep)
		I.SYMBOLS = append(I.SYMBOLS, int(symb))
		I.Freq[symb], I.C[symb], I.EP[symb] = freq, c, ep
	}

	// Second, load Suffix array, BWT and OCC
	I.OCC = make(map[byte][]uint64)
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
			Symb_OCC_chan <- Symb_OCC{symb, _load_slice(path.Join(dir, "occ."+string(symb)), I.LEN)}
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
// Save the index to directory.
func (I *Index) Save(dirname string) {

	_save_slice := func(s []uint64, filename string) {
		f, err := os.Create(filename)
		check_for_error(err)
		defer f.Close()
		w := bufio.NewWriter(f)
		err = binary.Write(w, binary.LittleEndian, s)
		check_for_error(err)
		w.Flush()
	}

	dir := dirname + ".index"
	os.Mkdir(dir, 0777)

	var wg sync.WaitGroup
	wg.Add(len(I.SYMBOLS) + 2)

	go func() {
		defer wg.Done()
		_save_slice(I.SA, path.Join(dir, "sa"))
	}()

	go func() {
		defer wg.Done()
		err := ioutil.WriteFile(path.Join(dir, "bwt"), I.BWT, 0777)
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
	fmt.Fprintf(w, "%d %d\n", I.LEN, I.END_POS)
	for i := 0; i < len(I.SYMBOLS); i++ {
		symb := byte(I.SYMBOLS[i])
		fmt.Fprintf(w, "%s %d %d %d\n", string(symb), I.Freq[symb], I.C[symb], I.EP[symb])
	}
	w.Flush()
	wg.Wait()
}

//-----------------------------------------------------------------------------
// BWT is saved into a separate file
func (I *Index) build_suffix_array() {
	I.LEN = uint64(len(SEQ))
	I.SA = make([]uint64, I.LEN)
	SA := make([]int, I.LEN)
	ws := &WorkSpace{}
	ws.ComputeSuffixArray(SEQ, SA)
	for i := range SA {
		I.SA[i] = uint64(SA[i])
	}
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
	I.Freq = make(map[byte]uint64)
	I.BWT = make([]byte, I.LEN)
	var i uint64
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

	I.C = make(map[byte]uint64)
	I.OCC = make(map[byte][]uint64)
	for c := range I.Freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]uint64, I.LEN)
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	I.EP = make(map[byte]uint64)
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
	var offset uint64
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
func ReadFasta(file string) {
	f, err := os.Open(file)
	check_for_error(err)
	defer f.Close()

	if file[len(file)-6:] != ".fasta" {
		panic("ReadFasta:" + file + "is not a fasta file.")
	}

	scanner := bufio.NewScanner(f)
	byte_array := make([]byte, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 {
			if line[0] != '>' {
				byte_array = append(byte_array, bytes.Trim(line, "\n\r ")...)
			} else if len(byte_array) > 0 {
				byte_array = append(byte_array, byte('|'))
			}
		}
	}
	SEQ = append(byte_array, byte('$'))
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
