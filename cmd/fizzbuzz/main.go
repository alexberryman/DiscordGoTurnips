package main

import (
	"fmt"
	"math"
)

func main() {
	for k := 1; k <= 100; k++ {
		var s string
		if math.Mod(float64(k), float64(3)) == 0 {
			s += "Fizz"
		}
		if math.Mod(float64(k), float64(5)) == 0 {
			s += "Buzz"
		}

		if s == "" {
			s += fmt.Sprint(k)
		}
		fmt.Println(s)
	}
}
