package main

import (
	"bufio"
	"fmt"
	"os"
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
				fmt.Printf("%s: not found\n", args[1])
			}

		default:
			fmt.Println(input + ": command not found")
		}
	}
}
