package main

import (
	// "fmt"
	"github.com/vtphan/fmi"
	"os"
)

//-----------------------------------------------------------------------------
func main() {
	if len(os.Args) != 2 {
		panic("Usage: go run build_index.go file.fasta")
	}
	idx := fmi.CompressedIndex(os.Args[1], 16)
	// idx.Save(os.Args[1])
	idx.Show()
	idx.Check()
	idx.SaveCompressedIndex()


	uncompressed_idx := fmi.New(os.Args[1])
	uncompressed_idx.Show()
	uncompressed_idx.Check()
}
