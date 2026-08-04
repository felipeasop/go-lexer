package testdata

import "fmt"

// CASO: else na linha SEGUINTE ao "}"
// ERRO ESPERADO — tanto no compilador Go oficial quanto no nosso.
// Motivo: o scanner insere ";" após "}" porque T_RBRACE está em
// nextRequiresSemicolon(). A sequência vira:
//   } ;
//   else   ← parser não espera "else" após ";", erro sintático
//
// $ go build → syntax error: unexpected else, expected }
// Nosso compilador → erro sintatico: instrucao invalida (encontrou "else")

func brokenIfElse() {
	var x int = 10

	if x > 5 {
		fmt.Println(x)
	}
	else {
		fmt.Println(x)
	}
}
