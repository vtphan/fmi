package main

import (
   "fmt"
   "bufio"
   "os"
)

func main() {
   f, err := os.Open("t.txt")
   if err != nil {
        panic("Error opening file")
   }
   defer f.Close()
   data := bufio.NewReader(f)
   var line, a, b []byte
   i := 0
   for {
      line, err = data.ReadBytes('\n') //ignore 1st line in input FASTQ file
      if err != nil {
         break
      }
      if i == 0 {
         a = line[1:3]
      } else if i == 1 {
         b = line[2:5]
      }
      fmt.Printf("i=%d %s a=%s, b=%s\n", i, string(line), a, b)
      i++
   }
}