package tests

import "fmt"

func uninitialized() {
	var x int
	fmt.Println(x)
}
