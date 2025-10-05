package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		input = strings.TrimSuffix(input, "\n")

		args := strings.Split(input, " ")

		validCommands := []string{"exit", "echo", "type"}

		switch args[0] {
		case "exit": // assuming the tester will always pass in 0 as the argument
			os.Exit(0)

		case "echo":
			fmt.Println(input[5:]) // "echo" + " "

		case "type":
			if slices.Contains(validCommands, args[1]) {
				fmt.Println(args[1], "is a shell builtin")
			} else {
				envPath := os.Getenv("PATH")
				paths := strings.Split(envPath, string(os.PathListSeparator)) // : on Unix, ; on Windows

				found := false
				var execPath string
				for _, path := range paths {
					execPath = filepath.Join(path, args[1])
					stat, err := os.Stat(execPath)
					if os.IsNotExist(err) {
						continue
					} else if err == nil {
						// Owner 	Group	Others
						// rwx   	rwx   	rwx
						// 421		421		421
						if stat.Mode()&0111 != 0 {
							found = true
							break
						}
					}
				}

				if found {
					fmt.Printf("%s is %s\n", args[1], execPath)
				} else {
					fmt.Printf("%s: not found\n", args[1])
				}
			}

		default:
			fmt.Println(input + ": command not found")
		}
	}
}
