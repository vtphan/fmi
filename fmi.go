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
	"sync"
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
	freq map[byte]uint32          // Frequency of each symbol
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

type Symb_OCC struct {
	Symb int
	OCC []uint32
}

//-----------------------------------------------------------------------------
// Load FM index
// Usage:  idx := Load(index_file)
func Load (dir string) *Index {

	I := new(Index)
	_load(&I.C, path.Join(dir, "c"))
	_load(&I.END_POS, path.Join(dir, "end_pos"))
	_load(&I.SYMBOLS, path.Join(dir, "symbols"))
	_load(&I.EP, path.Join(dir, "ep"))
	_load(&I.LEN, path.Join(dir, "len"))

	I.OCC = make(map[byte][]uint32)

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		_load(&I.SA, path.Join(dir, "sa"))
	}()
	Symb_OCC_chan := make(chan Symb_OCC)
	for _,symb := range I.SYMBOLS[0 : 4] {
		go func(symb int, symb_occ chan Symb_OCC) {
			defer wg.Done()
			tmp_symb_occ := Symb_OCC{}
			tmp_symb_occ.Symb = symb
			tmp_symb_occ.OCC = _load_occ(path.Join(dir, "occ."+string(symb)), I.LEN)
			symb_occ <- tmp_symb_occ
		}(symb, Symb_OCC_chan)
	}
	go func() {
		wg.Wait()
		close(Symb_OCC_chan)
	}()

	for symb_occ := range(Symb_OCC_chan) {
		I.OCC[byte(symb_occ.Symb)] = symb_occ.OCC
	}

	println("Len OCC ", len(I.OCC))
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
	I.SA = make([]uint32, I.LEN)
   SA := qsufsort(SEQ)
   for i := range SA {
      I.SA[i] = uint32(SA[i])
   }
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
	for j := 1; j < len(I.SYMBOLS); j++ {
		curr_c, prev_c := byte(I.SYMBOLS[j]), byte(I.SYMBOLS[j-1])
		I.C[curr_c] = I.C[prev_c] + I.freq[prev_c]
		I.EP[curr_c] = I.C[curr_c] + I.freq[curr_c] - 1
	}

	for j := 0; j < len(bwt); j++ {
		I.OCC[bwt[j]][j] = 1
		if j > 0 {
			for symbol := range I.OCC {
				I.OCC[symbol][j] += I.OCC[symbol][j-1]
			}
		}
	}
	I.SYMBOLS = I.SYMBOLS[1:]
	delete(I.OCC, '$')
	delete(I.C, '$')
   delete(I.OCC, 'Y')
   delete(I.C, 'Y')
   delete(I.OCC, 'W')
   delete(I.C, 'W')

   //fmt.Println(I)
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
         SEQ[i] = 'Y'
      } else if SEQ[i] == '*' {
         SEQ[i] = 'W'
      } else if SEQ[i] != 'A' && SEQ[i] != 'C' && SEQ[i] != 'G' && SEQ[i] != 'T' {
         panic("Sequence contains an illegal character: " + string(SEQ[i]))
      }
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

