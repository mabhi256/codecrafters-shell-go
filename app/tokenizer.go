package main

import (
	"strings"
)

type TokenType int

const (
	Word TokenType = iota
)

type Token struct {
	Type  TokenType
	Value string
}

type TokenizerState int

const (
	Normal TokenizerState = iota
	SingleQuote
	DoubleQuote
	BackSlash
	BackSlashDoubleQuote
)

func tokenize(input string) []Token {
	tokens := []Token{}
	var curr strings.Builder
	state := Normal

	for _, ch := range input {
		switch state {
		case Normal:
			switch ch {
			case '\'':
				state = SingleQuote
			case '"':
				state = DoubleQuote
			case '\\':
				state = BackSlash
			case ' ', '\n':
				if curr.Len() > 0 {
					tokens = append(tokens, Token{Type: Word, Value: curr.String()})
					curr.Reset()
				}

			default:
				curr.WriteRune(ch)
			}

		case SingleQuote:
			if ch == '\'' {
				state = Normal
			} else {
				curr.WriteRune(ch)
			}

		case DoubleQuote:
			if ch == '"' {
				state = Normal
			} else {
				curr.WriteRune(ch)
			}

		case BackSlash:
			curr.WriteRune(ch)
			state = Normal
		}
	}

	return tokens
}
