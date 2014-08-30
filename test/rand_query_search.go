// Copyright 2013 Vinhthuy Phan
// Package implements uncompressed FM index
//
package main

import (
   "github.com/vtphan/fmi"
   "math/rand"
   "fmt"
   "flag"
   // "time"
)


//-----------------------------------------------------------------------------
func main() {
   var seq_file = flag.String("s", "", "sequence file")
   var n = flag.Int("n", 10, "number of queries")
   flag.Parse()

   if *seq_file == "" {
      panic("Must provide sequence file (-s option)")
   }
   fmt.Println("Build index for", *seq_file)
   idx := fmi.New(*seq_file)
   fmt.Println("Save index")
   idx.Save(*seq_file)
   fmt.Println("Reload index")
   idx = fmi.Load(*seq_file + ".index")
   fmt.Println("Sequence: ", string(fmi.SEQ))
   idx.Show()
   fmt.Println("Begin random querying")

   // r := rand.New(rand.NewSource(time.Now().UnixNano()))
   r := rand.New(rand.NewSource(7))

   for i:=0; i<*n; i++ {
      q_len := 2 + r.Intn(int(idx.LEN)-2)
      q_pos := r.Intn(int(idx.LEN) - q_len)
      query := fmi.SEQ[q_pos : q_pos + q_len]
      result := idx.Search(query)

      if len(result) == 0 {
         fmt.Println(string(query), " does not occur in sequence.")
      } else {
         for _, v := range result {
            fmt.Printf("%t %s %d\n", string(query) == string(fmi.SEQ[v:v+q_len]), string(query), v)
         }
      }
   }
}