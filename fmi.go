/*
   Copyright 2013 Vinhthuy Phan
	FM index.
*/
package fmi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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

type Index struct {
	SA  []uint32          // suffix array
	C   map[byte]uint32   // count table
	OCC map[byte][]uint32 // occurence table

	END_POS uint32          // position of "$" in the text
	SYMBOLS []int           // sorted symbols
	EP      map[byte]uint32 // ending row/position of each symbol

	LEN  uint32
	Freq map[byte]uint32 // Frequency of each symbol
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
	OCC  []uint32
}

//-----------------------------------------------------------------------------
// Load FM index. Usage:  idx := Load(index_file)
func Load(dir string) *Index {

	_load_slice := func(filename string, length uint32) []uint32 {
		f, err := os.Open(filename)
		check_for_error(err)
		defer f.Close()

		v := make([]uint32, length)
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanBytes)
		for i := 0; scanner.Scan(); i++ {
			// convert 4 consecutive bytes to a uint32 number
			v[i] = uint32(scanner.Bytes()[0])
			scanner.Scan()
			v[i] += uint32(scanner.Bytes()[0]) << 8
			scanner.Scan()
			v[i] += uint32(scanner.Bytes()[0]) << 16
			scanner.Scan()
			v[i] += uint32(scanner.Bytes()[0]) << 24
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
	var freq, c, ep uint32
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	fmt.Sscanf(scanner.Text(), "%d%d\n", &I.LEN, &I.END_POS)

	I.Freq = make(map[byte]uint32)
	I.C = make(map[byte]uint32)
	I.EP = make(map[byte]uint32)
	for scanner.Scan() {
		fmt.Sscanf(scanner.Text(), "%c%d%d%d", &symb, &freq, &c, &ep)
		I.SYMBOLS = append(I.SYMBOLS, int(symb))
		I.Freq[symb], I.C[symb], I.EP[symb] = freq, c, ep
	}

	// Second, load Suffix array and OCC
	I.OCC = make(map[byte][]uint32)
	var wg sync.WaitGroup
	wg.Add(len(I.SYMBOLS) + 1)
	go func() {
		defer wg.Done()
		I.SA = _load_slice(path.Join(dir, "sa"), I.LEN)
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

	_save_slice := func(s []uint32, filename string) {
		f, err := os.Create(filename)
		check_for_error(err)
		defer f.Close()
		w := bufio.NewWriter(f)
		binary.Write(w, binary.LittleEndian, s)
		w.Flush()
	}

	dir := dirname + ".index"
	os.Mkdir(dir, 0777)

	var wg sync.WaitGroup
	wg.Add(len(I.SYMBOLS) + 1)

	go func() {
		defer wg.Done()
		_save_slice(I.SA, path.Join(dir, "sa"))
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
	I.LEN = uint32(len(SEQ))
	I.SA = make([]uint32, I.LEN)
	SA := make([]int, I.LEN)
	ws := &WorkSpace{}
	ws.ComputeSuffixArray(SEQ, SA)
	for i := range SA {
		I.SA[i] = uint32(SA[i])
	}
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
	I.Freq = make(map[byte]uint32)
	bwt := make([]byte, I.LEN)
	var i uint32
	for i = 0; i < I.LEN; i++ {
		I.Freq[SEQ[i]]++
		if I.SA[i] == 0 {
			bwt[i] = SEQ[I.LEN-1]
		} else {
			bwt[i] = SEQ[I.SA[i]-1]
		}
		if bwt[i] == '$' {
			I.END_POS = i // this is no longer correct due to existence of many $'s
		}
	}

	I.C = make(map[byte]uint32)
	I.OCC = make(map[byte][]uint32)
	for c := range I.Freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]uint32, I.LEN)
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	I.EP = make(map[byte]uint32)
	for j := 1; j < len(I.SYMBOLS); j++ {
		curr_c, prev_c := byte(I.SYMBOLS[j]), byte(I.SYMBOLS[j-1])
		I.C[curr_c] = I.C[prev_c] + I.Freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.Freq[curr_c] - 1
	}

	for j := 0; j < len(bwt); j++ {
		I.OCC[bwt[j]][j] = 1
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

func (I *Index) Check() {
	fmt.Println("Search for SEQ returns", I.Search(SEQ[0:len(SEQ)-1]))
}

// Search for all occurences of pattern in SEQ
func (I *Index) Search(pattern []byte) []int {
	sp, ep, _ := I.SearchFrom(pattern, len(pattern)-1)
	res := make([]int, ep-sp+1)
	for k := sp; k <= ep; k++ {
		res[k-sp] = int(I.SA[k])
	}
	return res
}


// Returns starting, ending positions (sp, ep) and last-matched position (i)
func (I *Index) SearchFrom(pattern []byte, start_pos int) (int, int, int) {
	var offset uint32
	var i int

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

// Search for all repeats of SEQ[j:j+read_len] in SEQ
// Panic: index of out range if encounter an unknown character
func (I *Index) Repeat(j, read_len int) []int {
	var sp, ep, offset uint32
	var ok bool

	c := SEQ[j+read_len-1]
	sp, ok = I.C[c]
	if !ok {
		return make([]int, 0)
	}
	ep = I.EP[c]
	for i := int(read_len - 2); sp <= ep && i >= 0; i-- {
		c = SEQ[j+int(i)]
		offset, ok = I.C[c]
		if ok {
			sp = offset + I.OCC[c][sp-1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			return make([]int, 0)
		}
	}
	res := make([]int, ep-sp+1)
	for k := sp; k <= ep; k++ {
		res[k-sp] = int(I.SA[k])
	}
	return res
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
	fmt.Println()
}

//-----------------------------------------------------------------------------
