package ast

import (
	"fmt"
	"strings"
)

// ─── FuncNode ──────────────────────────────────────────────────────
type FuncNode struct {
	Name string
	Body []ASTNode
}

func (n *FuncNode) Print(indent int) {
	fmt.Printf("%sFuncNode (func %s)\n", pad(indent), n.Name)
	for _, s := range n.Body {
		s.Print(indent + 2)
	}
}

func (n *FuncNode) ToJSON(indent int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s{\n%s\"tipo\": \"FuncNode\",\n%s\"nome\": %q,\n%s\"corpo\": [\n", jpd(indent), jpd(indent+1), jpd(indent+1), n.Name, jpd(indent+1)))
	for i, s := range n.Body {
		sb.WriteString(s.ToJSON(indent + 2))
		if i < len(n.Body)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("%s]\n%s}", jpd(indent+1), jpd(indent)))
	return sb.String()
}

// ─── VarDeclNode ───────────────────────────────────────────────────
type VarDeclNode struct {
	Name        string
	Initializer ASTNode
	Line        int // usado pelo Semantic Analyzer no relatório de erros
}

func (n *VarDeclNode) Print(indent int) {
	fmt.Printf("%sVarDeclNode (%s)\n", pad(indent), n.Name)
	if n.Initializer != nil {
		n.Initializer.Print(indent + 4)
	}
}

func (n *VarDeclNode) ToJSON(indent int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s{\n%s\"tipo\": \"VarDeclNode\",\n%s\"nome\": %q", jpd(indent), jpd(indent+1), jpd(indent+1), n.Name))
	if n.Initializer != nil {
		sb.WriteString(fmt.Sprintf(",\n%s\"inicializador\":\n%s", jpd(indent+1), n.Initializer.ToJSON(indent+2)))
	}
	sb.WriteString(fmt.Sprintf("\n%s}", jpd(indent)))
	return sb.String()
}

// ─── AssignNode ────────────────────────────────────────────────────
type AssignNode struct {
	Name string
	Expr ASTNode
	Line int // usado pelo Semantic Analyzer no relatório de erros
}

func (n *AssignNode) Print(indent int) {
	fmt.Printf("%sAssignNode (%s)\n", pad(indent), n.Name)
	n.Expr.Print(indent + 4)
}

func (n *AssignNode) ToJSON(indent int) string {
	return fmt.Sprintf("%s{\n%s\"tipo\": \"AssignNode\",\n%s\"nome\": %q,\n%s\"expr\":\n%s\n%s}",
		jpd(indent), jpd(indent+1), jpd(indent+1), n.Name, jpd(indent+1), n.Expr.ToJSON(indent+2), jpd(indent))
}

// ─── PrintCallNode ─────────────────────────────────────────────────
// fmt.Println(expr)
type PrintCallNode struct {
	Expr ASTNode
}

func (n *PrintCallNode) Print(indent int) {
	fmt.Printf("%sPrintCallNode (fmt.Println)\n", pad(indent))
	n.Expr.Print(indent + 4)
}

func (n *PrintCallNode) ToJSON(indent int) string {
	return fmt.Sprintf("%s{\n%s\"tipo\": \"PrintCallNode\",\n%s\"expr\":\n%s\n%s}",
		jpd(indent), jpd(indent+1), jpd(indent+1), n.Expr.ToJSON(indent+2), jpd(indent))
}

// ─── IfNode ────────────────────────────────────────────────────────
// else if é representado colocando um IfNode dentro de ElseBranch.
type IfNode struct {
	Init       ASTNode
	Condition  ASTNode
	ThenBranch []ASTNode
	ElseBranch []ASTNode
}

func (n *IfNode) Print(indent int) {
	fmt.Printf("%sIfNode\n", pad(indent))
	fmt.Printf("%sCondicao:\n", pad(indent+2))
	n.Condition.Print(indent + 4)
	fmt.Printf("%sThen:\n", pad(indent+2))
	for _, s := range n.ThenBranch {
		s.Print(indent + 4)
	}
	if len(n.ElseBranch) > 0 {
		fmt.Printf("%sElse:\n", pad(indent+2))
		for _, s := range n.ElseBranch {
			s.Print(indent + 4)
		}
	}
}

func (n *IfNode) ToJSON(indent int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s{\n%s\"tipo\": \"IfNode\",\n%s\"condicao\":\n%s,\n%s\"then\": [\n",
		jpd(indent), jpd(indent+1), jpd(indent+1), n.Condition.ToJSON(indent+2), jpd(indent+1)))
	for i, s := range n.ThenBranch {
		sb.WriteString(s.ToJSON(indent + 2))
		if i < len(n.ThenBranch)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("%s]", jpd(indent+1)))
	if len(n.ElseBranch) > 0 {
		sb.WriteString(fmt.Sprintf(",\n%s\"else\": [\n", jpd(indent+1)))
		for i, s := range n.ElseBranch {
			sb.WriteString(s.ToJSON(indent + 2))
			if i < len(n.ElseBranch)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("%s]", jpd(indent+1)))
	}
	sb.WriteString(fmt.Sprintf("\n%s}", jpd(indent)))
	return sb.String()
}

// ─── ForNode ───────────────────────────────────────────────────────
type ForNode struct {
	Condition ASTNode
	Body      []ASTNode
}

func (n *ForNode) Print(indent int) {
	fmt.Printf("%sForNode\n", pad(indent))
	fmt.Printf("%sCondicao:\n", pad(indent+2))
	if n.Condition != nil {
		n.Condition.Print(indent + 4)
	} else {
		fmt.Printf("%s(loop infinito)\n", pad(indent+4))
	}
	fmt.Printf("%sCorpo:\n", pad(indent+2))
	for _, s := range n.Body {
		s.Print(indent + 4)
	}
}

func (n *ForNode) ToJSON(indent int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s{\n%s\"tipo\": \"ForNode\",\n", jpd(indent), jpd(indent+1)))
	if n.Condition != nil {
		sb.WriteString(fmt.Sprintf("%s\"condicao\":\n%s,\n", jpd(indent+1), n.Condition.ToJSON(indent+2)))
	} else {
		sb.WriteString(fmt.Sprintf("%s\"condicao\": null,\n", jpd(indent+1)))
	}
	sb.WriteString(fmt.Sprintf("%s\"corpo\": [\n", jpd(indent+1)))
	for i, s := range n.Body {
		sb.WriteString(s.ToJSON(indent + 2))
		if i < len(n.Body)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("%s]\n%s}", jpd(indent+1), jpd(indent)))
	return sb.String()
}

// ─── ReturnNode ────────────────────────────────────────────────────
// return [expr]
type ReturnNode struct {
	Value ASTNode // nil para return sem valor
}

func (n *ReturnNode) Print(indent int) {
	fmt.Printf("%sReturnNode\n", pad(indent))
	if n.Value != nil {
		n.Value.Print(indent + 4)
	}
}

func (n *ReturnNode) ToJSON(indent int) string {
	if n.Value == nil {
		return fmt.Sprintf("%s{\"tipo\": \"ReturnNode\", \"valor\": null}", jpd(indent))
	}
	return fmt.Sprintf("%s{\n%s\"tipo\": \"ReturnNode\",\n%s\"valor\":\n%s\n%s}",
		jpd(indent), jpd(indent+1), jpd(indent+1), n.Value.ToJSON(indent+2), jpd(indent))
}

// ─── BreakNode ─────────────────────────────────────────────────────
type BreakNode struct{}

func (n *BreakNode) Print(indent int) {
	fmt.Printf("%sBreakNode\n", pad(indent))
}
func (n *BreakNode) ToJSON(indent int) string {
	return fmt.Sprintf("%s{\"tipo\": \"BreakNode\"}", jpd(indent))
}

// ─── ContinueNode ──────────────────────────────────────────────────
type ContinueNode struct{}

func (n *ContinueNode) Print(indent int) {
	fmt.Printf("%sContinueNode\n", pad(indent))
}
func (n *ContinueNode) ToJSON(indent int) string {
	return fmt.Sprintf("%s{\"tipo\": \"ContinueNode\"}", jpd(indent))
}
