package semantic

import (
	"fmt"
	"strings"
)

// =====================================================================
// RELATÓRIO DE ERROS SEMÂNTICOS
// =====================================================================

// Severity indica se a ocorrência impede a compilação (erro) ou é apenas
// informativa (aviso).
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityWarning {
		return "Aviso Semântico"
	}
	return "Erro Semântico"
}

// SemanticIssue é uma ocorrência (erro ou aviso) encontrada durante a análise.
type SemanticIssue struct {
	Severity    Severity
	Line        int
	Description string
}

func (e *SemanticIssue) Error() string {
	return fmt.Sprintf("Linha %d: %s: %s", e.Line, e.Severity, e.Description)
}

// Report agrega todos os erros e avisos encontrados pelo SemanticAnalyzer.
type Report struct {
	Issues []*SemanticIssue
}

// AddError registra um erro semântico (impede considerar o programa válido).
func (r *Report) AddError(line int, format string, args ...interface{}) {
	r.Issues = append(r.Issues, &SemanticIssue{
		Severity:    SeverityError,
		Line:        line,
		Description: fmt.Sprintf(format, args...),
	})
}

// AddWarning registra um aviso semântico (não impede a validade do programa).
func (r *Report) AddWarning(line int, format string, args ...interface{}) {
	r.Issues = append(r.Issues, &SemanticIssue{
		Severity:    SeverityWarning,
		Line:        line,
		Description: fmt.Sprintf(format, args...),
	})
}

// Errors retorna apenas as ocorrências de severidade Erro.
func (r *Report) Errors() []*SemanticIssue {
	var out []*SemanticIssue
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			out = append(out, i)
		}
	}
	return out
}

// Warnings retorna apenas as ocorrências de severidade Aviso.
func (r *Report) Warnings() []*SemanticIssue {
	var out []*SemanticIssue
	for _, i := range r.Issues {
		if i.Severity == SeverityWarning {
			out = append(out, i)
		}
	}
	return out
}

// HasErrors indica se o programa possui pelo menos um erro semântico.
func (r *Report) HasErrors() bool {
	return len(r.Errors()) > 0
}

// String monta o relatório textual final: tipo do erro, linha, descrição
// e contagem total, conforme pedido no enunciado.
func (r *Report) String() string {
	var sb strings.Builder

	if len(r.Issues) == 0 {
		sb.WriteString("Análise Semântica concluída com sucesso.\n")
		return sb.String()
	}

	for _, issue := range r.Issues {
		sb.WriteString(fmt.Sprintf("Linha %d:\n", issue.Line))
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", issue.Severity, issue.Description))
	}

	sb.WriteString(fmt.Sprintf("Total de erros: %d\n", len(r.Errors())))
	sb.WriteString(fmt.Sprintf("Total de avisos: %d\n", len(r.Warnings())))

	if r.HasErrors() {
		sb.WriteString("\nAnálise Semântica concluída com erros.\n")
	} else {
		sb.WriteString("\nAnálise Semântica concluída com sucesso (com avisos).\n")
	}

	return sb.String()
}

// SymbolsToJSON exporta a Tabela de Símbolos em formato JSON (melhoria obrigatória).
func SymbolsToJSON(symbols []*Symbol, indent int) string {
	pad := strings.Repeat("  ", indent)
	pad1 := strings.Repeat("  ", indent+1)
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, s := range symbols {
		sb.WriteString(fmt.Sprintf(
			"%s{\"nome\": %q, \"tipo\": %q, \"escopo\": %d, \"linha\": %d, \"inicializada\": %t, \"utilizada\": %t}",
			pad1, s.Name, s.Type.String(), s.Scope, s.DeclLine, s.Initialized, s.Used))
		if i < len(symbols)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(pad + "]")
	return sb.String()
}
