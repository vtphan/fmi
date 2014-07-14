// Copyright 2013 Vinhthuy Phan
// Package implements uncompressed FM index
//
package main

import (
   "github.com/vtphan/fmi"
   "fmt"
   "bufio"
   "os"
   "flag"
)

var Debug bool

//-----------------------------------------------------------------------------
func main() {
   var build_file = flag.String("build", "", "Specify a file, from which to build FM index.")
   var index_file = flag.String("i", "", "index file")
   var queries_file = flag.String("q", "", "queries file")
   flag.BoolVar(&Debug, "debug", false, "Turn on debug mode.")
   flag.Parse()

   if *build_file != "" {
      idx := fmi.New(*build_file)
      idx.Save(*build_file)
   } else if *index_file!="" && *queries_file!="" {
      idx := fmi.Load(*index_file)
      // fmt.Println("index", idx)
      f, err := os.Open(*queries_file)
      if err != nil { panic("error opening file " + *queries_file) }
      r := bufio.NewReader(f)
      for {
         line, err := r.ReadBytes('\n')
         if err != nil { break }
         if len(line) > 1 {
            line = line[0:len(line)-1]
            result := idx.Search(line)
            if len(result) == 0 {
               fmt.Println("na")
            } else {
               for _, v := range result {
                  fmt.Printf("%d ", v)
               }
               fmt.Println()
            }
         }
      }
   }

}