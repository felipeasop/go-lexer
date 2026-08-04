package tests

import "fmt"

// Divisão por zero com divisor literal, detectável estaticamente.
func divZero() {
	var x int = 10
	var y int = x / 0
	fmt.Println(y)
}
