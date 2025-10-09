package main

import (
	"fmt"
	"os"
)

// A Command is []string
// A Pipeline is []Command
func parse(tokens []Token) ([][]string, *os.File, *os.File) {
	outFile := os.Stdout
	errFile := os.Stderr
	pipeline := [][]string{{}}
	cmdIdx := 0
	var prev Token
	var err error

	for _, token := range tokens {
		switch token.Type {
		case Word:
			if prev.Type == Redirect {
				switch prev.Value {
				case ">", "1>":
					outFile, err = os.Create(token.Value)
				case "2>":
					errFile, err = os.Create(token.Value)
				case ">>", "1>>":
					// os.Create is O_RDWR|O_CREATE|O_TRUNC
					outFile, err = os.OpenFile(token.Value, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				case "2>>":
					errFile, err = os.OpenFile(token.Value, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				}

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating stdout/stderr file: %v\n", err)
					return nil, nil, nil
				}
			} else {
				pipeline[cmdIdx] = append(pipeline[cmdIdx], token.Value)
			}
		case Redirect:
		case Pipe:
			cmdIdx++
			pipeline = append(pipeline, []string{})
		}
		prev = token
	}

	return pipeline, outFile, errFile
}
