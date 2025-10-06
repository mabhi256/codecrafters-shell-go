package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func getExecPath(command string) (string, bool) {
	envPath := os.Getenv("PATH")
	paths := strings.Split(envPath, string(os.PathListSeparator)) // : on Unix, ; on Windows

	found := false
	var execPath string
	for _, path := range paths {
		execPath = filepath.Join(path, command)
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
		} else {
			fmt.Println("Stat error for", execPath, ":", err)
		}
	}

	return execPath, found
}

func execute(args []string, outFile *os.File, errFile *os.File) {
	validCommands := []string{"exit", "echo", "type", "pwd"}

	command := args[0]

	switch command {
	case "exit": // assuming the tester will always pass in 0 as the argument
		os.Exit(0)

	case "echo":
		fmt.Fprintf(outFile, "%s\n", strings.Join(args[1:], " ")) // "echo" + " "

	case "type":
		if slices.Contains(validCommands, args[1]) {
			fmt.Fprintf(outFile, "%s is a shell builtin\n", args[1])
			return
		}

		execPath, found := getExecPath(args[1])
		if found {
			fmt.Fprintf(outFile, "%s is %s\n", args[1], execPath)
		} else {
			fmt.Fprintf(outFile, "%s: not found\n", args[1])
		}

	case "pwd":
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(errFile, "Unable to get pwd\n")
			return
		}

		fmt.Fprintf(outFile, "%s\n", pwd)

	case "cd":
		if len(args) > 1 && args[1] != "~" {
			err := os.Chdir(args[1])
			if err != nil {
				fmt.Fprintf(errFile, "cd: %s: No such file or directory\n", args[1])
			}
		} else {
			home := os.Getenv("HOME")
			os.Chdir(home)
		}

	default:
		_, found := getExecPath(command)
		if found {
			var cmd *exec.Cmd
			if len(args) > 1 {
				cmd = exec.Command(command, args[1:]...)
			} else {
				cmd = exec.Command(command)
			}

			cmd.Stderr = errFile
			cmd.Stdout = outFile
			cmd.Run()
		} else {
			fmt.Fprintf(outFile, "%s: command not found\n", command)
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}

		tokens := tokenize(input)
		// fmt.Fprintf(os.Stderr, "tokens: len(%d) %v\n", len(tokens), tokens)

		outFile := os.Stdout
		errFile := os.Stderr
		args := []string{}
		for _, token := range tokens {
			switch token.Type {
			case Word:
				args = append(args, token.Value)
			case Stdout:
				outFile, err = os.Create(token.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating stdout file: %v\n", err)
					os.Exit(1)
				}
			case Stderr:
				errFile, err = os.Create(token.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating stderr file: %v\n", err)
					os.Exit(1)
				}
			case StdoutAppend:
				// os.Create is O_RDWR|O_CREATE|O_TRUNC
				outFile, err = os.OpenFile(token.Value, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error opening stdout file: %v\n", err)
					os.Exit(1)
				}
			case StderrAppend:
				errFile, err = os.OpenFile(token.Value, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error opening stderr file: %v\n", err)
					os.Exit(1)
				}
			}
		}

		execute(args, outFile, errFile)
	}
}
