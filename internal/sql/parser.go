package sql

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// SelectStmt represents a parsed SQL SELECT statement.
type SelectStmt struct {
	SelectAll bool
	Columns   []SelectColumn
	From      string
	Joins     []JoinClause
	Where     SQLExpr
	GroupBy   []string
	OrderBy   []OrderByClause
	Limit     int
}

// SelectColumn represents a column in the SELECT clause.
type SelectColumn struct {
	Name    string // column name or "*"
	Alias   string // AS alias
	AggFunc string // e.g., "SUM", "COUNT", etc.
}

// JoinClause represents a JOIN in the query.
type JoinClause struct {
	Type  string // INNER, LEFT, RIGHT, FULL, CROSS
	Table string
	On    string // single column name for now
}

// OrderByClause represents an ORDER BY column.
type OrderByClause struct {
	Column string
	Desc   bool
}

// SQLExpr represents a SQL expression (WHERE clause).
type SQLExpr interface {
	sqlExpr()
}

// ColumnRef is a column reference in an expression.
type ColumnRef struct{ Name string }

func (ColumnRef) sqlExpr() {}

// LiteralInt is an integer literal.
type LiteralInt struct{ Value int64 }

func (LiteralInt) sqlExpr() {}

// LiteralFloat is a float literal.
type LiteralFloat struct{ Value float64 }

func (LiteralFloat) sqlExpr() {}

// LiteralString is a string literal.
type LiteralString struct{ Value string }

func (LiteralString) sqlExpr() {}

// BinaryOp is a binary comparison/logical expression.
type BinaryOp struct {
	Left  SQLExpr
	Op    string // "=", "!=", "<", ">", "<=", ">=", "AND", "OR"
	Right SQLExpr
}

func (BinaryOp) sqlExpr() {}

// token types
type tokenType int

const (
	tokEOF tokenType = iota
	tokIdent
	tokNumber
	tokString
	tokComma
	tokStar
	tokDot
	tokLParen
	tokRParen
	tokOp // =, !=, <, >, <=, >=
)

type token struct {
	typ tokenType
	val string
}

// Parse parses a SQL SELECT statement.
func Parse(query string) (*SelectStmt, error) {
	p := &parser{tokens: tokenize(query)}
	return p.parseSelect()
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	t := p.peek()
	if t.typ != tokEOF {
		p.pos++
	}
	return t
}

func (p *parser) expect(val string) error {
	t := p.next()
	if !strings.EqualFold(t.val, val) {
		return fmt.Errorf("expected %q, got %q", val, t.val)
	}
	return nil
}

func (p *parser) isKeyword(kw string) bool {
	return strings.EqualFold(p.peek().val, kw)
}

func (p *parser) parseSelect() (*SelectStmt, error) {
	if err := p.expect("SELECT"); err != nil {
		return nil, err
	}

	stmt := &SelectStmt{}

	// Parse columns
	if p.peek().typ == tokStar {
		p.next()
		stmt.SelectAll = true
	} else {
		cols, err := p.parseColumns()
		if err != nil {
			return nil, err
		}
		stmt.Columns = cols
	}

	// FROM
	if err := p.expect("FROM"); err != nil {
		return nil, err
	}
	t := p.next()
	stmt.From = t.val

	// JOINs
	for p.isKeyword("JOIN") || p.isKeyword("INNER") || p.isKeyword("LEFT") ||
		p.isKeyword("RIGHT") || p.isKeyword("FULL") || p.isKeyword("CROSS") {
		j, err := p.parseJoin()
		if err != nil {
			return nil, err
		}
		stmt.Joins = append(stmt.Joins, j)
	}

	// WHERE
	if p.isKeyword("WHERE") {
		p.next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		stmt.Where = expr
	}

	// GROUP BY
	if p.isKeyword("GROUP") {
		p.next()
		if err := p.expect("BY"); err != nil {
			return nil, err
		}
		for {
			t := p.next()
			stmt.GroupBy = append(stmt.GroupBy, t.val)
			if p.peek().typ != tokComma {
				break
			}
			p.next() // skip comma
		}
	}

	// ORDER BY
	if p.isKeyword("ORDER") {
		p.next()
		if err := p.expect("BY"); err != nil {
			return nil, err
		}
		for {
			t := p.next()
			ob := OrderByClause{Column: t.val}
			if p.isKeyword("DESC") {
				p.next()
				ob.Desc = true
			} else if p.isKeyword("ASC") {
				p.next()
			}
			stmt.OrderBy = append(stmt.OrderBy, ob)
			if p.peek().typ != tokComma {
				break
			}
			p.next() // skip comma
		}
	}

	// LIMIT
	if p.isKeyword("LIMIT") {
		p.next()
		t := p.next()
		n, err := strconv.Atoi(t.val)
		if err != nil {
			return nil, fmt.Errorf("invalid LIMIT: %q", t.val)
		}
		stmt.Limit = n
	}

	return stmt, nil
}

func (p *parser) parseColumns() ([]SelectColumn, error) {
	var cols []SelectColumn
	for {
		col, err := p.parseColumn()
		if err != nil {
			return nil, err
		}
		cols = append(cols, col)
		if p.peek().typ != tokComma {
			break
		}
		p.next() // skip comma
	}
	return cols, nil
}

func (p *parser) parseColumn() (SelectColumn, error) {
	// Check for aggregate: SUM(col), COUNT(col), etc.
	if p.peek().typ == tokIdent && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].typ == tokLParen {
		aggName := strings.ToUpper(p.next().val)
		p.next() // skip (
		colName := p.next().val
		if p.peek().typ != tokRParen {
			return SelectColumn{}, fmt.Errorf("expected ')' after aggregate")
		}
		p.next() // skip )

		alias := colName
		if p.isKeyword("AS") {
			p.next()
			alias = p.next().val
		}

		return SelectColumn{Name: colName, Alias: alias, AggFunc: aggName}, nil
	}

	t := p.next()
	col := SelectColumn{Name: t.val}

	if p.isKeyword("AS") {
		p.next()
		col.Alias = p.next().val
	}

	return col, nil
}

func (p *parser) parseJoin() (JoinClause, error) {
	jt := "INNER"
	if p.isKeyword("LEFT") {
		jt = "LEFT"
		p.next()
	} else if p.isKeyword("RIGHT") {
		jt = "RIGHT"
		p.next()
	} else if p.isKeyword("FULL") {
		jt = "FULL"
		p.next()
	} else if p.isKeyword("CROSS") {
		jt = "CROSS"
		p.next()
	} else if p.isKeyword("INNER") {
		p.next()
	}

	if err := p.expect("JOIN"); err != nil {
		return JoinClause{}, err
	}

	table := p.next().val

	j := JoinClause{Type: jt, Table: table}

	if p.isKeyword("ON") {
		p.next()
		// Parse simple: left.col = right.col or just col = col
		left := p.next().val
		// Handle table.col notation
		if p.peek().typ == tokDot {
			p.next()
			left = p.next().val
		}
		p.next() // skip =
		right := p.next().val
		if p.peek().typ == tokDot {
			p.next()
			right = p.next().val
		}
		// Use the column name (should be the same for both sides)
		j.On = left
		_ = right
	}

	return j, nil
}

func (p *parser) parseExpr() (SQLExpr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		t := p.peek()
		if t.typ == tokOp {
			p.next()
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = BinaryOp{Left: left, Op: t.val, Right: right}
		} else if p.isKeyword("AND") {
			p.next()
			right, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			left = BinaryOp{Left: left, Op: "AND", Right: right}
		} else if p.isKeyword("OR") {
			p.next()
			right, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			left = BinaryOp{Left: left, Op: "OR", Right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *parser) parsePrimary() (SQLExpr, error) {
	t := p.peek()

	if t.typ == tokLParen {
		p.next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tokRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.next()
		return expr, nil
	}

	p.next()

	if t.typ == tokString {
		return LiteralString{Value: t.val}, nil
	}

	if t.typ == tokNumber {
		if strings.Contains(t.val, ".") {
			v, _ := strconv.ParseFloat(t.val, 64)
			return LiteralFloat{Value: v}, nil
		}
		v, _ := strconv.ParseInt(t.val, 10, 64)
		return LiteralInt{Value: v}, nil
	}

	return ColumnRef{Name: t.val}, nil
}

func tokenize(input string) []token {
	var tokens []token
	i := 0
	runes := []rune(input)

	for i < len(runes) {
		ch := runes[i]

		// Skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// String literal
		if ch == '\'' {
			i++
			start := i
			for i < len(runes) && runes[i] != '\'' {
				i++
			}
			tokens = append(tokens, token{typ: tokString, val: string(runes[start:i])})
			if i < len(runes) {
				i++ // skip closing quote
			}
			continue
		}

		// Number
		if unicode.IsDigit(ch) || (ch == '-' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])) {
			start := i
			if ch == '-' {
				i++
			}
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, token{typ: tokNumber, val: string(runes[start:i])})
			continue
		}

		// Operators
		if ch == '=' {
			tokens = append(tokens, token{typ: tokOp, val: "="})
			i++
			continue
		}
		if ch == '!' && i+1 < len(runes) && runes[i+1] == '=' {
			tokens = append(tokens, token{typ: tokOp, val: "!="})
			i += 2
			continue
		}
		if ch == '<' {
			if i+1 < len(runes) && runes[i+1] == '=' {
				tokens = append(tokens, token{typ: tokOp, val: "<="})
				i += 2
			} else if i+1 < len(runes) && runes[i+1] == '>' {
				tokens = append(tokens, token{typ: tokOp, val: "!="})
				i += 2
			} else {
				tokens = append(tokens, token{typ: tokOp, val: "<"})
				i++
			}
			continue
		}
		if ch == '>' {
			if i+1 < len(runes) && runes[i+1] == '=' {
				tokens = append(tokens, token{typ: tokOp, val: ">="})
				i += 2
			} else {
				tokens = append(tokens, token{typ: tokOp, val: ">"})
				i++
			}
			continue
		}

		// Punctuation
		if ch == ',' {
			tokens = append(tokens, token{typ: tokComma, val: ","})
			i++
			continue
		}
		if ch == '*' {
			tokens = append(tokens, token{typ: tokStar, val: "*"})
			i++
			continue
		}
		if ch == '(' {
			tokens = append(tokens, token{typ: tokLParen, val: "("})
			i++
			continue
		}
		if ch == ')' {
			tokens = append(tokens, token{typ: tokRParen, val: ")"})
			i++
			continue
		}
		if ch == '.' {
			tokens = append(tokens, token{typ: tokDot, val: "."})
			i++
			continue
		}
		if ch == ';' {
			i++
			continue
		}

		// Identifier/keyword
		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, token{typ: tokIdent, val: string(runes[start:i])})
			continue
		}

		i++ // skip unknown
	}

	return tokens
}
