package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go-compiler/lexer"
	"go-compiler/parser"
	"go-compiler/semantic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso correto: go run . <caminho_do_arquivo>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler o arquivo %s: %v\n", filePath, err)
		os.Exit(1)
	}

	os.MkdirAll("outputs", 0755)

	fmt.Printf("=== INICIANDO ANÁLISE LÉXICA DO ARQUIVO: %s ===\n", filePath)
	scanner := lexer.NewScanner(string(data))
	tokens, err := scanner.Tokenize()
	if err != nil {
		fmt.Printf("Erro lexico fatal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-20s %-20s %s\n", "TIPO DE TOKEN", "LEXEMA", "LINHA")
	fmt.Println(strings.Repeat("-", 50))
	for _, tok := range tokens {
		if tok.Type != lexer.EOF {
			fmt.Printf("%-20s %-20s %d\n", tok.Type.String(), tok.Lexeme, tok.Line)
		}
	}

	tokJSON, _ := json.MarshalIndent(tokens, "", "  ")
	os.WriteFile("outputs/tokens.json", tokJSON, 0644)

	fmt.Println("\n=== INICIANDO ANÁLISE SINTÁTICA ===")
	p := parser.NewParser(tokens)
	ast := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Erros sintáticos:")
		for _, e := range p.Errors() {
			fmt.Printf("  - %v\n", e)
		}
	} else {
		fmt.Println("Análise sintática concluída sem erros!")
		fmt.Println("\n=== AST ===")
		ast.Print(0)

		astJSON := ast.ToJSON(0)
		os.WriteFile("outputs/ast.json", []byte(astJSON), 0644)

		fmt.Println("\n=== INICIANDO ANÁLISE SEMÂNTICA ===")
		analyzer := semantic.NewSemanticAnalyzer()
		report := analyzer.Analyze(ast)

		fmt.Println(report.String())
		os.WriteFile("outputs/relatorio_semantico.txt", []byte(report.String()), 0644)

		symbolsJSON := semantic.SymbolsToJSON(analyzer.Symbols(), 0)
		os.WriteFile("outputs/tabela_simbolos.json", []byte(symbolsJSON), 0644)

		if report.HasErrors() {
			os.Exit(1)
		}
	}
}
