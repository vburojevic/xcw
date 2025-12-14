package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
)

type whereTokenType int

const (
	whereTokEOF whereTokenType = iota
	whereTokIdent
	whereTokString
	whereTokNumber
	whereTokRegex
	whereTokLParen
	whereTokRParen

	// Boolean operators (symbol forms only; keyword forms are parsed from identifiers).
	whereTokAnd // &&
	whereTokOr  // ||
	whereTokNot // !

	// Comparison operators
	whereTokEq       // =
	whereTokNe       // !=
	whereTokMatch    // ~
	whereTokNotMatch // !~
	whereTokGte      // >=
	whereTokLte      // <=
	whereTokStarts   // ^
	whereTokEnds     // $
)

type whereToken struct {
	typ whereTokenType
	val string
	pos int
}

func lexWhereExpr(input string) ([]whereToken, error) {
	var toks []whereToken
	i := 0
	for i < len(input) {
		ch := input[i]
		if isWhereSpace(ch) {
			i++
			continue
		}

		switch ch {
		case '(':
			toks = append(toks, whereToken{typ: whereTokLParen, pos: i})
			i++
			continue
		case ')':
			toks = append(toks, whereToken{typ: whereTokRParen, pos: i})
			i++
			continue
		case '&':
			if i+1 < len(input) && input[i+1] == '&' {
				toks = append(toks, whereToken{typ: whereTokAnd, pos: i})
				i += 2
				continue
			}
			return nil, fmt.Errorf("unexpected character '&' at %d (use && for AND)", i)
		case '|':
			if i+1 < len(input) && input[i+1] == '|' {
				toks = append(toks, whereToken{typ: whereTokOr, pos: i})
				i += 2
				continue
			}
			return nil, fmt.Errorf("unexpected character '|' at %d (use || for OR)", i)
		case '!':
			// !=, !~, or unary !
			if i+1 < len(input) && input[i+1] == '=' {
				toks = append(toks, whereToken{typ: whereTokNe, pos: i})
				i += 2
				continue
			}
			if i+1 < len(input) && input[i+1] == '~' {
				toks = append(toks, whereToken{typ: whereTokNotMatch, pos: i})
				i += 2
				continue
			}
			toks = append(toks, whereToken{typ: whereTokNot, pos: i})
			i++
			continue
		case '>':
			if i+1 < len(input) && input[i+1] == '=' {
				toks = append(toks, whereToken{typ: whereTokGte, pos: i})
				i += 2
				continue
			}
			return nil, fmt.Errorf("unexpected character '>' at %d (use >=)", i)
		case '<':
			if i+1 < len(input) && input[i+1] == '=' {
				toks = append(toks, whereToken{typ: whereTokLte, pos: i})
				i += 2
				continue
			}
			return nil, fmt.Errorf("unexpected character '<' at %d (use <=)", i)
		case '=':
			toks = append(toks, whereToken{typ: whereTokEq, pos: i})
			i++
			continue
		case '~':
			toks = append(toks, whereToken{typ: whereTokMatch, pos: i})
			i++
			continue
		case '^':
			toks = append(toks, whereToken{typ: whereTokStarts, pos: i})
			i++
			continue
		case '$':
			toks = append(toks, whereToken{typ: whereTokEnds, pos: i})
			i++
			continue
		case '\'', '"':
			unq, next, err := lexWhereString(input, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, whereToken{typ: whereTokString, val: unq, pos: i})
			i = next
			continue
		case '/':
			pat, next, err := lexWhereRegex(input, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, whereToken{typ: whereTokRegex, val: pat, pos: i})
			i = next
			continue
		default:
			if isWhereDigit(ch) {
				start := i
				for i < len(input) && isWhereDigit(input[i]) {
					i++
				}
				toks = append(toks, whereToken{typ: whereTokNumber, val: input[start:i], pos: start})
				continue
			}

			start := i
			for i < len(input) && !isWhereDelimiter(input[i]) {
				i++
			}
			val := strings.TrimSpace(input[start:i])
			if val == "" {
				return nil, fmt.Errorf("unexpected character %q at %d", input[start], start)
			}
			toks = append(toks, whereToken{typ: whereTokIdent, val: val, pos: start})
			continue
		}
	}
	toks = append(toks, whereToken{typ: whereTokEOF, pos: len(input)})
	return toks, nil
}

func isWhereSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isWhereDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isWhereDelimiter(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	case '(', ')', '&', '|', '!', '>', '<', '=', '~', '^', '$', '\'', '"', '/':
		return true
	default:
		return false
	}
}

func lexWhereString(input string, start int) (string, int, error) {
	quote := input[start]
	i := start + 1
	for i < len(input) {
		if input[i] == '\\' {
			i += 2
			continue
		}
		if input[i] == quote {
			lit := input[start : i+1]
			unq, err := strconv.Unquote(lit)
			if err != nil {
				return "", 0, fmt.Errorf("invalid quoted string at %d: %w", start, err)
			}
			return unq, i + 1, nil
		}
		i++
	}
	return "", 0, fmt.Errorf("unterminated string starting at %d", start)
}

func lexWhereRegex(input string, start int) (string, int, error) {
	// /.../flags where '/' inside pattern must be escaped as \/
	i := start + 1
	for i < len(input) {
		if input[i] == '\\' {
			i += 2
			continue
		}
		if input[i] == '/' {
			raw := input[start+1 : i]
			pat := unescapeRegexDelim(raw)

			// optional flags
			j := i + 1
			for j < len(input) && isWhereAlpha(input[j]) {
				j++
			}
			flags := strings.ToLower(input[i+1 : j])
			if flags != "" {
				patWithFlags, err := applyRegexFlags(pat, flags)
				if err != nil {
					return "", 0, fmt.Errorf("invalid regex flags at %d: %w", start, err)
				}
				pat = patWithFlags
			}

			return pat, j, nil
		}
		i++
	}
	return "", 0, fmt.Errorf("unterminated regex literal starting at %d", start)
}

func isWhereAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func unescapeRegexDelim(raw string) string {
	if !strings.Contains(raw, `\/`) {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) && raw[i+1] == '/' {
			b.WriteByte('/')
			i++
			continue
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}

func applyRegexFlags(pattern, flags string) (string, error) {
	allowed := map[rune]bool{'i': true, 'm': true, 's': true}
	seen := map[rune]bool{}
	var b strings.Builder
	for _, r := range flags {
		if !allowed[r] {
			return "", fmt.Errorf("unsupported flag %q (supported: i, m, s)", string(r))
		}
		if !seen[r] {
			seen[r] = true
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return pattern, nil
	}
	return "(?" + b.String() + ")" + pattern, nil
}

type whereExpr interface {
	Match(entry *domain.LogEntry) bool
}

type whereClauseExpr struct {
	clause *WhereClause
}

func (e *whereClauseExpr) Match(entry *domain.LogEntry) bool {
	if e == nil || e.clause == nil {
		return true
	}
	return e.clause.Match(entry)
}

type whereAndExpr struct {
	left  whereExpr
	right whereExpr
}

func (e *whereAndExpr) Match(entry *domain.LogEntry) bool {
	return e.left.Match(entry) && e.right.Match(entry)
}

type whereOrExpr struct {
	left  whereExpr
	right whereExpr
}

func (e *whereOrExpr) Match(entry *domain.LogEntry) bool {
	return e.left.Match(entry) || e.right.Match(entry)
}

type whereNotExpr struct {
	inner whereExpr
}

func (e *whereNotExpr) Match(entry *domain.LogEntry) bool {
	return !e.inner.Match(entry)
}

type whereParser struct {
	input string
	toks  []whereToken
	pos   int
}

func parseWhereExpr(input string) (whereExpr, error) {
	toks, err := lexWhereExpr(input)
	if err != nil {
		return nil, err
	}
	p := &whereParser{input: input, toks: toks}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().typ != whereTokEOF {
		return nil, fmt.Errorf("unexpected token %q at %d", p.peek().val, p.peek().pos)
	}
	return expr, nil
}

func (p *whereParser) peek() whereToken {
	if p.pos >= len(p.toks) {
		return whereToken{typ: whereTokEOF, pos: len(p.input)}
	}
	return p.toks[p.pos]
}

func (p *whereParser) next() whereToken {
	t := p.peek()
	if p.pos < len(p.toks) {
		p.pos++
	}
	return t
}

func (p *whereParser) matchIdentKeyword(kw string) bool {
	t := p.peek()
	if t.typ != whereTokIdent {
		return false
	}
	if !strings.EqualFold(t.val, kw) {
		return false
	}
	p.next()
	return true
}

func (p *whereParser) matchToken(typ whereTokenType) bool {
	if p.peek().typ != typ {
		return false
	}
	p.next()
	return true
}

func (p *whereParser) parseOr() (whereExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		if p.matchToken(whereTokOr) || p.matchIdentKeyword("or") {
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &whereOrExpr{left: left, right: right}
			continue
		}
		return left, nil
	}
}

func (p *whereParser) parseAnd() (whereExpr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		if p.matchToken(whereTokAnd) || p.matchIdentKeyword("and") {
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &whereAndExpr{left: left, right: right}
			continue
		}
		return left, nil
	}
}

func (p *whereParser) parseUnary() (whereExpr, error) {
	if p.matchToken(whereTokNot) || p.matchIdentKeyword("not") {
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &whereNotExpr{inner: inner}, nil
	}
	return p.parsePrimary()
}

func (p *whereParser) parsePrimary() (whereExpr, error) {
	if p.matchToken(whereTokLParen) {
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.matchToken(whereTokRParen) {
			return nil, fmt.Errorf("expected ')' at %d", p.peek().pos)
		}
		return inner, nil
	}
	return p.parseComparison()
}

func (p *whereParser) parseComparison() (whereExpr, error) {
	fieldTok := p.next()
	if fieldTok.typ != whereTokIdent {
		return nil, fmt.Errorf("expected field name at %d", fieldTok.pos)
	}

	opTok := p.next()
	op, ok := whereOpString(opTok.typ)
	if !ok {
		return nil, fmt.Errorf("expected operator after field %q at %d", fieldTok.val, opTok.pos)
	}

	valTok := p.next()
	switch valTok.typ {
	case whereTokIdent, whereTokString, whereTokNumber, whereTokRegex:
		// ok
	default:
		return nil, fmt.Errorf("expected value after %q at %d", op, valTok.pos)
	}

	wc := &WhereClause{
		Field:    fieldTok.val,
		Operator: op,
		Value:    valTok.val,
	}

	if op == "~" || op == "!~" {
		re, err := regexp.Compile(wc.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid regex in where expression: %w", err)
		}
		wc.regex = re
	}

	return &whereClauseExpr{clause: wc}, nil
}

func whereOpString(typ whereTokenType) (string, bool) {
	switch typ {
	case whereTokEq:
		return "=", true
	case whereTokNe:
		return "!=", true
	case whereTokMatch:
		return "~", true
	case whereTokNotMatch:
		return "!~", true
	case whereTokGte:
		return ">=", true
	case whereTokLte:
		return "<=", true
	case whereTokStarts:
		return "^", true
	case whereTokEnds:
		return "$", true
	default:
		return "", false
	}
}
