/*
   Copyright 2013 Vinhthuy Phan
	FM index
*/
package fmi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"log"
	"encoding/gob"
	"bufio"
	"path"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

type Index struct{
	SA []uint32 						// suffix array
	C map[byte]uint32  				// count table
	OCC map[byte][]uint32 			// occurence table

	END_POS uint32 					// position of "$" in the text
	SYMBOLS []int  					// sorted symbols
	EP map[byte]uint32 				// ending row/position of each symbol

	LEN uint32
	freq map[byte]uint32  // Frequency of each symbol
}
//
//-----------------------------------------------------------------------------
type BySuffix []uint32

func (s BySuffix) Len() int { return len(s) }
func (s BySuffix) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySuffix) Less(i, j int) bool { return (bytes.Compare(SEQ[s[i]:], SEQ[s[j]:]) == -1) }


//-----------------------------------------------------------------------------
// Build FM index given the file storing the text.

func New (file string) *Index {
	I := new(Index)
	ReadSequence(file)
	I.build_suffix_array()
	I.build_bwt_fmindex()
	return I
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
func _load(thing interface{}, filename string) {
	fin,err := os.Open(filename)
	decOCC := gob.NewDecoder(fin)
	err = decOCC.Decode(thing)
	if err != nil {
		fmt.Println("Unable to read file ("+filename+"): ",err)
	}
}

//-----------------------------------------------------------------------------
func _load_occ(filename string, Len uint32) []uint32 {
	thing := make([]uint32, Len)
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

	I.OCC = make(map[byte][]uint32)
	for _,symb := range I.SYMBOLS {
		I.OCC[byte(symb)] = _load_occ(path.Join(dir, "occ."+string(symb)), I.LEN)
	}
	return I
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
// BWT is saved into a separate file
func (I *Index) build_suffix_array() {
	I.LEN = uint32(len(SEQ))
	// I.SA = qsufsort(SEQ)
	I.SA = make([]uint32, I.LEN)
	var i uint32
	for i = 0; i < I.LEN; i++ {
		I.SA[i] = i
	}
	sort.Sort(BySuffix(I.SA))
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
	I.freq = make(map[byte]uint32)
	bwt := make([]byte, I.LEN)
	var i uint32
	for i = 0; i < I.LEN; i++ {
		I.freq[SEQ[i]]++
		bwt[i] = SEQ[(I.LEN+I.SA[i]-1)%I.LEN]
		if bwt[i] == '$' {
			I.END_POS = i
		}
	}

	I.C = make(map[byte]uint32)
	I.OCC = make(map[byte][]uint32)
	for c := range I.freq {
		I.SYMBOLS = append(I.SYMBOLS, int(c))
		I.OCC[c] = make([]uint32, I.LEN)
		I.C[c] = 0
	}
	sort.Ints(I.SYMBOLS)
	I.EP = make(map[byte]uint32)
	for i := 1; i < len(I.SYMBOLS); i++ {
		curr_c, prev_c := byte(I.SYMBOLS[i]), byte(I.SYMBOLS[i-1])
		I.C[curr_c] = I.C[prev_c] + I.freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.freq[curr_c] - 1
	}

	for i := 0; i < len(bwt); i++ {
		I.OCC[bwt[i]][i] = 1
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
// Search for all occurences of pattern in SEQ
//-----------------------------------------------------------------------------

func (I *Index) Search(pattern []byte) []uint32 {
	var sp, ep, offset uint32
	var ok bool

	c := pattern[len(pattern) - 1]
	sp, ok = I.C[c]
	if ! ok {
		return make([]uint32, 0)
	}
	ep = I.EP[c]
	for i:= len(pattern)-2; sp <= ep && i >= 0; i-- {
  		c = pattern[uint32(i)]
  		offset, ok = I.C[c]
  		if ok {
			sp = offset + I.OCC[c][sp - 1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			return make([]uint32, 0)
		}
	}
	res := make([]uint32, ep-sp+1)
	for k:=sp; k<=ep; k++ {
		res[k-sp] = I.SA[k]
	}
 	return res
}

//-----------------------------------------------------------------------------
// Search for all repeats of SEQ[j:j+read_len] in SEQ
//-----------------------------------------------------------------------------

func (I *Index) Repeat(j, read_len uint32) []uint32 {
	var sp, ep, offset uint32
	var ok bool

	c := SEQ[j+read_len-1]
	sp, ok = I.C[c]
	if ! ok {
		return make([]uint32, 0)
	}
	ep = I.EP[c]
	for i:=int(read_len-2); sp <= ep && i >= 0; i-- {
  		c = SEQ[j+uint32(i)]
  		offset, ok = I.C[c]
  		if ok {
			sp = offset + I.OCC[c][sp - 1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			return make([]uint32, 0)
		}
	}
	res := make([]uint32, ep-sp+1)
	for k:=sp; k<=ep; k++ {
		res[k-sp] = I.SA[k]
	}
 	return res
}

//-----------------------------------------------------------------------------
func ReadSequence(file string) {
   f, err := os.Open(file)
   if err != nil {
      panic(err)
   }
   defer f.Close()

   if file[len(file)-6:] == ".fasta" {
	   scanner := bufio.NewScanner(f)
	   byte_array := make([]byte, 0)
	   for scanner.Scan() {
	      line := scanner.Bytes()
	      if len(line)>0 && line[0] != '>' {
	         byte_array = append(byte_array, bytes.Trim(line,"\n\r ")...)
	      }
	   }
		SEQ = append(byte_array, byte('$'))
	} else {
		byte_array, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		SEQ = append(bytes.Trim(byte_array, "\n\r "), byte('$'))
	}
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
func print_byte_array(a []byte) {
	for i := 0; i < len(a); i++ {
		fmt.Printf("%c", a[i])
	}
	fmt.Println()
}

//-----------------------------------------------------------------------------

