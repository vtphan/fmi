/*
   Copyright 2014 Vinhthuy Phan
	FM index for DNA sequences.  A DNA sequence must contain only upper case letters of A, C, G, T, and N.
   N will be ignored. Reads contain N will not be located.
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
   "encoding/binary"
	"bufio"
	"path"
)

var Debug bool

//-----------------------------------------------------------------------------
// Global variables: sequence (SEQ), suffix array (SA), BWT, FM index (C, OCC)
//-----------------------------------------------------------------------------
var SEQ []byte

type Index struct{
   LEN uint32
   EP map[byte]uint32            // ending row/position of each symbol
	C map[byte]uint32  				// count table
   OCC map[byte][]uint32         // occurence table
   SA []uint32                   // suffix array
}
//

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
	_load(&I.EP, path.Join(dir, "ep"))
	_load(&I.LEN, path.Join(dir, "len"))

	I.OCC = make(map[byte][]uint32)
	for c := range(I.C) {
      I.OCC[c] = _load_occ(path.Join(dir, "occ."+string(c)), I.LEN)
   }
	return I
}

//-----------------------------------------------------------------------------
func (I *Index) Save(file string) {
	dir := file + ".index"
	os.Mkdir(dir, 0777)

	for symb := range I.OCC {
		_save(I.OCC[symb], path.Join(dir, "occ." + string(symb)),"Fail to save to occ."+string(symb))
	}
	_save(I.SA, path.Join(dir,"sa"), "Fail to save suffix array")
	_save(I.C, path.Join(dir,"c"), "Fail to save count")
	_save(I.EP, path.Join(dir,"ep"), "Fail to save ep")
	_save(I.LEN, path.Join(dir,"len"), "Fail to save len")
}

//-----------------------------------------------------------------------------
func (I *Index) SaveIndex() {
   buf := new(bytes.Buffer)
   err := binary.Write(buf, binary.LittleEndian, I)
   if err != nil {
      fmt.Println("binary.Write failed:", err)
   }
   fmt.Println("% x", buf.Bytes())
}
//-----------------------------------------------------------------------------
// BWT is saved into a separate file
func (I *Index) build_suffix_array() {
	I.LEN = uint32(len(SEQ))
	I.SA = make([]uint32, I.LEN)
   SA := qsufsort(SEQ)
   for i := range SA {
      I.SA[i] = uint32(SA[i])
   }
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
	freq := make(map[byte]uint32)
   symbols := make([]int, 0)
   I.C = make(map[byte]uint32)
   I.OCC = make(map[byte][]uint32)
	bwt := make([]byte, I.LEN)
	var i uint32
	for i = 0; i < I.LEN; i++ {
		freq[SEQ[i]]++
      I.C[SEQ[i]]++
		bwt[i] = SEQ[(I.LEN+I.SA[i]-1)%I.LEN]
	}

	for c := range freq {
		symbols = append(symbols, int(c))
		I.OCC[c] = make([]uint32, I.LEN)
      I.C[c] = 0
	}
	sort.Ints(symbols)
	I.EP = make(map[byte]uint32)
	for j := 1; j < len(symbols); j++ {
		curr_c, prev_c := byte(symbols[j]), byte(symbols[j-1])
		I.C[curr_c] = I.C[prev_c] + freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + freq[curr_c] - 1
	}

	for j := 0; j < len(bwt); j++ {
		I.OCC[bwt[j]][j] = 1
		if j > 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][j] += I.OCC[symbol][j-1]
			}
		}
	}
   I.show()

	delete(I.OCC, '$')
	delete(I.C, '$')
   delete(I.OCC, 'Z')
   delete(I.C, 'Z')

   I.show()
}

//-----------------------------------------------------------------------------
// Search for all occurences of pattern in SEQ
//-----------------------------------------------------------------------------

func (I *Index) Search(pattern []byte) []int {
   sp, ep, _ := I.SearchFrom(pattern, len(pattern)-1)
	res := make([]int, ep-sp+1)
	for k:=sp; k<=ep; k++ {
		res[k-sp] = int(I.SA[k])
	}
 	return res
}

//-----------------------------------------------------------------------------
// Returns starting, ending positions (sp, ep) and last-matched position (i)
//-----------------------------------------------------------------------------
func (I *Index) SearchFrom(pattern []byte, start_pos int) (int, int, int) {
   var offset uint32
   var i int

   c := pattern[start_pos]
   sp, ok := I.C[c]
   if ! ok {
      return 0, -1, -1
   }
   ep := I.EP[c]
   for i=int(start_pos-1); sp <= ep && i >= 0; i-- {
      c = pattern[i]
      offset, ok = I.C[c]
      if ok {
         sp = offset + I.OCC[c][sp - 1]
         ep = offset + I.OCC[c][ep] - 1
      } else {
         return 0, -1, -1
      }
   }
   return int(sp), int(ep), i+1
}

//-----------------------------------------------------------------------------
// Search for all repeats of SEQ[j:j+read_len] in SEQ
//-----------------------------------------------------------------------------

func (I *Index) Repeat(j, read_len int) []int {
	var sp, ep, offset uint32
	var ok bool

	c := SEQ[j+read_len-1]
	sp, ok = I.C[c]
	if ! ok {
		return make([]int, 0)
	}
	ep = I.EP[c]
	for i:=int(read_len-2); sp <= ep && i >= 0; i-- {
  		c = SEQ[j+int(i)]
  		offset, ok = I.C[c]
  		if ok {
			sp = offset + I.OCC[c][sp - 1]
			ep = offset + I.OCC[c][ep] - 1
		} else {
			return make([]int, 0)
		}
	}
	res := make([]int, ep-sp+1)
	for k:=sp; k<=ep; k++ {
		res[k-sp] = int(I.SA[k])
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

   // replace N with Y and '*' with W (last character is '$')
   for i:=0; i<len(SEQ)-1; i++ {
      if SEQ[i] == 'N' {
         SEQ[i] = 'Z'
      } else if SEQ[i] != 'A' && SEQ[i] != 'C' && SEQ[i] != 'G' && SEQ[i] != 'T' {
         panic("Sequence contains an illegal character: " + string(SEQ[i]))
      }
   }
}



//-----------------------------------------------------------------------------
func (I *Index) show() {
   fmt.Println("Sequence length", I.LEN)
	fmt.Printf(" %8s  %8s  OCC\n", "EP", "C")
   for c := range(I.OCC) {
		fmt.Printf("%c%8d  %8d  %d\n", c, I.EP[c], I.C[c], I.OCC[c])
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

