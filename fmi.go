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
   "encoding/binary"
	"bufio"
	"path"
   "runtime"
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

var memstats = new(runtime.MemStats)

//-----------------------------------------------------------------------------
// Build FM index given the file storing the text.

func show_memstat(mesg string) {
   runtime.ReadMemStats(memstats)
   log.Printf("%s:\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f", mesg,
      float64(memstats.Alloc)/float64(1<<20),
      float64(memstats.TotalAlloc)/float64(1<<20),
      float64(memstats.Sys)/float64(1<<20),
      float64(memstats.HeapAlloc)/float64(1<<20),
      float64(memstats.HeapSys)/float64(1<<20))
}

func New (file string) *Index {
	I := new(Index)

   show_memstat("before read sequence")
   show_memstat("before read sequence")

	ReadSequence(file)
   show_memstat("after  read sequence")

	I.build_suffix_array()
   show_memstat("after build sufarray")

	I.build_bwt_fmindex()
   show_memstat("after build fm-index")

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

   // _load_occ := func(filename string) []uint32 {
   //    // thing := make([]uint32, Len)
   //    var thing []uint32
   //    fin,err := os.Open(filename)
   //    decOCC := gob.NewDecoder(fin)
   //    err = decOCC.Decode(&thing)
   //    if err != nil {
   //       log.Fatal("Error loading occ table:", filename, err)
   //    }
   //    return thing
   //    // fmt.Println(thing[key], key)
   // }

   _load_slice := func(filename string, length uint32) []uint32 {
      f, err := os.Open(filename)
      if err != nil {
         panic("Error opening input read file")
      }
      defer f.Close()

      v := make([]uint32, length)
      scanner := bufio.NewScanner(f)
      scanner.Split(bufio.ScanBytes)
      var d [4]uint32
      fmt.Println("load slice", length)
      for i:=0; scanner.Scan(); i++ {
         // convert 4 consecutive bytes to a uint32 number
         d[0] = uint32(scanner.Bytes()[0])
         scanner.Scan()
         d[1] = uint32(scanner.Bytes()[0])
         scanner.Scan()
         d[2] = uint32(scanner.Bytes()[0])
         scanner.Scan()
         d[3] = uint32(scanner.Bytes()[0])
         v[i] = uint32(d[0]) + uint32(d[1])<<8 + uint32(d[2])<<16 + uint32(d[3])<<24
      }
      return v
   }

	I := new(Index)
	_load(&I.C, path.Join(dir, "c"))

	_load(&I.EP, path.Join(dir, "ep"))
	_load(&I.LEN, path.Join(dir, "len"))
   _load(&I.END_POS, path.Join(dir, "end_pos"))

   show_memstat("before load sa")
   // _load(&I.SA, path.Join(dir, "sa"))
   I.SA = _load_slice(path.Join(dir, "sa"), I.LEN)
   show_memstat("after  load sa")

	I.OCC = make(map[byte][]uint32)
	for c := range(I.C) {
      // I.OCC[c] = _load_occ(path.Join(dir, "occ."+string(c)))
      I.OCC[c] = _load_slice(path.Join(dir, "occ."+string(c)), I.LEN)
      show_memstat("before load occ." + string(c))
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

   _save_slice := func(s []uint32, filename string) {
      buf := new(bytes.Buffer)
      for i:=0; i<len(s); i++ {
         binary.Write(buf, binary.LittleEndian, s[i])
      }
      fmt.Println("save slice", len(s))
      ioutil.WriteFile(filename, buf.Bytes(), 0600)
   }

	dir := file + ".index"
	os.Mkdir(dir, 0777)

	for symb := range I.OCC {
		// _save(I.OCC[symb], path.Join(dir, "occ." + string(symb)),"Fail to save to occ."+string(symb))
      _save_slice(I.OCC[symb], path.Join(dir, "occ." + string(symb)))
	}
	// _save(I.SA, path.Join(dir,"sa"), "Fail to save suffix array")
   _save_slice(I.SA, path.Join(dir,"sa"))
	_save(I.C, path.Join(dir,"c"), "Fail to save count")
	_save(I.EP, path.Join(dir,"ep"), "Fail to save ep")
	_save(I.LEN, path.Join(dir,"len"), "Fail to save len")
   _save(I.END_POS, path.Join(dir,"end_pos"), "Fail to save end_pos")
}
//-----------------------------------------------------------------------------
func (I *Index) build_suffix_array() {
	I.LEN = uint32(len(SEQ))
	I.SA = make([]uint32, I.LEN)

   show_memstat("\tbefore sort")
   suffix_array := qsufsort(SEQ)
   for i := range suffix_array {
      I.SA[i] = uint32(suffix_array[i])
   }
   show_memstat("\tafter  sort")

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
   if Debug {
      for i:=uint32(0); i<I.LEN; i++ {
         sum := I.OCC['A'][i] + I.OCC['C'][i] + I.OCC['G'][i] + I.OCC['T'][i]
         fmt.Printf("%d = %d\t%d %d %d %d\n", i+1, sum, I.OCC['A'][i], I.OCC['C'][i], I.OCC['G'][i], I.OCC['T'][i])
      }
      for i:=uint32(0); i < I.LEN; i++ {
         fmt.Printf("%d\t%s\n", uint32(i), string(SEQ[I.SA[i]]))
      }
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

