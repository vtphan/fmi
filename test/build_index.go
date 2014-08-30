// Copyright 2013 Vinhthuy Phan
// Package implements uncompressed FM index
//
package main

import (
   "github.com/vtphan/fmi"
   "fmt"
   "os"
   "runtime"
)

var Debug bool

//-----------------------------------------------------------------------------
func main() {
   idx := fmi.New(os.Args[1])
   memstats := new(runtime.MemStats)
   runtime.ReadMemStats(memstats)
   fmt.Printf("memstats before GC: bytes = %d footprint = %d\n", memstats.HeapAlloc, memstats.Sys)
   fmt.Println("Size\t", len(idx.SA))
   idx.Save(os.Args[1])
}