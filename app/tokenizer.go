package main

import (
	"strings"
)

type TokenType int

const (
	Word     TokenType = iota
	Redirect           // >, >>, 1>, 2>, 1>>, 2>>
	Pipe
)

// todo: handle redirection '&>file' and '>file 2>&1'

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
	Pipeline
)

func tokenize(input string) []Token {
	tokens := []Token{}
	var curr strings.Builder
	state := Normal
	tokenType := Word

	flush := func() {
		if curr.Len() > 0 {
			tokens = append(tokens, Token{Type: tokenType, Value: curr.String()})
			curr.Reset()
		}
	}

	for i := 0; i < len(input); i++ {
		ch := input[i]

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
				flush()
			case '>':
				flush()
				// check for >>
				if i+1 < len(input) && input[i+1] == '>' {
					tokens = append(tokens, Token{Type: Redirect, Value: ">>"})
					i++
				} else {
					tokens = append(tokens, Token{Type: Redirect, Value: ">"})
				}
			case '1', '2':
				if i+1 < len(input) && input[i+1] == '>' {
					flush()
					if i+2 < len(input) && input[i+2] == '>' {
						tokens = append(tokens, Token{Type: Redirect, Value: string(ch) + ">>"})
						i += 2
					} else {
						tokens = append(tokens, Token{Type: Redirect, Value: string(ch) + ">"})
						i++
					}
				} else {
					curr.WriteByte(ch)
				}

			default:
				curr.WriteByte(ch)
			}

		case SingleQuote:
			if ch == '\'' {
				state = Normal
			} else {
				curr.WriteByte(ch)
			}

		case DoubleQuote:
			switch ch {
			case '"':
				state = Normal
			case '\\':
				state = BackSlashDoubleQuote
			default:
				curr.WriteByte(ch)
			}

		case BackSlash:
			curr.WriteByte(ch)
			state = Normal

		case BackSlashDoubleQuote:
			// Backslash inside double quotes only escapes " and \ (for this challenge)
			// If ch is neither of those, add the \ back to curr
			if ch != '"' && ch != '\\' {
				curr.WriteByte('\\')
			}
			curr.WriteByte(ch)
			state = DoubleQuote

		default:
			state = Normal

			if ch == ' ' || ch == '\n' {
				flush()
			} else {
				tokenType = Word
				curr.WriteByte(ch)
			}
		}
	}

	flush()

	return tokens
}
