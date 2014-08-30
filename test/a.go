package main

import (
   "fmt"
)

func f(x []int) []int {
   // var y []int
   y := make([]int, 2)
   copy(y, x[1:3])
   return y
}

func main() {
   read := []int{1,2,3,4,5}
   snp := f(read)
   fmt.Println(read, snp)
   snp[0] = 10
   fmt.Println(read, snp)
}