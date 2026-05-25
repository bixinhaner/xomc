// Package expr 提供 KPI 算术公式解析 / 求值 / 标识符提取。
//
// 拆到独立包是为了打破 pm/kpi → pm/kpi/router → pm/indicator → pm/kpi 这条
// 循环依赖（pm/indicator/formula_validator 也用 ParseFormula）。
// 本包零外部依赖，stdlib only。
package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Formula represents a parsed KPI calculation formula.
type Formula struct {
	Expression string
	root       node
}

type node interface {
	eval(counters map[string]float64) (float64, error)
}

type numberNode struct{ value float64 }

func (n *numberNode) eval(_ map[string]float64) (float64, error) { return n.value, nil }

type identNode struct{ name string }

func (n *identNode) eval(counters map[string]float64) (float64, error) {
	v, ok := counters[n.name]
	if !ok {
		return 0, fmt.Errorf("counter not found: %s", n.name)
	}
	return v, nil
}

type binaryNode struct {
	op    byte
	left  node
	right node
}

func (n *binaryNode) eval(counters map[string]float64) (float64, error) {
	l, err := n.left.eval(counters)
	if err != nil {
		return 0, err
	}
	r, err := n.right.eval(counters)
	if err != nil {
		return 0, err
	}
	switch n.op {
	case '+':
		return l + r, nil
	case '-':
		return l - r, nil
	case '*':
		return l * r, nil
	case '/':
		if r == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return l / r, nil
	default:
		return 0, fmt.Errorf("unknown operator: %c", n.op)
	}
}

// Parse parses a KPI formula expression into a Formula.
func Parse(expression string) (*Formula, error) {
	p := &parser{input: strings.TrimSpace(expression)}
	root, err := p.parseExpr()
	if err != nil {
		return nil, fmt.Errorf("parse formula %q: %w", expression, err)
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("parse formula %q: unexpected character at position %d", expression, p.pos)
	}
	return &Formula{Expression: expression, root: root}, nil
}

// Evaluate computes the formula result given counter values.
func (f *Formula) Evaluate(counters map[string]float64) (float64, error) {
	return f.root.eval(counters)
}

// Identifiers 返回公式中引用的去重 counter 标识符（按首次出现顺序）。
// KPI Router 需要从 DB 里取到的 formula 字符串反推它依赖哪些 counter，
// 不再依赖 carrier 适配器代码里硬编码的 Counters 列表。
func (f *Formula) Identifiers() []string {
	if f == nil || f.root == nil {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	collectIdentifiers(f.root, seen, &out)
	return out
}

func collectIdentifiers(n node, seen map[string]struct{}, out *[]string) {
	switch v := n.(type) {
	case *identNode:
		if _, dup := seen[v.name]; dup {
			return
		}
		seen[v.name] = struct{}{}
		*out = append(*out, v.name)
	case *binaryNode:
		collectIdentifiers(v.left, seen, out)
		collectIdentifiers(v.right, seen, out)
	}
}

type parser struct {
	input string
	pos   int
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) peek() byte {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) parseExpr() (node, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		op := p.peek()
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseTerm() (node, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		op := p.peek()
		if op != '*' && op != '/' {
			break
		}
		p.pos++
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseFactor() (node, error) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	ch := p.input[p.pos]

	if ch == '(' {
		p.pos++
		n, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return n, nil
	}

	if ch >= '0' && ch <= '9' || ch == '.' {
		start := p.pos
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
			p.pos++
		}
		val, err := strconv.ParseFloat(p.input[start:p.pos], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", p.input[start:p.pos])
		}
		return &numberNode{value: val}, nil
	}

	if ch == '_' || unicode.IsLetter(rune(ch)) {
		start := p.pos
		for p.pos < len(p.input) && (p.input[p.pos] == '_' || unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos]))) {
			p.pos++
		}
		return &identNode{name: p.input[start:p.pos]}, nil
	}

	return nil, fmt.Errorf("unexpected character: %c at position %d", ch, p.pos)
}
