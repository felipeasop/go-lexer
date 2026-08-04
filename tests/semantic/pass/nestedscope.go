package tests

import "fmt"

func nestedScope() {
	var total int = 0
	fmt.Println(total)

	if total == 0 {
		var total int = 99
		fmt.Println(total)
	}

	for total < 3 {
		var passo int = 1
		total = total + passo
		fmt.Println(total)
	}
}
