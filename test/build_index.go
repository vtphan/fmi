package main

import (
	// "fmt"
	"github.com/vtphan/fmi"
	"log"
	"os"
	"runtime"
)

var memstats = new(runtime.MemStats)

func show_memstat(mesg string) {
	runtime.ReadMemStats(memstats)
	log.Printf("%s:\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f", mesg,
		float64(memstats.Alloc)/float64(1<<10),
		float64(memstats.TotalAlloc)/float64(1<<10),
		float64(memstats.Sys)/float64(1<<10),
		float64(memstats.HeapAlloc)/float64(1<<10),
		float64(memstats.HeapSys)/float64(1<<10))
}

//-----------------------------------------------------------------------------
func main() {
	if len(os.Args) != 2 {
		panic("Usage: go run build_index.go file.fasta")
	}
	// runtime.GOMAXPROCS(runtime.NumCPU())
	// show_memstat("before")
	idx := fmi.New(os.Args[1])
	idx.Save(os.Args[1])
	// idx.Show()
	show_memstat("after")
	idx.Show()
	idx.Check()
	// show_memstat("after")
}
