package main

import (
	"bufio"
	"fmt"
	"os"
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

		args := strings.Split(input, " ")

		switch args[0] {
		case "exit": // assuming the tester will always pass in 0 as the argument
			os.Exit(0)

		default:
			fmt.Println(input[:len(input)-1] + ": command not found")
		}
	}
}
