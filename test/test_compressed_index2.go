package main

import (
	"fmt"
	"github.com/vtphan/fmi"
	"os"
	"runtime"
	"math/rand"
)

//-----------------------------------------------------------------------------
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) != 2 {
		panic("Usage: go run build_index.go file.fasta")
	}
	fmt.Println("======BUILDING INDEX")
	idx := fmi.CompressedIndex(os.Args[1], 20)
	idx.Show()
	idx.Check()

	fmt.Println("======SAVING INDEX")
	idx.SaveCompressedIndex()

	fmt.Println("======RELOADING INDEX")
	saved_idx := fmi.LoadCompressedIndex(os.Args[1] + ".fmi")
	saved_idx.Show()
	saved_idx.Check()

	fmt.Println("======TEST SEARCH")
	uncompressed_idx := fmi.New(os.Args[1])
	var x,y,z,x1,y1,z1 int
	for i:=0; i<500000; i++ {
		a := rand.Int63n(saved_idx.LEN)
		b := rand.Int63n(saved_idx.LEN)
		if a!=b {
			if a > b {
				a, b = b, a
			}
			// fmt.Printf("%d %d %d ", i, a, b)
			seq := fmi.SEQ[a:b]
			x,y,z = saved_idx.Search(seq)
			x1,y1,z1 = uncompressed_idx.Search(seq)
			// fmt.Println(x,y,z, x==x1, y==y1, z==z1)
			if x!=x1 || y!=y1 || z!=z1 {
				fmt.Println(i, a, b, x,y,z, x1,y1,z1)
				panic("Something is wrong")
			}
			if i%100000 == 0 {
				fmt.Println("finish testing", i, "random substring searches.")
			}
		}
	}
}
