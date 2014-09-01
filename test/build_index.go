package main

import (
   "github.com/vtphan/fmi"
   "os"
)

//-----------------------------------------------------------------------------
func main() {
   idx := fmi.New(os.Args[1])
   idx.Save(os.Args[1])
   idx.Show()
}