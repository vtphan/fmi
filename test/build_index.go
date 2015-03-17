package main

import (
   "github.com/vtphan/fmi"
   "os"
   "runtime"
   "log"
   // "fmt"
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
   runtime.GOMAXPROCS(runtime.NumCPU())
   show_memstat("before")
   idx := fmi.New(os.Args[1])
   idx.Save(os.Args[1])
   // for i:=0; i<len(idx.SA); i++ {
   //    fmt.Println(i, idx.SA[i])
   // }
   idx.Show()
   show_memstat("after")
}