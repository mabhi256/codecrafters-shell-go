package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

var inbuiltCommands = []string{"exit", "echo", "type", "pwd"}

func findMatchingCmd(partial string) []string {
	// fmt.Printf("partial: %s\r\n", partial)
	var matches []string
	for _, cmd := range inbuiltCommands {
		if strings.HasPrefix(cmd, partial) {
			matches = append(matches, cmd)
		}
	}

	envPath := os.Getenv("PATH")
	paths := strings.SplitSeq(envPath, string(os.PathListSeparator)) // : on Unix, ; on Windows

	for path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), partial) {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				if !info.IsDir() && info.Mode()&0111 != 0 {
					matches = append(matches, entry.Name())
				}
			}
		}
	}

	slices.Sort(matches)
	return matches
}

func readline(reader *bufio.Reader) (string, error) {
	var input []byte
	var matches []string
	matchIdx := 0

	for {
		ch, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		switch ch {
		case 3: // ctrl+C
			fmt.Println("^C")
			os.Exit(0)

		case 4: // ctrl + D (EOF)
			if len(input) == 0 {
				fmt.Println()
				os.Exit(0)
			}
		case '\t': // 9 - tab
			if len(input) > 0 {
				partial := string(input)

				if len(matches) == 0 {
					matches = findMatchingCmd(partial)
				} else {
					matchIdx = (matchIdx + 1) % len(matches)
				}

				if len(matches) > 0 {
					for range len(partial) {
						fmt.Print("\b \b")
					}
					input = []byte(matches[matchIdx] + " ")
					fmt.Print(string(input))
				} else {
					fmt.Printf("%c", 0x07) // bell
					fmt.Print("\b \b")
				}
			}

		case '\r', '\n': // 10, 13
			if len(input) > 0 {
				matches = nil
				matchIdx = 0
				return string(input), nil
			}

		case '\b', 127:
			if len(input) > 0 {
				// \b is 8 but most terminals send 127 for backspace
				input = input[:len(input)-1]
				// first \b moves cursor back one position, over the character. ' ' replaces it,
				// second \b moves cursor back again to its correct position
				fmt.Print("\b \b")
			}
			matches = nil
			matchIdx = 0

		default:
			input = append(input, ch)
			fmt.Printf("%c", ch)
			matches = nil
			matchIdx = 0
		}

	}
}
