
package main

import (
   "bytes"
   "fmt"
   "io/ioutil"
   "encoding/gob"
   "encoding/binary"
   "log"
   "time"
   "compress/gzip"
)

func main() {
   N := int64(1<<22)
   data := make([]int64, N)
   for i:=int64(0); i<N; i++ {
      data[i] = i
   }
   fmt.Printf("%d %.2f %.2f\n",N, float32(N*8/(1<<10)), float32(N*8.0)/(1<<20))

   start := time.Now()
   fmt.Println("GOB")
   buffer1 := new(bytes.Buffer)
   enc := gob.NewEncoder(buffer1)
   err := enc.Encode(data)
   if err != nil {
      log.Fatal(err)
   }
   ioutil.WriteFile("data_gob", buffer1.Bytes(), 0600)
   elapsed := time.Since(start)
   fmt.Println("time", elapsed)

   start = time.Now()
   fmt.Println("BIN")
   buffer2 := new(bytes.Buffer)
   err = binary.Write(buffer2, binary.LittleEndian, data)
   if err != nil {
      log.Fatal(err)
   }
   ioutil.WriteFile("data_bin", buffer2.Bytes(), 0600)
   elapsed = time.Since(start)
   fmt.Println("time", elapsed)


   start = time.Now()
   fmt.Println("BIN-gzip")
   buffer3 := new(bytes.Buffer)
   w := gzip.NewWriter(buffer3)
   w.Write(buffer2.Bytes())
   w.Close()
   ioutil.WriteFile("data_bin.gz", buffer3.Bytes(), 0666)
   elapsed = time.Since(start)
   fmt.Println("time", elapsed)

}