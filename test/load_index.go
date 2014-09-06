package main

import (
   "github.com/vtphan/fmi"
   "os"
   "fmt"
   "runtime"
   "log"
)


var memstats = new(runtime.MemStats)

func show_memstat(mesg string) {
   runtime.ReadMemStats(memstats)
   log.Printf("%s:\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f", mesg,
      float64(memstats.Alloc)/float64(1<<20),
      float64(memstats.TotalAlloc)/float64(1<<20),
      float64(memstats.Sys)/float64(1<<20),
      float64(memstats.HeapAlloc)/float64(1<<20),
      float64(memstats.HeapSys)/float64(1<<20))
}

//-----------------------------------------------------------------------------
func main() {
   runtime.GOMAXPROCS(runtime.NumCPU())
   show_memstat("before")
   idx := fmi.Load(os.Args[1])
   fmt.Println(idx.LEN)
   // idx.Show()
   show_memstat("after")
}