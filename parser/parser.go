package parser

import (
	"fmt"

	"go-compiler/ast"
	"go-compiler/lexer"
)

// =====================================================================
// PARSER DESCENDENTE RECURSIVO
// =====================================================================
// Gramática:
//   Program      → "package" IDENT { Import } { FuncDecl | VarDecl }
//   FuncDecl     → "func" IDENT "(" ")" "{" Statement* "}"
//   Statement    → VarDecl | ShortDecl | Assignment | PrintStmt
//                | IfStmt | ForStmt | ReturnStmt | BreakStmt | ContinueStmt
//   Expr         → LogicalExpr
//   LogicalExpr  → RelationalExpr ( ( "&&" | "||" ) RelationalExpr )*
//   RelExpr      → SimpleExpr [ ( "==" | "!=" | "<" | ">" | "<=" | ">=" ) SimpleExpr ]
//   SimpleExpr   → Term ( ( "+" | "-" ) Term )*
//   Term         → Unary ( ( "*" | "/" | "%" ) Unary )*
//   Unary        → [ "!" | "-" ] Factor
//   Factor       → LITERAL | IDENT | "(" Expr ")"
// =====================================================================

type Parser struct {
	tokens []lexer.TokenData
	pos    int
	errors []error
}

func NewParser(tokens []lexer.TokenData) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) Errors() []error { return p.errors }

// ─── helpers ───────────────────────────────────────────────────────

func (p *Parser) peek() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.EOF
	}
	return p.tokens[p.pos].Type
}

func (p *Parser) peekToken() lexer.TokenData {
	if p.pos >= len(p.tokens) {
		return lexer.TokenData{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekLexeme() string {
	return p.peekToken().Lexeme
}

// lookahead retorna o tipo do token na posição pos+offset sem avançar.
func (p *Parser) lookahead(offset int) lexer.Token {
	i := p.pos + offset
	if i >= len(p.tokens) {
		return lexer.EOF
	}
	return p.tokens[i].Type
}

func (p *Parser) advance() lexer.TokenData {
	tok := p.peekToken()
	if tok.Type != lexer.EOF {
		p.pos++
	}
	return tok
}

// match consome o token se for do tipo esperado; registra erro caso contrário.
func (p *Parser) match(expected lexer.Token) (lexer.TokenData, error) {
	if p.peek() == expected {
		return p.advance(), nil
	}
	tok := p.peekToken()
	err := fmt.Errorf("erro sintatico linha %d: esperava %s, encontrou %s (%q)",
		tok.Line, expected, tok.Type, tok.Lexeme)
	p.errors = append(p.errors, err)
	return lexer.TokenData{}, err
}

// syncError registra erro e aplica Panic Mode: avança até ponto seguro.
func (p *Parser) syncError(msg string) {
	tok := p.peekToken()
	p.errors = append(p.errors, fmt.Errorf("erro sintatico linha %d: %s (encontrou %q)",
		tok.Line, msg, tok.Lexeme))
	for p.peek() != lexer.EOF &&
		p.peek() != lexer.SEMICOLON &&
		p.peek() != lexer.RBRACE {
		p.advance()
	}
	if p.peek() == lexer.SEMICOLON {
		p.advance()
	}
}

// skipSemicolons descarta ponto e vírgula soltos (inseridos automaticamente).
func (p *Parser) skipSemicolons() {
	for p.peek() == lexer.SEMICOLON {
		p.advance()
	}
}

// ─── Program ───────────────────────────────────────────────────────

func (p *Parser) ParseProgram() *ast.ProgramNode {
	prog := &ast.ProgramNode{}

	p.skipSemicolons()

	// package <nome>
	if p.peek() == lexer.PACKAGE {
		p.advance()
		if id, err := p.match(lexer.IDENT); err == nil {
			prog.PackageName = id.Lexeme
		}
		p.match(lexer.SEMICOLON)
	}

	// import "pkg"  ou  import ( "pkg1" "pkg2" )
	for p.peek() == lexer.IMPORT {
		p.advance()
		if p.peek() == lexer.LPAREN {
			p.advance()
			for p.peek() == lexer.STRING {
				imp := p.advance()
				prog.Imports = append(prog.Imports, imp.Lexeme)
				p.match(lexer.SEMICOLON)
			}
			p.match(lexer.RPAREN)
		} else {
			if imp, err := p.match(lexer.STRING); err == nil {
				prog.Imports = append(prog.Imports, imp.Lexeme)
			}
		}
		p.match(lexer.SEMICOLON)
	}

	// declarações de nível de pacote: func e var globais
	for p.peek() != lexer.EOF {
		p.skipSemicolons()
		if p.peek() == lexer.EOF {
			break
		}

		saved := p.pos
		switch p.peek() {
		case lexer.FUNC:
			if f := p.parseFunc(); f != nil {
				prog.Functions = append(prog.Functions, f)
			}
		case lexer.VAR:
			if v := p.parseVarDecl(); v != nil {
				prog.Globals = append(prog.Globals, v)
			}
		case lexer.TYPE, lexer.STRUCT, lexer.CONST, lexer.INTERFACE:
			p.syncError(fmt.Sprintf("'%s' nao suportado nesta versao do compilador", p.peekLexeme()))
		default:
			p.syncError("esperava 'func' ou 'var' no nivel do pacote")
		}
		// salvaguarda anti-loop infinito
		if p.pos == saved {
			p.advance()
		}
	}

	return prog
}

// ─── FuncDecl ──────────────────────────────────────────────────────

func (p *Parser) parseFunc() ast.ASTNode {
	p.advance() // consome 'func'

	id, err := p.match(lexer.IDENT)
	if err != nil {
		return nil
	}

	// lista de parâmetros — não parseamos tipos ainda, só consumimos
	p.match(lexer.LPAREN)
	for p.peek() != lexer.RPAREN && p.peek() != lexer.EOF {
		p.advance()
	}
	p.match(lexer.RPAREN)

	// tipo de retorno opcional (simples, sem parênteses)
	if p.peek() == lexer.IDENT ||
		p.peek() == lexer.MUL ||
		isTypeKeyword(p.peek()) {
		p.advance()
	}

	if _, err := p.match(lexer.LBRACE); err != nil {
		return nil
	}
	body := p.parseBlock()
	p.match(lexer.RBRACE)
	p.match(lexer.SEMICOLON)

	return &ast.FuncNode{Name: id.Lexeme, Body: body}
}

func isTypeKeyword(t lexer.Token) bool {
	_ = t
	return false
}

// ─── Block ─────────────────────────────────────────────────────────

func (p *Parser) parseBlock() []ast.ASTNode {
	var stmts []ast.ASTNode
	for p.peek() != lexer.RBRACE && p.peek() != lexer.EOF {
		saved := p.pos
		if s := p.parseStatement(); s != nil {
			stmts = append(stmts, s)
		}
		if p.pos == saved {
			p.advance() // salvaguarda anti-loop
		}
	}
	return stmts
}

// ─── Statement ─────────────────────────────────────────────────────

func (p *Parser) parseStatement() ast.ASTNode {
	p.skipSemicolons()

	if p.peek() == lexer.RBRACE || p.peek() == lexer.EOF {
		return nil
	}

	switch p.peek() {

	case lexer.VAR:
		return p.parseVarDecl()

	case lexer.IDENT:
		// fmt.Println(...)
		if p.peekLexeme() == "fmt" && p.lookahead(1) == lexer.PERIOD {
			return p.parsePrintStmt()
		}
		// x := expr
		if p.lookahead(1) == lexer.DEFINE {
			return p.parseShortDecl()
		}
		// x = expr
		if p.lookahead(1) == lexer.ASSIGN {
			return p.parseAssignment()
		}
		// x++ ou x--
		if p.lookahead(1) == lexer.INC || p.lookahead(1) == lexer.DEC {
			return p.parseIncDec()
		}
		// chamada de função genérica: ident(...)
		if p.lookahead(1) == lexer.LPAREN {
			return p.parseFuncCall()
		}
		p.syncError("instrucao invalida com identificador")
		return nil

	case lexer.IF:
		return p.parseIf()

	case lexer.FOR:
		return p.parseFor()

	case lexer.RETURN:
		return p.parseReturn()

	case lexer.BREAK:
		p.advance()
		p.match(lexer.SEMICOLON)
		return &ast.BreakNode{}

	case lexer.CONTINUE:
		p.advance()
		p.match(lexer.SEMICOLON)
		return &ast.ContinueNode{}

	case lexer.TYPE, lexer.STRUCT, lexer.MAP, lexer.CONST,
		lexer.SELECT, lexer.SWITCH, lexer.DEFER, lexer.RANGE,
		lexer.INTERFACE, lexer.GO:
		p.syncError(fmt.Sprintf("'%s' nao suportado nesta versao do compilador", p.peekLexeme()))
		return nil

	default:
		p.syncError("instrucao invalida")
		return nil
	}
}

// ─── VarDecl ───────────────────────────────────────────────────────

func (p *Parser) parseVarDecl() ast.ASTNode {
	p.advance() // consome 'var'

	id, err := p.match(lexer.IDENT)
	if err != nil {
		return nil
	}

	if p.peek() == lexer.IDENT {
		p.advance()
	}

	var init ast.ASTNode
	if p.peek() == lexer.ASSIGN {
		p.advance()
		init = p.parseExpr()
	}

	p.match(lexer.SEMICOLON)
	return &ast.VarDeclNode{Name: id.Lexeme, Initializer: init, Line: id.Line}
}

// ─── ShortDecl ─────────────────────────────────────────────────────

func (p *Parser) parseShortDecl() ast.ASTNode {
	id := p.advance() // consome IDENT
	p.advance()       // consome ':='
	init := p.parseExpr()
	p.match(lexer.SEMICOLON)
	return &ast.VarDeclNode{Name: id.Lexeme, Initializer: init, Line: id.Line}
}

// ─── Assignment ────────────────────────────────────────────────────

func (p *Parser) parseAssignment() ast.ASTNode {
	id := p.advance() // consome IDENT
	p.advance()       // consome '='
	expr := p.parseExpr()
	p.match(lexer.SEMICOLON)
	return &ast.AssignNode{Name: id.Lexeme, Expr: expr, Line: id.Line}
}

// ─── IncDec ────────────────────────────────────────────────────────

func (p *Parser) parseIncDec() ast.ASTNode {
	id := p.advance() // consome IDENT
	op := p.advance() // consome ++ ou --
	p.match(lexer.SEMICOLON)

	opStr := "+"
	if op.Type == lexer.DEC {
		opStr = "-"
	}
	return &ast.AssignNode{
		Name: id.Lexeme,
		Expr: &ast.BinaryOpNode{OpStr: opStr,
			Left:  &ast.VariableNode{Name: id.Lexeme, Line: id.Line},
			Right: &ast.LiteralNode{Value: "1"}},
		Line: id.Line,
	}
}

// ─── FuncCall ──────────────────────────────────────────────────────
// CORRIGIDO: Avanço incondicional para evitar loop infinito silencioso

func (p *Parser) parseFuncCall() ast.ASTNode {
	id := p.advance() // consome IDENT
	p.advance()       // consome '('
	depth := 1
	for depth > 0 && p.peek() != lexer.EOF {
		tok := p.advance() // Avança consumindo o token atual!
		if tok.Type == lexer.LPAREN {
			depth++
		} else if tok.Type == lexer.RPAREN {
			depth--
		}
	}
	// Não chamamos p.match(lexer.RPAREN) porque o advance() já o engoliu.
	p.match(lexer.SEMICOLON)
	return &ast.VariableNode{Name: id.Lexeme + "(...)", Line: id.Line}
}

// ─── PrintStmt ─────────────────────────────────────────────────────

func (p *Parser) parsePrintStmt() ast.ASTNode {
	p.advance()           // consome 'fmt'
	p.match(lexer.PERIOD) // consome '.'
	p.match(lexer.IDENT)  // consome 'Println'
	p.match(lexer.LPAREN)
	expr := p.parseExpr()
	p.match(lexer.RPAREN)
	p.match(lexer.SEMICOLON)
	return &ast.PrintCallNode{Expr: expr}
}

// ─── IfStmt ────────────────────────────────────────────────────────

func (p *Parser) parseIf() ast.ASTNode {
	p.advance() // consome 'if'

	var init ast.ASTNode
	var cond ast.ASTNode

	// Tenta ler a primeira parte (pode ser a inicialização ou já a condição)
	firstPt := p.parseSimpleStmt()

	if p.peek() == lexer.SEMICOLON {
		p.advance() // consome o ';'
		init = firstPt
		// Go permite que a condição seja lida diretamente após o ';' (sem inicialização)
		cond = p.parseExpr()
	} else {
		// Se não houver ';', a condição é lida diretamente após o 'if'
		cond = firstPt
	}

	if _, err := p.match(lexer.LBRACE); err != nil {
		return nil
	}
	thenBranch := p.parseBlock()
	p.match(lexer.RBRACE)

	var elseBranch []ast.ASTNode
	if p.peek() == lexer.ELSE {
		p.advance() // consome 'else'
		if p.peek() == lexer.IF {
			// else if
			elseBranch = append(elseBranch, p.parseIf())
		} else {
			// else
			if _, err := p.match(lexer.LBRACE); err != nil {
				return nil
			}
			elseBranch = p.parseBlock()
			p.match(lexer.RBRACE)
		}
	}

	// Retorna o nó montado
	return &ast.IfNode{Init: init, Condition: cond, ThenBranch: thenBranch, ElseBranch: elseBranch}
}

// ─── ForStmt ───────────────────────────────────────────────────────

func (p *Parser) parseFor() ast.ASTNode {
	p.advance() // consome 'for'

	if p.peek() == lexer.LBRACE {
		p.advance()
		body := p.parseBlock()
		p.match(lexer.RBRACE)
		return &ast.ForNode{Condition: nil, Body: body}
	}

	isClassic := false
	for i := p.pos; i < len(p.tokens); i++ {
		tt := p.tokens[i].Type
		if tt == lexer.LBRACE || tt == lexer.EOF {
			break
		}
		if tt == lexer.SEMICOLON {
			isClassic = true
			break
		}
	}

	node := &ast.ForNode{}

	if isClassic {
		init := p.parseSimpleStmt()
		node.Body = append([]ast.ASTNode{}, init)
		p.match(lexer.SEMICOLON)
		if p.peek() != lexer.SEMICOLON {
			node.Condition = p.parseExpr()
		}
		p.match(lexer.SEMICOLON)
		var post ast.ASTNode
		if p.peek() != lexer.LBRACE {
			post = p.parseSimpleStmt()
		}
		p.match(lexer.LBRACE)
		body := p.parseBlock()
		p.match(lexer.RBRACE)

		full := []ast.ASTNode{init}
		full = append(full, body...)
		if post != nil {
			full = append(full, post)
		}
		node.Body = full
	} else {
		node.Condition = p.parseExpr()
		p.match(lexer.LBRACE)
		node.Body = p.parseBlock()
		p.match(lexer.RBRACE)
	}

	return node
}

func (p *Parser) parseSimpleStmt() ast.ASTNode {
	if p.peek() == lexer.IDENT {
		if p.lookahead(1) == lexer.DEFINE {
			id := p.advance()
			p.advance()
			expr := p.parseExpr()
			return &ast.VarDeclNode{Name: id.Lexeme, Initializer: expr, Line: id.Line}
		}
		if p.lookahead(1) == lexer.ASSIGN {
			id := p.advance()
			p.advance()
			expr := p.parseExpr()
			return &ast.AssignNode{Name: id.Lexeme, Expr: expr, Line: id.Line}
		}
		if p.lookahead(1) == lexer.INC || p.lookahead(1) == lexer.DEC {
			id := p.advance()
			op := p.advance()
			opStr := "+"
			if op.Type == lexer.DEC {
				opStr = "-"
			}
			return &ast.AssignNode{
				Name: id.Lexeme,
				Expr: &ast.BinaryOpNode{OpStr: opStr,
					Left:  &ast.VariableNode{Name: id.Lexeme},
					Right: &ast.LiteralNode{Value: "1"}},
			}
		}
	}
	return p.parseExpr()
}

func (p *Parser) parseReturn() ast.ASTNode {
	p.advance()

	if p.peek() == lexer.SEMICOLON || p.peek() == lexer.RBRACE || p.peek() == lexer.EOF {
		p.match(lexer.SEMICOLON)
		return &ast.ReturnNode{Value: nil}
	}

	val := p.parseExpr()
	p.match(lexer.SEMICOLON)
	return &ast.ReturnNode{Value: val}
}

// =====================================================================
// CASCATA DE PRECEDÊNCIA CORRIGIDA (LOGICAL -> RELATIONAL -> MATH)
// =====================================================================

// Camada 1: Operadores Lógicos (&&, ||)
func (p *Parser) parseExpr() ast.ASTNode {
	return p.parseLogicalExpr()
}

func (p *Parser) parseLogicalExpr() ast.ASTNode {
	left := p.parseRelationalExpr()
	// Agora ele reconhece && e ||
	for p.peek() == lexer.LAND || p.peek() == lexer.LOR {
		op := p.advance()
		right := p.parseRelationalExpr()
		left = &ast.BinaryOpNode{OpStr: op.Lexeme,
			Left:  left,
			Right: right}
	}
	return left
}

// Camada 2: Operadores Relacionais (==, !=, <, >, <=, >=)
func (p *Parser) parseRelationalExpr() ast.ASTNode {
	left := p.parseSimpleExpr()
	switch p.peek() {
	case lexer.EQL, lexer.NEQ, lexer.LSS, lexer.GTR, lexer.LEQ, lexer.GEQ:
		op := p.advance()
		right := p.parseSimpleExpr()
		return &ast.BinaryOpNode{OpStr: op.Lexeme, Left: left, Right: right}
	}
	return left
}

// Camada 3: Soma e Subtração
func (p *Parser) parseSimpleExpr() ast.ASTNode {
	left := p.parseTerm()
	for p.peek() == lexer.ADD || p.peek() == lexer.SUB {
		op := p.advance()
		right := p.parseTerm()
		left = &ast.BinaryOpNode{OpStr: op.Lexeme, Left: left, Right: right}
	}
	return left
}

// Camada 4: Multiplicação, Divisão e Resto
func (p *Parser) parseTerm() ast.ASTNode {
	left := p.parseUnary()
	for p.peek() == lexer.MUL || p.peek() == lexer.QUO || p.peek() == lexer.REM {
		op := p.advance()
		right := p.parseUnary()
		left = &ast.BinaryOpNode{OpStr: op.Lexeme, Left: left, Right: right}
	}
	return left
}

// Camada 5: Unários (Negação Lógica e Numérica)
func (p *Parser) parseUnary() ast.ASTNode {
	switch p.peek() {
	case lexer.NOT: // O operador '!' agora cai perfeitamente aqui
		op := p.advance()
		return &ast.UnaryOpNode{OpStr: op.Lexeme, Operand: p.parseUnary()}
	case lexer.SUB:
		op := p.advance()
		return &ast.UnaryOpNode{OpStr: op.Lexeme, Operand: p.parseUnary()}
	}
	return p.parseFactor()
}

// Camada 6: Literais, Variáveis e Parênteses
func (p *Parser) parseFactor() ast.ASTNode {
	tok := p.peekToken()

	switch p.peek() {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.IMAG, lexer.CHAR:
		p.advance()
		return &ast.LiteralNode{Value: tok.Lexeme}

	case lexer.IDENT:
		if tok.Lexeme == "true" || tok.Lexeme == "false" {
			p.advance()
			return &ast.LiteralNode{Value: tok.Lexeme}
		}
		p.advance()
		if p.peek() == lexer.LPAREN {
			p.advance()
			depth := 1
			for depth > 0 && p.peek() != lexer.EOF {
				tokInterno := p.advance()
				if tokInterno.Type == lexer.LPAREN {
					depth++
				} else if tokInterno.Type == lexer.RPAREN {
					depth--
				}
			}
			return &ast.VariableNode{Name: tok.Lexeme + "(...)", Line: tok.Line}
		}
		return &ast.VariableNode{Name: tok.Lexeme, Line: tok.Line}

	case lexer.LPAREN:
		p.advance()
		expr := p.parseExpr()
		p.match(lexer.RPAREN)
		return expr

	default:
		p.syncError(fmt.Sprintf("fator invalido: %q", tok.Lexeme))
		return &ast.LiteralNode{Value: "0"}
	}
}
