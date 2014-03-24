/*
   Copyright 2013 Vinhthuy Phan
   Usage:  go run generate_reads.go --help

Expected input format: fasta format file.

Each line of the output has the following format:
   R  N p1 … pN  E q1 … qE
where
   + R is the content of the read
   + N is the number of occurrences of this read in the genome
   + p1 to pN are the locations of the read in the genome
   + E is the number of errors
   + q1 to qE is the locations of the errors in the read.  If E is equal to 0, then this list is empty.

Example:

ATTTAGGTACTACTAACTCGTGTGTTGCAGTTTTCGAAAATGAAAAAGTCGCGTTATAGAAAATTCAGAACGTGCCCATACTACTCACCCTTCTATAATT 1 25 2 51 70

   This means the read occurs at 1 genome location: 25, with 2 errors at read locations: 51 and 70.
*/
package main

import (
   "github.com/vtphan/fmi"
	"fmt"
	"flag"
	"math/rand"
	"time"
   "bytes"
)

var Debug bool
var rand_gen = rand.New(rand.NewSource(time.Now().UnixNano()))

//-----------------------------------------------------------------------------

func random_error(base byte) byte {
   not_A, not_T, not_C, not_G := []byte{'C','G','T'}, []byte{'C','G','A'}, []byte{'A','G','T'}, []byte{'C','A','T'}
   c := base
   switch (base){
      case 'A': c = not_A[rand_gen.Intn(3)]
      case 'C': c = not_C[rand_gen.Intn(3)]
      case 'G': c = not_G[rand_gen.Intn(3)]
      case 'T': c = not_T[rand_gen.Intn(3)]
   }
   if ! Debug {
      return c
   }
   return bytes.ToLower([]byte{c})[0]
}

//-----------------------------------------------------------------------------
// return true if SEQ[pos: pos+length] is NNNNNNNNNNNN

func justN(pos, read_len int) bool {
   for i:=pos; i<pos+read_len; i++ {
      if fmi.SEQ[pos] != 'N' {
         return false
      }
   }
   return true
}

//-----------------------------------------------------------------------------


func main() {
	var seq_file = flag.String("s", "", "Specify a file containing the sequence.")
	var rl = flag.Int("l", 100, "Read length.")
   var coverage = flag.Float64("c", 2.0, "Coverage")
	var error_rate = flag.Float64("e", 0.01, "Error rate.")
	flag.BoolVar(&Debug, "debug", false, "Turn on debug mode.")
	flag.Parse()
	read_len := int(*rl)
	if *seq_file != "" {
		if *coverage > 0 && read_len > 0 {
			idx := fmi.New(*seq_file)
         num_of_reads := int(*coverage * float64(idx.LEN) / float64(read_len))
			read_indices := make([]int, num_of_reads)
			the_read := make([]byte, read_len)
         var rand_pos int

			for i:=0; i<num_of_reads; i++ {
            rand_pos = int(rand_gen.Intn(int(idx.LEN - read_len)))
            if justN(rand_pos, int(read_len)) {
               i--
               continue
            }

				read_indices = idx.Repeat(rand_pos, read_len)
            var errors []int
            if int(rand_pos+read_len) >= len(fmi.SEQ) {
               panic("Read may be shorter than wanted.")
            }

            copy(the_read, fmi.SEQ[rand_pos: rand_pos + read_len])
            for k:=0; k<len(the_read); k++ {
               if rand_gen.Float64() < *error_rate {
                  the_read[k] = random_error(the_read[k])
                  errors = append(errors, k)
               }
            }
            if Debug {
	            for j:=0; j<int(rand_pos); j++ {
   	            fmt.Printf(" ")
      	      }
      	   }
				fmt.Printf("%s %d ", the_read, len(read_indices))
				for j:=0; j<len(read_indices); j++ {
					fmt.Printf("%d ", read_indices[j])
				}
            fmt.Printf("%d", len(errors))
            for j:=0; j<len(errors); j++ {
               fmt.Printf(" %d", errors[j])
            }
				fmt.Println()
			}
		} else {
			idx := fmi.New(*seq_file)
			idx.Save(*seq_file)
		}
	} else {
		fmt.Println("Must provide sequence file")
	}
}