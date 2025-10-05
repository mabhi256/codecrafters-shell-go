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

		validCommands := []string{"exit", "echo", "type", "pwd"}

		switch args[0] {
		case "exit": // assuming the tester will always pass in 0 as the argument
			os.Exit(0)

		case "echo":
			fmt.Println(input[5:]) // "echo" + " "

		case "type":
			if slices.Contains(validCommands, args[1]) {
				fmt.Println(args[1], "is a shell builtin")
				continue
			}

			execPath, found := getExecPath(args[1])
			if found {
				fmt.Printf("%s is %s\n", args[1], execPath)
			} else {
				fmt.Printf("%s: not found\n", args[1])
			}

		case "pwd":
			pwd, err := os.Getwd()
			if err != nil {
				fmt.Println("Unable to get pwd")
				continue
			}

			fmt.Println(pwd)

		case "cd":
			if len(args) > 1 {
				err := os.Chdir(args[1])
				if err != nil {
					fmt.Printf("cd: %s: No such file or directory\n", args[1])
				}
			}

		default:
			_, found := getExecPath(args[0])
			if found {
				cmd := exec.Command(args[0], args[1:]...)
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				cmd.Run()
			} else {
				fmt.Println(input + ": command not found")
			}
		}
	}
}
