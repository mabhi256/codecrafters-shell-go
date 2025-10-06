package main

import (
	"strings"
)

type TokenType int

const (
	Word TokenType = iota
	Stdout
	Stderr
	StdoutAppend
	StderrAppend
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
	RedirectOut
	RedirectOne
	RedirectErr
	RedirectTwo
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
				flush()
			case '>':
				flush()
				state = RedirectOut
			case '1':
				if curr.Len() == 0 {
					state = RedirectOne
				} else {
					curr.WriteRune(ch)
				}
			case '2':
				if curr.Len() == 0 {
					state = RedirectTwo
				} else {
					curr.WriteRune(ch)
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
			switch ch {
			case '"':
				state = Normal
			case '\\':
				state = BackSlashDoubleQuote
			default:
				curr.WriteRune(ch)
			}

		case BackSlash:
			curr.WriteRune(ch)
			state = Normal

		case BackSlashDoubleQuote:
			// Backslash inside double quotes only escapes " and \ (for this challenge)
			// If ch is neither of those, add the \ back to curr
			if ch != '"' && ch != '\\' {
				curr.WriteRune('\\')
			}
			curr.WriteRune(ch)
			state = DoubleQuote

		case RedirectOut:
			if ch == '>' {
				tokenType = StdoutAppend
				state = Normal

				continue
			} else {
				tokenType = Stdout
				state = Normal

				if ch != ' ' {
					curr.WriteRune(ch)
				}
			}

		case RedirectOne:
			if ch == '>' {
				state = RedirectOut
				continue
			}

			// Not a redirect, add 1 back to curr
			state = Normal
			curr.WriteRune('1')

			if ch == ' ' || ch == '\n' {
				flush()
			} else {
				curr.WriteRune(ch)
			}

		case RedirectErr:
			if ch == '>' {
				tokenType = StderrAppend
				state = Normal

				continue
			} else {
				tokenType = Stderr
				state = Normal

				if ch != ' ' {
					curr.WriteRune(ch)
				}
			}

		case RedirectTwo:
			if ch == '>' {
				state = RedirectErr
				continue
			}

			// Not a redirect, add 2 back to curr
			state = Normal
			curr.WriteRune('2')

			if ch == ' ' || ch == '\n' {
				flush()
			} else {
				curr.WriteRune(ch)
			}
		}
	}

	flush()

	return tokens
}
