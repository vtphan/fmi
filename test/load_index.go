package main

import (
   "github.com/vtphan/fmi"
   "os"
)

var Debug bool

//-----------------------------------------------------------------------------
func main() {
   idx := fmi.Load(os.Args[1])
   idx.Show()
}