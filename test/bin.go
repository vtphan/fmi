package main

import (
   "bytes"
   "encoding/binary"
   "fmt"
   "math"
)

type V struct {
   A int8
   B float64
}

func main() {
   v := &V{1,2.3}
   fmt.Println(v)

   buf := new(bytes.Buffer)
   var pi float64 = math.Pi
   var err error
   err = binary.Write(buf, binary.LittleEndian, pi)
   if err != nil {
      fmt.Println("binary.Write failed:", err)
   }
   err = binary.Write(buf, binary.LittleEndian, v.A)
   if err != nil {
      fmt.Println("binary.Write failed here:", err)
   }
   fmt.Printf("% x", buf.Bytes())
}