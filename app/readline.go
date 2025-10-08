package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var inbuiltCommands = []string{"exit", "echo", "type", "pwd"}

func findMatchingCmd(partial string) []string {
	// fmt.Printf("partial: %s\r\n", partial)
	seen := map[string]bool{}

	var matches []string
	for _, cmd := range inbuiltCommands {
		if strings.HasPrefix(cmd, partial) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) > 0 {
		return matches
	}

	envPath := os.Getenv("PATH")
	paths := strings.SplitSeq(envPath, string(os.PathListSeparator)) // : on Unix, ; on Windows

	for path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !seen[entry.Name()] && strings.HasPrefix(entry.Name(), partial) {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				if !info.IsDir() && info.Mode()&0111 != 0 {
					matches = append(matches, entry.Name())
					seen[entry.Name()] = true
				}
			}
		}
	}

	slices.Sort(matches)
	return matches
}

func findMatchingPath(path string) []string {
	var matches []string

	dir := filepath.Dir(path)

	var base string
	if len(path) > 0 && path[len(path)-1] == '/' {
		base = "."
	} else {
		base = filepath.Base(path)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches
	}

	for _, entry := range entries {
		if base == "." || strings.HasPrefix(entry.Name(), base) {
			fullPath := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				fullPath += "/"
			}

			matches = append(matches, fullPath)
		}
	}

	return matches
}

func shouldCompleteCmd(input string) bool {
	if len(input) > 0 && input[len(input)-1] == ' ' {
		return false
	}

	trimmed := strings.TrimSpace(input)

	if strings.Contains(trimmed, "/") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.HasPrefix(trimmed, ".") {
		return false
	}

	if strings.Contains(trimmed, " ") {
		return false
	}

	return true
}

func readline(reader *bufio.Reader) (string, bool, error) {
	cursorPos := 0
	var input []byte
	var prefix []byte
	var matches []string
	matchIdx := 0
	isCmd := false
	matchAlert := false

	for {
		ch, err := reader.ReadByte()
		if err != nil {
			return "", false, err
		}

		switch ch {
		case 1: // Crtl+A
			if cursorPos > 0 {
				fmt.Printf("\x1b[%dD", cursorPos) // move cursor left by cursorPos units
				cursorPos = 0
			}

		case 3: // Ctrl+C
			fmt.Println("^C")
			os.Exit(0)

		case 4: // Ctrl+D (EOF)
			if len(input) == 0 {
				fmt.Println()
				os.Exit(0)
			}

		case 5: // Crtl+E
			if cursorPos < len(input) {
				fmt.Printf("\x1b[%dC", len(input)-cursorPos) // move cursor right
				cursorPos = len(input)
			}

		case '\t': // 9 - tab
			if len(input) > 0 {
				partial := string(input)
				if len(matches) == 0 {
					if shouldCompleteCmd(string(input)) {
						matches = findMatchingCmd(partial)
						prefix = nil
						isCmd = true
					} else {
						isCmd = false
						if len(input) > 0 && input[len(input)-1] == ' ' {
							prefix = input
							partial = ""
						} else {
							fields := strings.Fields(string(input))
							partial = fields[len(fields)-1]
							lastFieldStart := strings.LastIndex(string(input), partial)
							prefix = input[:lastFieldStart]
						}
						matches = findMatchingPath(partial)
					}
					matchIdx = 0
				} else {
					matchIdx = (matchIdx + 1) % len(matches)
				}

				if len(matches) > 0 {
					if isCmd {
						nextMatchId := (matchIdx + 1) % len(matches)

						// If current match is a prefix of the next match, show it immediately
						// else ring a bell and then show options
						// If len(matches) is 1, the the next and the current matches are same
						if strings.HasPrefix(matches[nextMatchId], matches[matchIdx]) {
							for range len(partial) {
								fmt.Print("\b \b")
							}
							input = []byte(matches[matchIdx])

							if len(matches) == 1 {
								input = append(input, ' ')
							}
							fmt.Print(string(input))
							cursorPos = len(input)
						} else {
							if !matchAlert {
								fmt.Print("\a")
								matchAlert = true
							} else {
								matchAlert = false
								fmt.Print("\r\n")
								fmt.Print(strings.Join(matches, "  "))
								fmt.Print("\r\n")

								return string(input), true, nil
							}
						}
					} else {
						for range len(input) {
							fmt.Print("\b \b")
						}
						input = append(prefix, []byte(matches[matchIdx])...)
						fmt.Print(string(input))
						cursorPos = len(input)
					}
				} else {
					fmt.Print("\a") // bell, 0x07
				}
			}

		case '\r', '\n': // 10, 13
			if len(input) > 0 {
				matches = nil
				matchIdx = 0
				return string(input), false, nil
			}

		case 23: // Ctrl+W
			if cursorPos > 0 {
				startPos := cursorPos - 1

				// skip trailing spaces
				for startPos > 0 && input[startPos] == ' ' {
					startPos--
				}
				// find the last word
				for startPos > 0 && input[startPos-1] != ' ' {
					startPos--
				}

				deleted := cursorPos - startPos
				input = append(input[:startPos], input[cursorPos:]...)

				fmt.Printf("\x1b[%dD", deleted)                     // Move cursor back to where deletion started
				fmt.Print(string(input[startPos:]))                 // Print remaining text
				fmt.Print(strings.Repeat(" ", deleted))             // Overwrite old chars (which have now moved left after deletion)
				fmt.Printf("\x1b[%dD", len(input)-startPos+deleted) // Move the cursor back after writing the remaining text + spaces

				cursorPos = startPos
			}

		case 27: // ESC - arrow keys start with this
			// Read the next two bytes for arrow key sequences
			seq := make([]byte, 2)
			n, err := reader.Read(seq)
			if err != nil || n != 2 {
				continue
			}

			// Arrow keys: ESC [ A/B/C/D
			if seq[0] == '[' {
				switch seq[1] {
				case 'A': // Up
				case 'B': // Down
				case 'C': // Right
					if cursorPos < len(input) {
						fmt.Print("\x1b[C") // ANSI code to move cursor right
						cursorPos++
					}
					// Commit the completion and reset for next tab
					if input[len(input)-1] == '/' {
						matches = nil
						matchIdx = 0
					}
				case 'D': // Left
					if cursorPos > 0 {
						fmt.Print("\x1b[D") // ANSI code to move cursor left
						cursorPos--
					}

				case 'H': // Home key (ESC [ H)
					if cursorPos > 0 {
						fmt.Printf("\x1b[%dD", cursorPos)
						cursorPos = 0
					}

				case 'F': // End key (ESC [ F)
					if cursorPos < len(input) {
						fmt.Printf("\x1b[%dC", len(input)-cursorPos)
						cursorPos = len(input)
					}
				}
			}

		case '\b', 127:
			if cursorPos > 0 {
				// \b is 8 but most terminals send 127 for backspace
				input = append(input[:cursorPos-1], input[cursorPos:]...)
				cursorPos--

				// Redraw line from cursor position
				fmt.Print("\b")

				// the space at the end erases the last character of the input.
				// Without this the the last char would be still visible
				// Before: h e l l o
				// Delete middle 'l' and print "lo" : h e l o o  ← old 'o' still visible
				fmt.Print(string(input[cursorPos:]) + " ")
				// Move cursor back
				fmt.Printf("\x1b[%dD", len(input)-cursorPos+1)
			}
			matches = nil
			matchIdx = 0

		default:
			// Insert character at cursor position
			tail := append([]byte(nil), input[cursorPos:]...)
			input = append(input[:cursorPos], ch)
			input = append(input, tail...)
			cursorPos++

			// Redraw from cursorPos to end
			fmt.Printf("%s", string(input[cursorPos-1:]))
			if cursorPos < len(input) {
				fmt.Printf("\x1b[%dD", len(input)-cursorPos)
			}
			matches = nil
			matchIdx = 0
		}

	}
}
