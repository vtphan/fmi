package main

import (
   "github.com/vtphan/fmi"
   "os"
   "runtime"
)

//-----------------------------------------------------------------------------
func main() {
   runtime.GOMAXPROCS(runtime.NumCPU())
   idx := fmi.New(os.Args[1])
   idx.Save(os.Args[1])
   // idx.Show()
}