package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/term"
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
			fmt.Printf("Stat error for %s: %s\n", execPath, err.Error())
		}
	}

	return execPath, found
}

func execute(args []string, outFile *os.File, errFile *os.File) {
	command := args[0]

	switch command {
	case "exit": // assuming the tester will always pass in 0 as the argument
		os.Exit(0)

	case "echo":
		fmt.Fprintf(outFile, "%s\n", strings.Join(args[1:], " ")) // "echo" + " "

	case "type":
		if slices.Contains(inbuiltCommands, args[1]) {
			fmt.Fprintf(outFile, "%s is a shell builtin\n", args[1])
			return
		}

		execPath, found := getExecPath(args[1])
		if found {
			fmt.Fprintf(outFile, "%s is %s\n", args[1], execPath)
		} else {
			fmt.Fprintf(outFile, "%s: not found\r\n", args[1])
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
				fmt.Fprintf(errFile, "cd: %s: No such file or directory\r\n", args[1])
			}
		} else {
			home := os.Getenv("HOME")
			os.Chdir(home)
		}

	default:
		// can use exec.LookPath(command) instead of custom getExecPath(command)
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
			fmt.Print("\r")
		} else {
			fmt.Fprintf(outFile, "%s: command not found\n", command)
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	contCmd := ""
	for {
		// switch stdin to 'raw' mode for readline()
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(os.Stdout, "$ %s", contCmd)

		// Wait for user input
		input, shouldContinue, err := readline(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		if shouldContinue {
			contCmd = input
			continue
		}

		// Resume 'cooked' mode for executing commands
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println()
		// fmt.Printf("len: %d, input: %s\n", len(input), input)

		tokens := tokenize(input)
		// fmt.Fprintf(os.Stderr, "tokens: len(%d) %v\n", len(tokens), tokens)

		outFile := os.Stdout
		errFile := os.Stderr
		args := []string{}
		var prev Token
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
						os.Exit(1)
					}
				} else {
					args = append(args, token.Value)
				}
			case Redirect:
			}
			prev = token
		}

		execute(args, outFile, errFile)

		// Close files immediately after use
		if outFile != os.Stdout {
			outFile.Close()
		}
		if errFile != os.Stderr {
			errFile.Close()
		}
	}
}
