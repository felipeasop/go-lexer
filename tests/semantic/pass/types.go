package tests

import "fmt"

func typesOk() {
	var idade int = 25
	var altura float64 = 1.75
	var nome string = "MicroC"

	idade = 26
	altura = 1.80
	nome = "MicroGo"

	fmt.Println(idade)
	fmt.Println(altura)
	fmt.Println(nome)
}
