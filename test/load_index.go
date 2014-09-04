package main

import (
   "github.com/vtphan/fmi"
   "os"
   "fmt"
   "runtime"
)

var Debug bool

//-----------------------------------------------------------------------------
func main() {
   runtime.GOMAXPROCS(runtime.NumCPU())
   idx := fmi.Load(os.Args[1])
   fmt.Println(idx.LEN)
   // idx.Show()
}