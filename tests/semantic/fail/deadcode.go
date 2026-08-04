package tests

import "fmt"

// Instruções após 'return' nunca são alcançadas. A atribuição 'x = 2'
// é reportada (AssignNode guarda linha); o Println logo em seguida não
// gera um segundo aviso porque PrintCallNode ainda não guarda número de
// linha na AST — limitação conhecida, não um erro de análise.
func deadCode() {
	var x int = 1
	return
	x = 2
	fmt.Println(x)
}
