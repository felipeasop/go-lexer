package semantic

import (
	"strconv"
	"strings"

	"go-compiler/ast"
)

// =====================================================================
// ANALISADOR SEMÂNTICO
// =====================================================================
// Percorre a AST produzida pelo Parser (padrão Visitor) e valida o
// programa segundo as regras semânticas da linguagem: uso de variáveis
// não declaradas, declaração duplicada, uso antes da inicialização,
// incompatibilidade de tipos, variáveis não utilizadas, divisão por
// zero detectável estaticamente e código morto após return/break/continue.
// =====================================================================

// SemanticAnalyzer percorre a AST e produz um Report com erros e avisos.
type SemanticAnalyzer struct {
	symbols *SymbolTable
	report  *Report

	// allDeclared guarda TODOS os símbolos já vistos (mesmo após o escopo
	// deles fechar) para a checagem final de "declarada e nunca utilizada"
	// e para exportação da Tabela de Símbolos completa em JSON.
	allDeclared []*Symbol
}

// NewSemanticAnalyzer cria um analisador pronto para uso.
func NewSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{
		symbols: NewSymbolTable(),
		report:  &Report{},
	}
}

// Analyze é o ponto de entrada da análise semântica. Retorna o relatório
// completo (erros + avisos) ao final do percurso da AST.
func (s *SemanticAnalyzer) Analyze(program *ast.ProgramNode) *Report {
	s.visitProgram(program)
	s.checkUnusedInScope(0) // variáveis globais não utilizadas
	return s.report
}

// Symbols retorna todos os símbolos vistos durante a análise (de todos
// os escopos, mesmo os já fechados). Usado pelo main para exportar a
// Tabela de Símbolos completa em JSON ao final da execução.
func (s *SemanticAnalyzer) Symbols() []*Symbol {
	return s.allDeclared
}

// ─── visitProgram ──────────────────────────────────────────────────

func (s *SemanticAnalyzer) visitProgram(n *ast.ProgramNode) {
	// Declarações globais (var no nível do pacote) entram no escopo 0.
	for _, g := range n.Globals {
		if vd, ok := g.(*ast.VarDeclNode); ok {
			s.visitDeclaration(vd)
		}
	}

	// Cada função ganha seu próprio escopo.
	for _, f := range n.Functions {
		if fn, ok := f.(*ast.FuncNode); ok {
			s.visitFunc(fn)
		}
	}
}

// visitFunc entra em um novo escopo para o corpo da função.
func (s *SemanticAnalyzer) visitFunc(n *ast.FuncNode) {
	s.symbols.EnterScope()
	level := s.symbols.currentScopeLevel()

	s.visitBlock(n.Body)

	s.checkUnusedInScope(level)
	s.symbols.ExitScope()
}

// ─── código morto ─────────────────────────────────────────────────

// visitBlock percorre uma sequência de statements de um mesmo bloco.
// Além de visitar cada um, verifica código morto: qualquer instrução
// que apareça depois de um return/break/continue no mesmo bloco nunca
// será alcançada em tempo de execução.
func (s *SemanticAnalyzer) visitBlock(stmts []ast.ASTNode) {
	terminated := false
	var lastTerminator string

	for _, stmt := range stmts {
		if terminated {
			if line := lineOf(stmt); line > 0 {
				s.report.AddWarning(line, "código inalcançável (instrução após '%s')", lastTerminator)
			}
			// Segue analisando o statement morto mesmo assim: se ele usar
			// uma variável inexistente, por exemplo, isso continua sendo
			// um erro real que vale a pena reportar.
		}

		s.visitStatement(stmt)

		if kind, ok := terminatorKind(stmt); ok {
			terminated = true
			lastTerminator = kind
		}
	}
}

// terminatorKind reporta se um statement encerra o fluxo do bloco
// (return, break, continue) e qual é a palavra-chave responsável.
func terminatorKind(node ast.ASTNode) (string, bool) {
	switch node.(type) {
	case *ast.ReturnNode:
		return "return", true
	case *ast.BreakNode:
		return "break", true
	case *ast.ContinueNode:
		return "continue", true
	}
	return "", false
}

// lineOf tenta extrair a linha de um nó, quando disponível. Nós que
// ainda não guardam linha na AST (ex.: PrintCallNode, BreakNode) usam
// o valor mais próximo disponível ou retornam 0 — nesse caso a checagem
// de código morto simplesmente não gera aviso para aquele nó específico.
func lineOf(node ast.ASTNode) int {
	switch n := node.(type) {
	case *ast.VarDeclNode:
		return n.Line
	case *ast.AssignNode:
		return n.Line
	case *ast.VariableNode:
		return n.Line
	case *ast.ReturnNode:
		if n.Value != nil {
			return lineOf(n.Value)
		}
	}
	return 0
}

// ─── dispatch genérico de statement ─────────────────────────────────

func (s *SemanticAnalyzer) visitStatement(node ast.ASTNode) {
	switch n := node.(type) {
	case *ast.VarDeclNode:
		s.visitDeclaration(n)
	case *ast.AssignNode:
		s.visitAssignment(n)
	case *ast.PrintCallNode:
		s.visitPrint(n)
	case *ast.IfNode:
		s.visitIf(n)
	case *ast.ForNode:
		s.visitWhile(n)
	case *ast.ReturnNode:
		if n.Value != nil {
			s.visitExpression(n.Value)
		}
	case *ast.BreakNode, *ast.ContinueNode:
		// nada a validar semanticamente
	default:
		// VariableNode "solto" (chamada de função genérica ignorada
		// no parser), BinaryOpNode etc. — trata como expressão.
		if node != nil {
			s.visitExpression(node)
		}
	}
}

// ─── 1. Declaração de variável ──────────────────────────────────────
// Verificação obrigatória #2 (declaração duplicada) e infraestrutura
// para #3 (uso antes da inicialização).

func (s *SemanticAnalyzer) visitDeclaration(n *ast.VarDeclNode) {
	initialized := n.Initializer != nil
	inferredType := TypeUnknown

	if n.Initializer != nil {
		inferredType = s.visitExpression(n.Initializer)
	}

	if err := s.symbols.Declare(n.Name, inferredType, n.Line, initialized); err != nil {
		// Verificação #2: Declaração Duplicada
		s.report.AddError(n.Line, "variável '%s' já declarada", n.Name)
		return
	}

	sym, _ := s.symbols.Lookup(n.Name)
	s.allDeclared = append(s.allDeclared, sym)
}

// ─── 2. Atribuição ───────────────────────────────────────────────────
// Verificação obrigatória #1 (variável não declarada) e #4 (tipos).

func (s *SemanticAnalyzer) visitAssignment(n *ast.AssignNode) {
	sym, found := s.symbols.Lookup(n.Name)
	if !found {
		// Verificação #1: Variável Declarada
		s.report.AddError(n.Line, "variável '%s' não declarada", n.Name)
		// Ainda assim analisamos o lado direito para não perder outros erros.
		s.visitExpression(n.Expr)
		return
	}

	exprType := s.visitExpression(n.Expr)

	// Verificação #4: Verificação de Tipos
	if sym.Type != TypeUnknown && exprType != TypeUnknown && sym.Type != exprType {
		s.report.AddError(n.Line, "atribuição incompatível: variável '%s' é do tipo '%s', mas recebeu valor do tipo '%s'",
			n.Name, sym.Type, exprType)
	} else if sym.Type == TypeUnknown {
		// Primeira atribuição de fato define o tipo (caso de 'var x' sem tipo/valor).
		sym.Type = exprType
	}

	sym.Initialized = true
}

// ─── 3. print (fmt.Println) ─────────────────────────────────────────

func (s *SemanticAnalyzer) visitPrint(n *ast.PrintCallNode) {
	s.visitExpression(n.Expr)
}

// ─── 4. if / else ────────────────────────────────────────────────────

func (s *SemanticAnalyzer) visitIf(n *ast.IfNode) {
	s.symbols.EnterScope()
	level := s.symbols.currentScopeLevel()

	if n.Init != nil {
		s.visitStatement(n.Init)
	}
	if n.Condition != nil {
		s.visitExpression(n.Condition)
	}
	s.visitBlock(n.ThenBranch)

	s.checkUnusedInScope(level)
	s.symbols.ExitScope()

	if len(n.ElseBranch) > 0 {
		s.symbols.EnterScope()
		elseLevel := s.symbols.currentScopeLevel()
		s.visitBlock(n.ElseBranch)
		s.checkUnusedInScope(elseLevel)
		s.symbols.ExitScope()
	}
}

// ─── 5. for (while / clássico / infinito) ───────────────────────────

func (s *SemanticAnalyzer) visitWhile(n *ast.ForNode) {
	s.symbols.EnterScope()
	level := s.symbols.currentScopeLevel()

	if n.Condition != nil {
		s.visitExpression(n.Condition)
	}
	s.visitBlock(n.Body)

	s.checkUnusedInScope(level)
	s.symbols.ExitScope()
}

// ─── 6. Expressões ───────────────────────────────────────────────────
// Percorre a expressão, verifica uso de variáveis (declaradas? já
// inicializadas?) e retorna o tipo resultante para checagem de tipos
// no nó pai (ex.: atribuição, condição de if/for).

func (s *SemanticAnalyzer) visitExpression(node ast.ASTNode) SymbolType {
	switch n := node.(type) {

	case *ast.LiteralNode:
		return literalType(n.Value)

	case *ast.VariableNode:
		sym, found := s.symbols.Lookup(n.Name)
		if !found {
			// chamadas de função genéricas viram VariableNode com "(...)"
			// no nome — não são variáveis reais, então ignoramos aqui.
			if strings.HasSuffix(n.Name, "(...)") {
				return TypeUnknown
			}
			// Verificação #1: Variável Declarada
			s.report.AddError(n.Line, "variável '%s' não declarada", n.Name)
			return TypeUnknown
		}

		// Verificação #3: Uso Antes da Inicialização
		if !sym.Initialized {
			s.report.AddWarning(n.Line, "variável '%s' utilizada antes de receber valor", n.Name)
		}
		sym.Used = true
		return sym.Type

	case *ast.BinaryOpNode:
		leftType := s.visitExpression(n.Left)
		rightType := s.visitExpression(n.Right)

		// Divisão por zero detectável em tempo de compilação: só pegamos
		// o caso em que o divisor é um literal numérico "0" escrito
		// diretamente na expressão — divisão por uma variável cujo valor
		// só é conhecido em tempo de execução foge do escopo da análise
		// estática (exigiria interpretação/execução simbólica).
		if n.OpStr == "/" || n.OpStr == "%" {
			if lit, ok := n.Right.(*ast.LiteralNode); ok && isZeroLiteral(lit.Value) {
				s.report.AddError(lineOf(n.Left), "divisão por zero")
			}
		}

		return combineTypes(leftType, rightType)

	case *ast.UnaryOpNode:
		return s.visitExpression(n.Operand)

	default:
		return TypeUnknown
	}
}

// ─── Auxiliares de símbolo/tabela ────────────────────────────────────

// declareSymbol expõe a declaração de símbolo diretamente (uso por
// outros pontos do analisador, ex.: parâmetros de função no futuro).
func (s *SemanticAnalyzer) declareSymbol(name string, typ SymbolType, line int, initialized bool) error {
	err := s.symbols.Declare(name, typ, line, initialized)
	if err == nil {
		sym, _ := s.symbols.Lookup(name)
		s.allDeclared = append(s.allDeclared, sym)
	}
	return err
}

// lookupSymbol expõe a busca de símbolo diretamente.
func (s *SemanticAnalyzer) lookupSymbol(name string) (*Symbol, bool) {
	return s.symbols.Lookup(name)
}

// reportError expõe o registro de erro/aviso de forma centralizada.
func (s *SemanticAnalyzer) reportError(line int, msg string) {
	s.report.AddError(line, "%s", msg)
}

// checkUnusedInScope varre os símbolos de um escopo específico e emite
// o aviso de "declarada mas nunca utilizada" (Verificação #5) para os
// que não foram lidos em nenhuma expressão.
func (s *SemanticAnalyzer) checkUnusedInScope(level int) {
	for _, sym := range s.symbols.AllSymbolsInScope(level) {
		if !sym.Used {
			s.report.AddWarning(sym.DeclLine, "variável '%s' declarada mas nunca utilizada", sym.Name)
		}
	}
}

// ─── Inferência de tipos a partir de literais ────────────────────────

// literalType infere o SymbolType de um LiteralNode a partir do próprio
// lexema salvo pelo parser (a AST atual não guarda o Token original,
// então a inferência é feita pelo formato do valor).
func literalType(value string) SymbolType {
	if value == "true" || value == "false" {
		return TypeBool
	}
	if len(value) >= 2 && (value[0] == '"' || value[0] == '`') {
		return TypeString
	}
	if strings.Contains(value, ".") {
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return TypeFloat
		}
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return TypeInt
	}
	return TypeUnknown
}

// isZeroLiteral reconhece "0", "0.0" e variações equivalentes escritas
// como literal direto no código-fonte.
func isZeroLiteral(value string) bool {
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f == 0
	}
	return false
}

// combineTypes resolve o tipo resultante de uma operação binária.
// Regra simplificada: se os dois lados forem iguais, mantém o tipo;
// caso contrário (ou desconhecido), resulta em TypeUnknown para não
// gerar falso-positivo de incompatibilidade em cascata.
func combineTypes(left, right SymbolType) SymbolType {
	if left == right {
		return left
	}
	return TypeUnknown
}
