// Copyright 2013 Vinhthuy Phan
// Package implements uncompressed FM index
//
package main

import (
   "github.com/vtphan/fmi"
   "fmt"
   "os"
)

var Debug bool

//-----------------------------------------------------------------------------
func main() {
   idx := fmi.Load(os.Args[1])
   fmt.Println(idx.LEN)
}