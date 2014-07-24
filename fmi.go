/*
   Copyright 2014 Vinhthuy Phan
	FM index for DNA sequences.  A DNA sequence must contain only upper case letters of A, C, G, T, and N.
   N will be ignored. Reads that contain N will not be located.
*/
package fmi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	// "sort"
	"log"
	"encoding/gob"
   // "encoding/binary"
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
   END_POS uint32                // rank of terminal symbol $ in the BWT
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
// Load FM index
// Usage:  idx := Load(index_file)
func Load (dir string) *Index {
   _load := func(thing interface{}, filename string) {
      fin,err := os.Open(filename)
      decOCC := gob.NewDecoder(fin)
      err = decOCC.Decode(thing)
      if err != nil {
         fmt.Println("Unable to read file ("+filename+"): ",err)
      }
   }

   _load_occ := func(filename string, Len uint32) []uint32 {
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

	I := new(Index)
	_load(&I.C, path.Join(dir, "c"))
	_load(&I.SA, path.Join(dir, "sa"))
	_load(&I.EP, path.Join(dir, "ep"))
	_load(&I.LEN, path.Join(dir, "len"))
   _load(&I.END_POS, path.Join(dir, "end_pos"))

	I.OCC = make(map[byte][]uint32)
	for c := range(I.C) {
      I.OCC[c] = _load_occ(path.Join(dir, "occ."+string(c)), I.LEN)
   }
	return I
}

//-----------------------------------------------------------------------------
func (I *Index) Save(file string) {
   _save := func(thing interface{}, filename string, error_message string) {
      buffer := new(bytes.Buffer)
      enc := gob.NewEncoder(buffer)
      err := enc.Encode(thing)
      // err := binary.Write(buffer, binary.LittleEndian, thing)
      if err != nil {
         log.Fatal(error_message)
      }
      fmt.Println("save", filename)
      ioutil.WriteFile(filename, buffer.Bytes(), 0600)
   }

	dir := file + ".index"
	os.Mkdir(dir, 0777)

	for symb := range I.OCC {
		_save(I.OCC[symb], path.Join(dir, "occ." + string(symb)),"Fail to save to occ."+string(symb))
	}
	_save(I.SA, path.Join(dir,"sa"), "Fail to save suffix array")
	_save(I.C, path.Join(dir,"c"), "Fail to save count")
	_save(I.EP, path.Join(dir,"ep"), "Fail to save ep")
	_save(I.LEN, path.Join(dir,"len"), "Fail to save len")
   _save(I.END_POS, path.Join(dir,"end_pos"), "Fail to save end_pos")
}

//-----------------------------------------------------------------------------
// func (I *Index) SaveIndex() {
//    buf := new(bytes.Buffer)
//    err := binary.Write(buf, binary.LittleEndian, I)
//    if err != nil {
//       fmt.Println("binary.Write failed:", err)
//    }
//    fmt.Println("% x", buf.Bytes())
// }
//-----------------------------------------------------------------------------
func (I *Index) build_suffix_array() {
	I.LEN = uint32(len(SEQ))
	I.SA = make([]uint32, I.LEN)
   suffix_array := qsufsort(SEQ)
   for i := range suffix_array {
      I.SA[i] = uint32(suffix_array[i])
   }
}

//-----------------------------------------------------------------------------
func (I *Index) build_bwt_fmindex() {
   I.C = make(map[byte]uint32)
   I.OCC = make(map[byte][]uint32)
   I.EP = make(map[byte]uint32)
	freq := make(map[byte]uint32)
	bwt := make([]byte, I.LEN)

	for i := uint32(0); i < I.LEN; i++ {
		bwt[i] = SEQ[(I.LEN+I.SA[i]-1)%I.LEN]
      if bwt[i] == '$' {
         I.END_POS = i
      }
      if SEQ[i] != '$' {
         freq[SEQ[i]]++
      }
	}
   for c := range freq {
      I.OCC[c] = make([]uint32, I.LEN)
      I.C[c] = 0
   }
   for j := 0; j < len(bwt); j++ {
      if bwt[j] != '$' {
         I.OCC[bwt[j]][j] = 1
      }
      if j > 0 {
         for symbol := range I.OCC {
            I.OCC[symbol][j] += I.OCC[symbol][j-1]
         }
      }
   }
   I.C['A'] = 1
   I.C['C'] = I.C['A'] + freq['A']
   I.C['G'] = I.C['C'] + freq['C']
   I.C['T'] = I.C['G'] + freq['G']
   I.EP['A'] = freq['A']
   I.EP['C'] = I.C['C'] + freq['C'] - 1
   I.EP['G'] = I.C['G'] + freq['G'] - 1
   I.EP['T'] = I.C['T'] + freq['T'] - 1

   delete(I.OCC, 'Z')
   delete(I.C, 'Z')
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
   // var sp_occ, ep_occ uint32

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
         // if c != 'T' {
         //    sp_occ = I.OCC[c][sp-1]
         //    ep_occ = I.OCC[c][ep]
         // } else {
         //    // i+1 = sum of occ of all symbols (including $) at i
         //    sp_occ = sp - (I.OCC['A'][sp-1] + I.OCC['C'][sp-1] + I.OCC['G'][sp-1])
         //    if sp >= I.END_POS {
         //       sp_occ -= 1
         //    }
         //    ep_occ = ep+1 - (I.OCC['A'][ep] + I.OCC['C'][ep] + I.OCC['G'][ep])
         //    if ep >= I.END_POS {
         //       ep_occ -= 1
         //    }
         // }
         // // fmt.Println("\t", string(c), sp_occ, ep_occ, I.OCC[c][sp-1], I.OCC[c][ep], "\t", sp, I.OCC['A'][sp-1], I.OCC['C'][sp-1], I.OCC['G'][sp-1], I.OCC['T'][sp-1] )
         // sp = offset + sp_occ
         // ep = offset + ep_occ - 1
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
func (I *Index) Show() {
   fmt.Println("Sequence length", I.LEN, "; $ position in BWT", I.END_POS)
	fmt.Printf(" %8s  %8s  OCC\n", "EP", "C")
   for c := range(I.OCC) {
		fmt.Printf("%c%8d  %8d  %d\n", c, I.EP[c], I.C[c], I.OCC[c])
	}
   // for i:=uint32(0); i<I.LEN; i++ {
   //    sum := I.OCC['A'][i] + I.OCC['C'][i] + I.OCC['G'][i] + I.OCC['T'][i]
   //    fmt.Printf("%d = %d\t%d %d %d %d %d\n", i+1, sum, I.OCC['A'][i], I.OCC['C'][i], I.OCC['G'][i], I.OCC['T'][i], I.OCC['$'][i])
   // }
   // for i:=uint32(0); i < I.LEN; i++ {
   //    fmt.Printf("%d\t%s\n", uint32(i), string(SEQ[I.SA[i]]))
   // }
}

//-----------------------------------------------------------------------------
func print_byte_array(a []byte) {
	for i := 0; i < len(a); i++ {
		fmt.Printf("%c", a[i])
	}
	fmt.Println()
}

//-----------------------------------------------------------------------------

